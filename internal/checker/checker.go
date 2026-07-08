package checker

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type CheckConfig struct {
	Timeout  int
	Port     int
	MMDBPath string
}

type CheckResult struct {
	Domain        string        `json:"domain"`
	IP            string        `json:"ip"`
	Error         string        `json:"error,omitempty"`
	Accessible    bool          `json:"accessible"`
	StatusCode    int           `json:"status_code"`
	ResponseTime  time.Duration `json:"response_time"`
	FinalDomain   string        `json:"final_domain"`
	RedirectCount int           `json:"redirect_count"`
	TLSVersion    string        `json:"tls_version"`
	SupportsTLS13 bool          `json:"supports_tls13"`
	SupportsH2    bool          `json:"supports_h2"`
	HandshakeTime time.Duration `json:"handshake_time"`
	CertValid     bool          `json:"cert_valid"`
	CertDomain    string        `json:"cert_domain"`
	CertIssuer    string        `json:"cert_issuer"`
	CertDaysLeft  int           `json:"cert_days_left"`
	SNIMatch      bool          `json:"sni_match"`
	IsCDN         bool          `json:"is_cdn"`
	CDNProvider   string        `json:"cdn_provider"`
	CDNConfidence string        `json:"cdn_confidence"`
	IsHotWebsite  bool          `json:"is_hot_website"`
	IsBlocked     bool          `json:"is_blocked"`
	BlockedReason string        `json:"blocked_reason"`
	Country       string        `json:"country"`
	Suitable      bool          `json:"suitable"`
	Score         int           `json:"score"`
	Recommendation string       `json:"recommendation"`
}

type Checker struct {
	config       CheckConfig
	httpClient   *http.Client
	cdnDetector  *CDNDetector
	blockDetector *BlockDetector
	hotWebsites  map[string]bool
}

func NewChecker(config CheckConfig) *Checker {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}
	return &Checker{
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Duration(config.Timeout) * time.Second,
			Transport: tr,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		cdnDetector:   NewCDNDetector(),
		blockDetector: NewBlockDetector(),
		hotWebsites:   loadHotWebsites(),
	}
}

func (c *Checker) Check(domain string) *CheckResult {
	r := &CheckResult{Domain: domain}

	tlsResult := c.checkTLS(domain)
	if tlsResult == nil {
		r.Error = "DNS resolution or TLS handshake failed"
		r.Suitable = false
		return r
	}
	r.IP = tlsResult.ip
	r.TLSVersion = tlsResult.version
	r.SupportsTLS13 = tlsResult.supportsTLS13
	r.SupportsH2 = tlsResult.supportsH2
	r.HandshakeTime = tlsResult.handshakeTime
	r.CertDomain = tlsResult.certDomain
	r.CertIssuer = tlsResult.certIssuer
	r.CertDaysLeft = tlsResult.certDaysLeft
	r.CertValid = tlsResult.certValid
	r.SNIMatch = tlsResult.sniMatch

	httpResult := c.checkHTTP(domain)
	r.Accessible = httpResult.accessible
	r.StatusCode = httpResult.statusCode
	r.ResponseTime = httpResult.responseTime
	r.FinalDomain = httpResult.finalDomain
	r.RedirectCount = httpResult.redirectCount

	cdnResult := c.cdnDetector.Detect(domain, r.IP)
	if !cdnResult.isCDN && httpResult.accessible {
		cdnResult = c.cdnDetector.DetectFromHeaders(httpResult.headers)
	}
	if !cdnResult.isCDN && r.CertIssuer != "" {
		cdnResult = c.cdnDetector.DetectFromCert(r.CertIssuer)
	}
	r.IsCDN = cdnResult.isCDN
	r.CDNProvider = cdnResult.provider
	r.CDNConfidence = cdnResult.confidence
	if r.CDNConfidence == "" {
		r.CDNConfidence = "-"
	}

	r.IsHotWebsite = c.hotWebsites[domain] || c.hotWebsites[normalizeDomain(domain)]

	blockResult := c.blockDetector.Check(domain)
	r.IsBlocked = blockResult.isBlocked
	r.BlockedReason = blockResult.reason

	r.Country = "N/A"
	r.Score = c.calculateScore(r)
	r.Suitable = r.Score >= 3
	r.Recommendation = scoreToLabel(r.Score)
	return r
}

type tlsCheckResult struct {
	ip            string
	version       string
	supportsTLS13 bool
	supportsH2    bool
	handshakeTime time.Duration
	certDomain    string
	certIssuer    string
	certDaysLeft  int
	certValid     bool
	sniMatch      bool
}

func (c *Checker) checkTLS(domain string) *tlsCheckResult {
	ips, err := net.LookupIP(domain)
	if err != nil || len(ips) == 0 {
		return nil
	}
	ip := ips[0].String()
	port := "443"
	if c.config.Port > 0 { port = fmt.Sprint(c.config.Port) }
	addr := net.JoinHostPort(ip, port)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, time.Duration(c.config.Timeout)*time.Second)
	if err != nil {
		return nil
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Duration(c.config.Timeout) * time.Second))
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         domain,
		NextProtos:         []string{"h2", "http/1.1"},
		CurvePreferences:   []tls.CurveID{tls.X25519},
	}
	tlsConn := tls.Client(conn, tlsCfg)
	err = tlsConn.Handshake()
	hsTime := time.Since(start)
	if err != nil {
		return nil
	}
	state := tlsConn.ConnectionState()
	r := &tlsCheckResult{
		ip: ip, version: tls.VersionName(state.Version),
		handshakeTime: hsTime,
		supportsTLS13: state.Version == tls.VersionTLS13,
		supportsH2:    state.NegotiatedProtocol == "h2",
	}
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		r.certDomain = cert.Subject.CommonName
		r.certIssuer = strings.Join(cert.Issuer.Organization, " | ")
		r.certDaysLeft = int(cert.NotAfter.Sub(time.Now()).Hours() / 24)
		r.certValid = time.Now().Before(cert.NotAfter) && time.Now().After(cert.NotBefore)
		for _, name := range cert.DNSNames {
			if name == domain || strings.HasPrefix(name, "*.") {
				r.sniMatch = true
				break
			}
		}
	}
	return r
}

type httpCheckResult struct {
	accessible    bool
	statusCode    int
	responseTime  time.Duration
	finalDomain   string
	redirectCount int
	headers       map[string]string
}

func (c *Checker) checkHTTP(domain string) *httpCheckResult {
	r := &httpCheckResult{headers: make(map[string]string)}
	start := time.Now()
	req, _ := http.NewRequest("GET", "https://"+domain, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := c.httpClient.Do(req)
	r.responseTime = time.Since(start)
	if err != nil {
		return r
	}
	defer resp.Body.Close()
	r.accessible = true
	r.statusCode = resp.StatusCode
	r.finalDomain = resp.Request.URL.Host
	for k, v := range resp.Header {
		r.headers[strings.ToLower(k)] = strings.Join(v, ", ")
	}
	return r
}

func (c *Checker) calculateScore(r *CheckResult) int {
	score := 0
	if r.SupportsTLS13 {
		score += 2
	}
	if r.SupportsH2 {
		score += 1
	}
	if r.SNIMatch {
		score += 1
	}
	if r.CertValid && r.CertDaysLeft > 30 {
		score += 1
	}
	if r.Accessible {
		switch r.StatusCode {
		case 200, 301, 302, 404:
			score += 2
		}
	}
	switch r.CDNConfidence {
	case "-":
		score += 2
	case "低":
		score += 1
	case "中":
		score -= 1
	case "高":
		score -= 2
	}
	if r.IsHotWebsite {
		score -= 1
	}
	if r.IsBlocked {
		score -= 3
	}
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}
	return score
}

func scoreToLabel(score int) string {
	switch {
	case score >= 9:
		return "强烈推荐"
	case score >= 7:
		return "推荐"
	case score >= 5:
		return "可用"
	case score >= 3:
		return "勉强可用"
	default:
		return "不推荐"
	}
}

func (r *CheckResult) ScoreStars() string {
	switch {
	case r.Score >= 9:
		return "⭐⭐⭐⭐⭐"
	case r.Score >= 7:
		return "⭐⭐⭐⭐"
	case r.Score >= 5:
		return "⭐⭐⭐"
	case r.Score >= 3:
		return "⭐⭐"
	default:
		return "⭐"
	}
}

func normalizeDomain(domain string) string {
	return strings.TrimPrefix(strings.TrimPrefix(domain, "www."), "www2.")
}

var hotWebsitesList = []string{
	"google.com", "youtube.com", "facebook.com", "instagram.com",
	"twitter.com", "x.com", "tiktok.com", "whatsapp.com",
	"apple.com", "icloud.com", "microsoft.com", "azure.com",
	"amazon.com", "aws.amazon.com", "cloudflare.com", "netflix.com",
	"spotify.com", "telegram.org", "discord.com", "reddit.com",
	"linkedin.com", "github.com", "gitlab.com", "stackoverflow.com",
	"wikipedia.org", "yahoo.com", "bing.com", "baidu.com",
	"wechat.com", "qq.com", "alibaba.com", "taobao.com",
	"tesla.com", "nvidia.com", "adobe.com", "oracle.com",
	"salesforce.com", "zoom.us", "office.com", "live.com",
}

func loadHotWebsites() map[string]bool {
	m := make(map[string]bool, len(hotWebsitesList))
	for _, h := range hotWebsitesList {
		m[h] = true
		m["www."+h] = true
	}
	return m
}

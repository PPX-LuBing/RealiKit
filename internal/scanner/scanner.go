package scanner

import (
	"crypto/tls"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ScannerConfig struct {
	Port        int
	Threads     int
	Timeout     int
	MaxTargets  int
	EnableIPv6  bool
	Verbose     bool
	MMDBPath    string
}

type Scanner struct {
	config ScannerConfig
	geo    *Geo
}

func NewScanner(config ScannerConfig) *Scanner {
	return &Scanner{
		config: config,
		geo:    NewGeo(config.MMDBPath),
	}
}

func (s *Scanner) Close() {
	s.geo.Close()
}

func (s *Scanner) Scan(hosts <-chan Host, results chan<- ScanResult, done func()) {
	var wg sync.WaitGroup
	hostsCh := make(chan Host)
	limit := s.config.MaxTargets

	go func() {
		count := 0
		for host := range hosts {
			if limit > 0 && count >= limit {
				break
			}
			hostsCh <- host
			count++
		}
		close(hostsCh)
	}()

	wg.Add(s.config.Threads)
	for i := 0; i < s.config.Threads; i++ {
		go func() {
			for host := range hostsCh {
				result := s.scanOne(host)
				results <- result
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if done != nil {
		done()
	}
}

func (s *Scanner) scanOne(host Host) ScanResult {
	result := ScanResult{
		Origin: host.Origin,
	}
	if host.IP == nil {
		ip, err := LookupIP(host.Origin, s.config.EnableIPv6)
		if err != nil {
			slog.Debug("Failed to resolve", "origin", host.Origin, "err", err)
			result.Feasible = false
			return result
		}
		host.IP = ip
	}
	result.IP = host.IP.String()
	hostPort := net.JoinHostPort(host.IP.String(), strconv.Itoa(s.config.Port))

	start := time.Now()
	conn, err := net.DialTimeout("tcp", hostPort, time.Duration(s.config.Timeout)*time.Second)
	if err != nil {
		slog.Debug("Cannot dial", "target", hostPort)
		result.Feasible = false
		return result
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(time.Duration(s.config.Timeout) * time.Second))

	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2", "http/1.1"},
		CurvePreferences:   []tls.CurveID{tls.X25519},
	}
	if host.Type == HostTypeDomain {
		tlsCfg.ServerName = host.Origin
	}

	c := tls.Client(conn, tlsCfg)
	err = c.Handshake()
	result.HandshakeTime = time.Since(start)
	if err != nil {
		slog.Debug("TLS handshake failed", "target", hostPort)
		result.Feasible = false
		return result
	}

	state := c.ConnectionState()
	result.TLSVersion = tls.VersionName(state.Version)
	result.ALPN = state.NegotiatedProtocol
	result.Curve = tls.CurveID(state.CipherSuite).String()
	result.ConnectionState = &state

	if len(state.PeerCertificates) > 0 {
		result.CertDomain = state.PeerCertificates[0].Subject.CommonName
		result.CertIssuer = strings.Join(state.PeerCertificates[0].Issuer.Organization, " | ")
		leaf := state.PeerCertificates[0]
		for _, cert := range state.PeerCertificates {
			result.CertLength += len(cert.Raw)
			if len(cert.DNSNames) != 0 {
				leaf = cert
			}
		}
		result.CertCount = len(state.PeerCertificates)
		result.CertSignature = leaf.SignatureAlgorithm.String()
		result.CertPubKey = leaf.PublicKeyAlgorithm.String()
	}

	result.GeoCode = s.geo.GetCode(host.IP)

	if state.Version == tls.VersionTLS13 && result.ALPN == "h2" &&
		result.CertDomain != "" && result.CertIssuer != "" {
		result.Feasible = true
	} else {
		result.Feasible = false
	}

	return result
}

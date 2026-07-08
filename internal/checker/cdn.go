package checker

import (
	"strings"
)

type cdnResult struct {
	isCDN      bool
	provider   string
	confidence string
}

type CDNDetector struct {
	cnameSuffixes   map[string]string
	headerValues    map[string]string
	headerKeywords  map[string]string
	asnKeywords     map[string]string
	certIssuers     map[string]string
}

func NewCDNDetector() *CDNDetector {
	return &CDNDetector{
		cnameSuffixes: map[string]string{
			".cloudflare.net":        "Cloudflare",
			".cloudflare.com":        "Cloudflare",
			".akamaiedge.net":        "Akamai",
			".akamaitechnologies.com": "Akamai",
			".edgesuite.net":         "Akamai",
			".edgekey.net":           "Akamai",
			".fastly.net":            "Fastly",
			".fastlylb.net":          "Fastly",
			".cloudfront.net":        "AWS CloudFront",
			".cdn.cloudflare.net":    "Cloudflare",
			".trafficmanager.net":    "Azure",
			".azureedge.net":         "Azure",
			".azurefd.net":           "Azure Front Door",
			".fbcdn.net":             "Facebook CDN",
			"ggpht.com":              "Google CDN",
			".googleusercontent.com": "Google CDN",
			".kxcdn.com":             "KeyCDN",
			".stackpathdns.com":      "StackPath",
			".stackpathcdn.com":      "StackPath",
			".rlcdn.com":             "ReapCDN",
			".cachefly.net":          "CacheFly",
			".panthercdn.com":        "PantherCDN",
			".cdnga.net":             "CDNGA",
			".bitgravity.com":        "BitGravity",
			".cdn77.net":             "CDN77",
			".cdn77.org":             "CDN77",
			".b-cdn.net":             "BunnyCDN",
			".singularcdn.net":       "SingularCDN",
			".csgo.com":              "CSGO CDN",
		},
		headerValues: map[string]string{
			"cf-ray":              "Cloudflare",
			"server: cloudflare":  "Cloudflare",
			"x-amz-cf-id":        "AWS CloudFront",
			"x-amz-cf-pop":       "AWS CloudFront",
			"x-amz-req-id":       "AWS CloudFront",
			"x-cache":            "CDN",
			"x-iinfo":            "Akamai",
			"akamai-x-cache-on":  "Akamai",
			"x-akamai-":          "Akamai",
			"x-servedby":         "CDN",
			"x-cdn":              "CDN",
			"x-edge":             "EdgeCast",
			"fastly-":            "Fastly",
			"x-sucuri-id":        "Sucuri",
			"x-sucuri-cache":     "Sucuri",
			"x-encoded-content-": "CDN",
		},
		headerKeywords: map[string]string{
			"cloudflare": "Cloudflare",
			"akamai":     "Akamai",
			"fastly":     "Fastly",
			"cloudfront": "AWS CloudFront",
			"azure":      "Azure",
			"stackpath":  "StackPath",
			"bunnycdn":   "BunnyCDN",
			"keycdn":     "KeyCDN",
			"cdn77":      "CDN77",
		},
		certIssuers: map[string]string{
			"Cloudflare":    "Cloudflare",
			"Fastly":        "Fastly",
			"Akamai":        "Akamai",
			"Azure":         "Azure",
		},
	}
}

func (d *CDNDetector) Detect(domain, ip string) *cdnResult {
	r := &cdnResult{}

	if provider, ok := d.checkCNAME(domain); ok {
		r.isCDN = true
		r.provider = provider
		r.confidence = "高"
		return r
	}

	r.isCDN = false
	return r
}

func (d *CDNDetector) DetectFromHeaders(headers map[string]string) *cdnResult {
	r := &cdnResult{}

	for key, value := range headers {
		combined := strings.ToLower(key + ": " + value)
		for pattern, provider := range d.headerValues {
			if strings.Contains(combined, pattern) {
				r.isCDN = true
				r.provider = provider
				r.confidence = "高"
				return r
			}
		}
		for keyword, provider := range d.headerKeywords {
			if strings.Contains(value, keyword) {
				r.isCDN = true
				r.provider = provider
				r.confidence = "中"
				return r
			}
		}
	}

	return r
}

func (d *CDNDetector) DetectFromCert(issuer string) *cdnResult {
	r := &cdnResult{}
	for name, provider := range d.certIssuers {
		if strings.Contains(strings.ToLower(issuer), strings.ToLower(name)) {
			r.isCDN = true
			r.provider = provider
			r.confidence = "低"
			return r
		}
	}
	return r
}

func (d *CDNDetector) checkCNAME(domain string) (string, bool) {
	// In a real implementation we'd do DNS CNAME lookup
	return "", false
}

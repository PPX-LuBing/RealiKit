package scanner

import (
	"crypto/tls"
	"net"
	"time"
)

type HostType int

const (
	HostTypeIP HostType = iota + 1
	HostTypeCIDR
	HostTypeDomain
)

type Host struct {
	IP     net.IP
	Origin string
	Type   HostType
}

type ScanResult struct {
	IP              string        `json:"ip"`
	Origin          string        `json:"origin"`
	TLSVersion      string        `json:"tls_version"`
	ALPN            string        `json:"alpn"`
	Curve           string        `json:"curve"`
	CertDomain      string        `json:"cert_domain"`
	CertIssuer      string        `json:"cert_issuer"`
	CertSignature   string        `json:"cert_signature"`
	CertPubKey      string        `json:"cert_pubkey"`
	CertLength      int           `json:"cert_length"`
	CertCount       int           `json:"cert_count"`
	GeoCode         string        `json:"geo_code"`
	Feasible        bool          `json:"feasible"`
	HandshakeTime   time.Duration `json:"handshake_time"`
	ConnectionState *tls.ConnectionState
}

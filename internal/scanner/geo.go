package scanner

import (
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

type Geo struct {
	reader *geoip2.Reader
	mu     sync.Mutex
	enabled bool
}

func NewGeo(mmdbPath string) *Geo {
	g := &Geo{}
	reader, err := geoip2.Open(mmdbPath)
	if err != nil {
		return g
	}
	g.reader = reader
	g.enabled = true
	return g
}

func (g *Geo) GetCode(ip net.IP) string {
	if !g.enabled || g.reader == nil {
		return "N/A"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	country, err := g.reader.Country(ip)
	if err != nil {
		return "N/A"
	}
	return country.Country.IsoCode
}

func (g *Geo) Close() {
	if g.reader != nil {
		g.reader.Close()
	}
}

package scanner

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
)

func ValidateDomainName(domain string) bool {
	r := regexp.MustCompile(`(?m)^[A-Za-z0-9\-.]+$`)
	return r.MatchString(domain)
}

func RemoveDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	var list []string
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func LookupIP(addr string, enableIPv6 bool) (net.IP, error) {
	ips, err := net.LookupIP(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup: %w", err)
	}
	var arr []net.IP
	for _, ip := range ips {
		if ip.To4() != nil || enableIPv6 {
			arr = append(arr, ip)
		}
	}
	if len(arr) == 0 {
		return nil, errors.New("no IP found")
	}
	return arr[0], nil
}

func ParseTargets(reader io.Reader, enableIPv6 bool) <-chan Host {
	sc := bufio.NewScanner(reader)
	hostChan := make(chan Host)
	go func() {
		defer close(hostChan)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			ip := net.ParseIP(line)
			if ip != nil && (ip.To4() != nil || enableIPv6) {
				hostChan <- Host{IP: ip, Origin: line, Type: HostTypeIP}
				continue
			}
			_, _, err := net.ParseCIDR(line)
			if err == nil {
				p, err := netip.ParsePrefix(line)
				if err != nil {
					continue
				}
				if !p.Addr().Is4() && !enableIPv6 {
					continue
				}
				p = p.Masked()
				addr := p.Addr()
				for {
					if !p.Contains(addr) {
						break
					}
					ip = net.ParseIP(addr.String())
					if ip != nil {
						hostChan <- Host{IP: ip, Origin: line, Type: HostTypeCIDR}
					}
					addr = addr.Next()
				}
				continue
			}
			if ValidateDomainName(line) {
				hostChan <- Host{IP: nil, Origin: line, Type: HostTypeDomain}
				continue
			}
		}
	}()
	return hostChan
}

func CIDRSize(addr string) int {
	_, ipnet, err := net.ParseCIDR(addr)
	if err != nil {
		return 0
	}
	ones, bits := ipnet.Mask.Size()
	if bits == 32 {
		return 1 << (bits - ones)
	}
	return 0
}

func ExpandAddr(addr string, enableIPv6 bool) <-chan Host {
	hostChan := make(chan Host)
	_, _, err := net.ParseCIDR(addr)
	if err == nil {
		return ParseTargets(strings.NewReader(addr), enableIPv6)
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		resolved, err := LookupIP(addr, enableIPv6)
		if err != nil {
			close(hostChan)
			return hostChan
		}
		ip = resolved
	}
	go func() {
		hostChan <- Host{IP: ip, Origin: addr, Type: HostTypeIP}
		close(hostChan)
	}()
	return hostChan
}

func ExtractDomainsFromURL(rawURL string) ([]string, error) {
	c := &http.Client{Timeout: 15e9}
	resp, err := c.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch URL failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}
	re := regexp.MustCompile(`(http|https)://(.*?)[/"<>\s]+`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	var domains []string
	for _, m := range matches {
		domains = append(domains, m[2])
	}
	return RemoveDuplicateStr(domains), nil
}

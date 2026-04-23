package resolver

import (
	"net"
	"strings"

	"github.com/miekg/dns"
)

var privateBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"127.0.0.0/8", "169.254.0.0/16", "100.64.0.0/10",
		"::1/128", "fc00::/7", "fe80::/10",
	} {
		_, block, _ := net.ParseCIDR(cidr)
		if block != nil {
			privateBlocks = append(privateBlocks, block)
		}
	}
}

func isPrivate(ip net.IP) bool {
	for _, b := range privateBlocks {
		if b.Contains(ip) {
			return true
		}
	}
	return false
}

func (r *Resolver) isRebinding(name string, resp *dns.Msg) bool {
	if isLocalDomain(name) {
		return false
	}
	for _, rr := range resp.Answer {
		switch v := rr.(type) {
		case *dns.A:
			if isPrivate(v.A) {
				return true
			}
		case *dns.AAAA:
			if isPrivate(v.AAAA) {
				return true
			}
		}
	}
	return false
}

func isLocalDomain(name string) bool {
	d := strings.ToLower(strings.TrimSuffix(name, "."))
	for _, s := range []string{".local", ".localhost", ".internal", ".home", ".lan", ".localdomain"} {
		if strings.HasSuffix(d, s) || d == strings.TrimPrefix(s, ".") {
			return true
		}
	}
	return false
}

func nxDomain(req *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(req, dns.RcodeNameError)
	return m
}

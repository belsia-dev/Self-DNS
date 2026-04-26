package dns

import (
	"net"
	"strings"

	"github.com/miekg/dns"
)

const (
	staticHostTTL  uint32 = 300
	blockedHostTTL uint32 = 5
)

func writeRcode(w dns.ResponseWriter, req *dns.Msg, rcode int) {
	_ = w.WriteMsg(rcodeResponse(req, rcode))
}

func rcodeResponse(req *dns.Msg, rcode int) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(req, rcode)
	return m
}

func emptyResponse(req *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(req)
	m.Authoritative = true
	return m
}

func staticHostResponse(req *dns.Msg, ipStr string) *dns.Msg {
	return buildHostResponse(req, ipStr, staticHostTTL)
}

func blockedHostResponse(req *dns.Msg, ipStr string) *dns.Msg {
	return buildHostResponse(req, ipStr, blockedHostTTL)
}

func buildHostResponse(req *dns.Msg, ipStr string, ttl uint32) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(req)
	m.Authoritative = true

	ip := net.ParseIP(ipStr)
	if ip == nil {
		m.Rcode = dns.RcodeNameError
		return m
	}

	q := req.Question[0]
	if ip4 := ip.To4(); ip4 != nil && q.Qtype == dns.TypeA {
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
			A:   ip4,
		})
	} else if q.Qtype == dns.TypeAAAA {
		if ip6 := ip.To16(); ip6 != nil && ip.To4() == nil {
			m.Answer = append(m.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
				AAAA: ip6,
			})
		}
	}
	return m
}

func lookupHost(hosts map[string]string, name string) (string, bool) {
	clean := strings.ToLower(strings.TrimSuffix(name, "."))
	ip, ok := hosts[clean]
	return ip, ok
}

func extractIP(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}

func anonymise(domain string) string {
	parts := strings.SplitN(strings.TrimSuffix(domain, "."), ".", 2)
	if len(parts) == 2 {
		return "*." + parts[1]
	}
	return domain
}

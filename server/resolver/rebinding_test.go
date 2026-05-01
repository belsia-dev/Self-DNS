package resolver

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

// ---------------------------------------------------------------------------
// isPrivate
// ---------------------------------------------------------------------------

func TestIsPrivate_KnownPrivateIPs(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want bool
	}{
		{"RFC1918 10.x", "10.0.0.1", true},
		{"RFC1918 172.16", "172.16.0.1", true},
		{"RFC1918 172.31", "172.31.255.255", true},
		{"RFC1918 192.168", "192.168.1.1", true},
		{"loopback", "127.0.0.1", true},
		{"link-local", "169.254.1.1", true},
		{"CGNAT", "100.64.0.1", true},
		{"IPv6 loopback", "::1", true},
		{"IPv6 ULA fc00", "fc00::1", true},
		{"IPv6 ULA fd00", "fd00::1", true},
		{"IPv6 link-local fe80", "fe80::1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isPrivate(net.ParseIP(tc.ip))
			if got != tc.want {
				t.Errorf("isPrivate(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIsPrivate_PublicIPs(t *testing.T) {
	cases := []struct {
		name string
		ip   string
	}{
		{"Cloudflare", "1.1.1.1"},
		{"Google DNS", "8.8.8.8"},
		{"Quad9", "9.9.9.9"},
		{"random public", "203.0.113.1"},
		{"IPv6 public", "2001:db8::1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if isPrivate(net.ParseIP(tc.ip)) {
				t.Errorf("isPrivate(%s) = true, want false", tc.ip)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isPrivate – edge cases
// ---------------------------------------------------------------------------

func TestIsPrivate_EdgeCases(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want bool
	}{
		// 0.0.0.0 is not in any of the private blocks defined in init()
		{"0.0.0.0", "0.0.0.0", false},
		{"127.0.0.1", "127.0.0.1", true},
		{"127.255.255.255", "127.255.255.255", true},
		{"169.254.169.254 metadata", "169.254.169.254", true},
		{"224.0.0.1 multicast", "224.0.0.1", false},
		{"255.255.255.255 broadcast", "255.255.255.255", false},
		{"172.15.255.255 not private", "172.15.255.255", false},
		{"172.32.0.1 not private", "172.32.0.1", false},
		{"100.63.255.255 below CGNAT", "100.63.255.255", false},
		{"100.128.0.1 above CGNAT", "100.128.0.1", false},
		{"100.127.255.255 CGNAT edge", "100.127.255.255", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isPrivate(net.ParseIP(tc.ip))
			if got != tc.want {
				t.Errorf("isPrivate(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isLocalDomain
// ---------------------------------------------------------------------------

func TestIsLocalDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   bool
	}{
		{"local", "myhost.local", true},
		{"localhost", "myhost.localhost", true},
		{"internal", "svc.internal", true},
		{"home", "router.home", true},
		{"lan", "nas.lan", true},
		{"localdomain", "pc.localdomain", true},
		{"bare local", "local", true},
		{"bare localhost", "localhost", true},
		{"bare internal", "internal", true},
		{"trailing dot", "myhost.local.", true},
		{"public domain", "example.com", false},
		{"public subdomain", "api.example.com", false},
		{"com.local is not .local suffix", "comlocal", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isLocalDomain(tc.domain)
			if got != tc.want {
				t.Errorf("isLocalDomain(%q) = %v, want %v", tc.domain, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// isRebinding – full method test
// ---------------------------------------------------------------------------

func TestIsRebinding_PrivateARecord(t *testing.T) {
	r := &Resolver{}
	msg := new(dns.Msg)
	rr, _ := dns.NewRR("evil.com. 60 IN A 192.168.1.1")
	msg.Answer = append(msg.Answer, rr)
	if !r.isRebinding("evil.com.", msg) {
		t.Error("expected rebinding detection for private A record")
	}
}

func TestIsRebinding_PublicARecord(t *testing.T) {
	r := &Resolver{}
	msg := new(dns.Msg)
	rr, _ := dns.NewRR("example.com. 60 IN A 93.184.216.34")
	msg.Answer = append(msg.Answer, rr)
	if r.isRebinding("example.com.", msg) {
		t.Error("public IP should not be flagged as rebinding")
	}
}

func TestIsRebinding_LocalDomainAllowed(t *testing.T) {
	r := &Resolver{}
	msg := new(dns.Msg)
	rr, _ := dns.NewRR("myhost.local. 60 IN A 192.168.1.1")
	msg.Answer = append(msg.Answer, rr)
	if r.isRebinding("myhost.local.", msg) {
		t.Error("local domains should bypass rebinding protection")
	}
}

func TestIsRebinding_IPv6Private(t *testing.T) {
	r := &Resolver{}
	msg := new(dns.Msg)
	rr, _ := dns.NewRR("evil.com. 60 IN AAAA ::1")
	msg.Answer = append(msg.Answer, rr)
	if !r.isRebinding("evil.com.", msg) {
		t.Error("expected rebinding detection for IPv6 loopback")
	}
}

func TestIsRebinding_IPv6Public(t *testing.T) {
	r := &Resolver{}
	msg := new(dns.Msg)
	rr, _ := dns.NewRR("example.com. 60 IN AAAA 2606:4700:4700::1111")
	msg.Answer = append(msg.Answer, rr)
	if r.isRebinding("example.com.", msg) {
		t.Error("public IPv6 should not be flagged")
	}
}

func TestIsRebinding_MixedAnswers(t *testing.T) {
	r := &Resolver{}
	msg := new(dns.Msg)
	rr1, _ := dns.NewRR("evil.com. 60 IN A 93.184.216.34")
	rr2, _ := dns.NewRR("evil.com. 60 IN A 10.0.0.1")
	msg.Answer = append(msg.Answer, rr1, rr2)
	if !r.isRebinding("evil.com.", msg) {
		t.Error("mixed answers with one private IP should be flagged")
	}
}

func TestIsRebinding_NoAnswers(t *testing.T) {
	r := &Resolver{}
	msg := new(dns.Msg)
	if r.isRebinding("example.com.", msg) {
		t.Error("empty answers should not be flagged")
	}
}

func TestIsRebinding_LocalhostDomainVariants(t *testing.T) {
	r := &Resolver{}
	for _, name := range []string{"test.localhost.", "test.home.", "test.lan.", "test.localdomain."} {
		msg := new(dns.Msg)
		rr, _ := dns.NewRR(name + " 60 IN A 127.0.0.1")
		msg.Answer = append(msg.Answer, rr)
		if r.isRebinding(name, msg) {
			t.Errorf("local domain %q should bypass rebinding protection", name)
		}
	}
}

// ---------------------------------------------------------------------------
// nxDomain
// ---------------------------------------------------------------------------

func TestNxDomain(t *testing.T) {
	req := new(dns.Msg)
	req.SetQuestion("evil.com.", dns.TypeA)
	resp := nxDomain(req)
	if resp.Rcode != dns.RcodeNameError {
		t.Errorf("nxDomain rcode = %d, want %d", resp.Rcode, dns.RcodeNameError)
	}
}

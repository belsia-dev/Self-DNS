package blocker

import (
	"testing"

	"github.com/belsia-dev/Self-DNS/server/config"
)

func TestIsBlockedMatchesSubdomains(t *testing.T) {
	b := New(config.BlocklistConfig{
		Enabled: true,
		Domains: []string{"example.com"},
	})

	if !b.IsBlocked("example.com.") {
		t.Fatalf("expected exact domain to be blocked")
	}
	if !b.IsBlocked("www.example.com.") {
		t.Fatalf("expected subdomain to be blocked")
	}
	if b.IsBlocked("badexample.com.") {
		t.Fatalf("did not expect sibling domain to be blocked")
	}
}

package config

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// validConfig returns a minimal valid config suitable for modification in tests.
func validConfig() *Config {
	cfg := defaultConfig()
	cfg.Upstream = []string{"1.1.1.1:853", "8.8.8.8:853"}
	return cfg
}

// ---------------------------------------------------------------------------
// 1. TestDefaultConfig
// ---------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if !strings.HasPrefix(cfg.Listen, "127.0.0.1") {
		t.Errorf("default Listen = %q, want loopback", cfg.Listen)
	}
	if !strings.HasPrefix(cfg.APIListen, "127.0.0.1") {
		t.Errorf("default APIListen = %q, want loopback", cfg.APIListen)
	}
	if !cfg.UseTLS {
		t.Error("default UseTLS = false, want true")
	}
	if !cfg.DNSRebindingProtection {
		t.Error("default DNSRebindingProtection = false, want true")
	}
	if !cfg.DNSSEC {
		t.Error("default DNSSEC = false, want true")
	}
	if !cfg.RateLimit.Enabled {
		t.Error("default RateLimit.Enabled = false, want true")
	}
	if len(cfg.Upstream) == 0 {
		t.Error("default Upstream is empty, want at least one upstream")
	}
	for i, u := range cfg.Upstream {
		if _, _, err := net.SplitHostPort(u); err != nil {
			t.Errorf("default Upstream[%d] = %q invalid: %v", i, u, err)
		}
	}
	if cfg.ServiceMode != "local" {
		t.Errorf("default ServiceMode = %q, want %q", cfg.ServiceMode, "local")
	}
	if !cfg.Cache.Enabled {
		t.Error("default Cache.Enabled = false, want true")
	}
	if cfg.Cache.MaxSize <= 0 {
		t.Errorf("default Cache.MaxSize = %d, want > 0", cfg.Cache.MaxSize)
	}
}

// ---------------------------------------------------------------------------
// 2. TestValidateEmptyUpstreams
// ---------------------------------------------------------------------------

func TestValidateEmptyUpstreams(t *testing.T) {
	cfg := validConfig()
	cfg.Upstream = nil

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for empty upstreams, got nil")
	}
	if !strings.Contains(err.Error(), "at least one upstream") {
		t.Errorf("error = %q, want mention of upstream", err)
	}
}

// ---------------------------------------------------------------------------
// 3. TestValidateBadUpstreamFormat
// ---------------------------------------------------------------------------

func TestValidateBadUpstreamFormat(t *testing.T) {
	cfg := validConfig()
	cfg.Upstream = []string{"not-valid"}

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid upstream, got nil")
	}
	if !strings.Contains(err.Error(), "upstream[0]") {
		t.Errorf("error = %q, want upstream index hint", err)
	}
	if !strings.Contains(err.Error(), "not-valid") {
		t.Errorf("error = %q, want upstream value in message", err)
	}
}

// ---------------------------------------------------------------------------
// 4. TestValidateGoodUpstreams
// ---------------------------------------------------------------------------

func TestValidateGoodUpstreams(t *testing.T) {
	cfg := validConfig()
	cfg.Upstream = []string{"1.1.1.1:853", "8.8.8.8:853"}

	err := validate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 5. TestValidateBadHostIP
// ---------------------------------------------------------------------------

func TestValidateBadHostIP(t *testing.T) {
	cfg := validConfig()
	cfg.Hosts = map[string]string{
		"bad.local": "not-an-ip",
	}

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid host IP, got nil")
	}
	if !strings.Contains(err.Error(), "bad.local") {
		t.Errorf("error = %q, want domain name in message", err)
	}
	if !strings.Contains(err.Error(), "invalid IP") {
		t.Errorf("error = %q, want 'invalid IP' in message", err)
	}
}

// ---------------------------------------------------------------------------
// 6. TestValidateGoodHosts
// ---------------------------------------------------------------------------

func TestValidateGoodHosts(t *testing.T) {
	cfg := validConfig()
	cfg.Hosts = map[string]string{
		"myapp.local": "127.0.0.1",
		"dual.local":  "::1",
	}

	err := validate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 7. TestValidateWhitelistIPs
// ---------------------------------------------------------------------------

func TestValidateWhitelistIPs(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cfg := validConfig()
		cfg.RateLimit.WhitelistIPs = []string{"10.0.0.1", "192.168.1.0/24", "::1", "fd00::/8", ""}

		if err := validate(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		cfg := validConfig()
		cfg.RateLimit.WhitelistIPs = []string{"not-an-ip"}

		err := validate(cfg)
		if err == nil {
			t.Fatal("expected error for invalid whitelist IP, got nil")
		}
		if !strings.Contains(err.Error(), "whitelist_ips") {
			t.Errorf("error = %q, want 'whitelist_ips' in message", err)
		}
	})
}

// ---------------------------------------------------------------------------
// 8. TestClone
// ---------------------------------------------------------------------------

func TestClone(t *testing.T) {
	orig := validConfig()
	orig.Hosts = map[string]string{
		"app.local": "10.0.0.1",
	}

	clone := Clone(orig)

	clone.Upstream[0] = "9.9.9.9:853"
	clone.RateLimit.WhitelistIPs = append(clone.RateLimit.WhitelistIPs, "10.0.0.0/8")
	clone.Hosts["evil.local"] = "1.2.3.4"
	clone.Listen = "0.0.0.0:53"

	if orig.Upstream[0] != "1.1.1.1:853" {
		t.Errorf("original Upstream[0] = %q, want %q (slice not deep-copied)", orig.Upstream[0], "1.1.1.1:853")
	}
	if orig.Listen != "127.0.0.1:53" {
		t.Errorf("original Listen = %q, want %q", orig.Listen, "127.0.0.1:53")
	}
	if _, ok := orig.Hosts["evil.local"]; ok {
		t.Error("original Hosts contains key added to clone (map not deep-copied)")
	}
	if len(orig.RateLimit.WhitelistIPs) != 0 {
		t.Errorf("original WhitelistIPs = %v, want empty", orig.RateLimit.WhitelistIPs)
	}
}

func TestCloneNil(t *testing.T) {
	if result := Clone(nil); result != nil {
		t.Errorf("Clone(nil) = %v, want nil", result)
	}
}

// ---------------------------------------------------------------------------
// 9. TestPrepareLoopbackEnforcement
// ---------------------------------------------------------------------------

func TestPrepareLoopbackEnforcement(t *testing.T) {
	cfg := validConfig()
	cfg.ServiceMode = "local"
	cfg.APIListen = "0.0.0.0:5380"

	if err := Prepare(cfg); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}

	host, _, _ := net.SplitHostPort(cfg.APIListen)
	if host != "127.0.0.1" {
		t.Errorf("APIListen host = %q in local mode, want 127.0.0.1", host)
	}
}

// ---------------------------------------------------------------------------
// 10. TestPrepareDNSListenNormalisation
// ---------------------------------------------------------------------------

func TestPrepareDNSListenNormalisation(t *testing.T) {
	tests := []struct {
		name        string
		serviceMode string
		input       string
		wantHost    string
	}{
		{"local mode forces loopback", "local", "0.0.0.0:53", "127.0.0.1"},
		{"external mode uses wildcard", "external", "127.0.0.1:5353", "0.0.0.0"},
		{"internal mode uses wildcard", "internal", "127.0.0.1:53", "0.0.0.0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.ServiceMode = tc.serviceMode
			cfg.Listen = tc.input

			if err := Prepare(cfg); err != nil {
				t.Fatalf("Prepare failed: %v", err)
			}

			host, port, err := net.SplitHostPort(cfg.Listen)
			if err != nil {
				t.Fatalf("SplitHostPort(%q): %v", cfg.Listen, err)
			}
			if host != tc.wantHost {
				t.Errorf("host = %q, want %q", host, tc.wantHost)
			}
			if port == "" {
				t.Error("port is empty after normalisation")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 11. TestLoadAndSave
// ---------------------------------------------------------------------------

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-config.yaml")

	original := validConfig()
	original.LogQueries = true
	original.DNSSEC = false
	original.Hosts = map[string]string{
		"custom.local": "192.168.1.100",
	}
	original.RateLimit.WhitelistIPs = []string{"10.0.0.0/8"}

	if err := Save(original, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("saved config file is empty")
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.LogQueries != original.LogQueries {
		t.Errorf("LogQueries = %v, want %v", loaded.LogQueries, original.LogQueries)
	}
	if loaded.DNSSEC != original.DNSSEC {
		t.Errorf("DNSSEC = %v, want %v", loaded.DNSSEC, original.DNSSEC)
	}
	if loaded.UseTLS != original.UseTLS {
		t.Errorf("UseTLS = %v, want %v", loaded.UseTLS, original.UseTLS)
	}
	if len(loaded.Upstream) != len(original.Upstream) {
		t.Errorf("len(Upstream) = %d, want %d", len(loaded.Upstream), len(original.Upstream))
	}
	if loaded.Hosts["custom.local"] != "192.168.1.100" {
		t.Errorf("Hosts[custom.local] = %q, want %q", loaded.Hosts["custom.local"], "192.168.1.100")
	}

	path2 := filepath.Join(dir, "test-config-round2.yaml")
	if err := Save(loaded, path2); err != nil {
		t.Fatalf("Save(2) failed: %v", err)
	}

	loaded2, err := Load(path2)
	if err != nil {
		t.Fatalf("Load(2) failed: %v", err)
	}

	y1, _ := yaml.Marshal(loaded)
	y2, _ := yaml.Marshal(loaded2)
	if string(y1) != string(y2) {
		t.Errorf("config drift after double round-trip:\n--- first\n+++ second\n%s", diffStrings(string(y1), string(y2)))
	}
}

func diffStrings(a, b string) string {
	var out strings.Builder
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	max := len(la)
	if len(lb) > max {
		max = len(lb)
	}
	for i := 0; i < max; i++ {
		sa, sb := "", ""
		if i < len(la) {
			sa = la[i]
		}
		if i < len(lb) {
			sb = lb[i]
		}
		if sa != sb {
			out.WriteString("- ")
			out.WriteString(sa)
			out.WriteString("\n+ ")
			out.WriteString(sb)
			out.WriteString("\n")
		}
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// Additional edge-case tests
// ---------------------------------------------------------------------------

func TestValidateEmptyListen(t *testing.T) {
	cfg := validConfig()
	cfg.Listen = ""

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for empty listen, got nil")
	}
	if !strings.Contains(err.Error(), "listen") {
		t.Errorf("error = %q, want mention of listen", err)
	}
}

func TestValidateEmptyAPIListen(t *testing.T) {
	cfg := validConfig()
	cfg.APIListen = ""

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for empty api_listen, got nil")
	}
	if !strings.Contains(err.Error(), "api_listen") {
		t.Errorf("error = %q, want mention of api_listen", err)
	}
}

func TestValidateBadServiceMode(t *testing.T) {
	cfg := validConfig()
	cfg.ServiceMode = "bogus"

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for bad service_mode, got nil")
	}
	if !strings.Contains(err.Error(), "service_mode") {
		t.Errorf("error = %q, want mention of service_mode", err)
	}
}

func TestValidateHostsEmptyDomain(t *testing.T) {
	cfg := validConfig()
	cfg.Hosts = map[string]string{" ": "127.0.0.1"}

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for empty host domain, got nil")
	}
	if !strings.Contains(err.Error(), "empty domain") {
		t.Errorf("error = %q, want 'empty domain'", err)
	}
}

func TestValidateBadBlockPageIPv4(t *testing.T) {
	cfg := validConfig()
	cfg.Blocklist.ResponseMode = ResponseModeBlockPage
	cfg.Blocklist.BlockPage.IPv4 = "not-an-ip"

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for bad block_page ipv4, got nil")
	}
	if !strings.Contains(err.Error(), "ipv4") {
		t.Errorf("error = %q, want mention of ipv4", err)
	}
}

func TestValidateBadBlockPageIPv6(t *testing.T) {
	cfg := validConfig()
	cfg.Blocklist.BlockPage.IPv6 = "1.2.3.4" // IPv4 is not valid IPv6

	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error for bad block_page ipv6, got nil")
	}
	if !strings.Contains(err.Error(), "ipv6") {
		t.Errorf("error = %q, want mention of ipv6", err)
	}
}

func TestNormaliseDomain(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Example.COM.", "example.com"},
		{"  MYAPP.Local  ", "myapp.local"},
		{"sub.Domain.EXAMPLE.", "sub.domain.example"},
		{"already-clean", "already-clean"},
	}

	for _, tc := range tests {
		got := NormalizeDomain(tc.in)
		if got != tc.want {
			t.Errorf("NormalizeDomain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLoadNonexistentCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "absent.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load nonexistent: %v", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("default config not written: %v", statErr)
	}
	if !cfg.UseTLS {
		t.Error("loaded default UseTLS = false, want true")
	}
	if len(cfg.Upstream) == 0 {
		t.Error("loaded default has no upstreams")
	}
}

func TestLoadBadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("::not-valid-yaml\n"), 0o644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error loading bad YAML, got nil")
	}
	if !strings.Contains(err.Error(), "parse config") {
		t.Errorf("error = %q, want 'parse config'", err)
	}
}

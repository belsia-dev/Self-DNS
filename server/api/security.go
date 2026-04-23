package api

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

type SecurityCheck struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pass    bool   `json:"pass"`
	Message string `json:"message"`
	CanFix  bool   `json:"can_fix"`
}

type securityReport struct {
	Score  int             `json:"score"`
	Checks []SecurityCheck `json:"checks"`
}

func (a *API) runSecurityAudit() securityReport {
	cfg := a.srv.Config()
	var checks []SecurityCheck

	checks = append(checks, SecurityCheck{
		ID:   "loopback_bind",
		Name: "DNS bound to 127.0.0.1 only",
		Pass: strings.HasPrefix(cfg.Listen, "127."),
		Message: func() string {
			if strings.HasPrefix(cfg.Listen, "127.") {
				return "DNS server is not exposed to the network"
			}
			return fmt.Sprintf("DNS is listening on %s — should be 127.0.0.1", cfg.Listen)
		}(),
		CanFix: false,
	})

	checks = append(checks, SecurityCheck{
		ID:      "dot_enabled",
		Name:    "DNS-over-TLS enabled",
		Pass:    cfg.UseTLS,
		Message: map[bool]string{true: "Upstream queries are encrypted (TLS 1.2+)", false: "Upstream queries are sent in plaintext"}[cfg.UseTLS],
		CanFix:  true,
	})

	checks = append(checks, SecurityCheck{
		ID:      "rate_limit",
		Name:    "Rate limiting active",
		Pass:    cfg.RateLimit.Enabled,
		Message: map[bool]string{true: fmt.Sprintf("Max %d req/sec per source", cfg.RateLimit.MaxRPS), false: "Rate limiting is disabled — vulnerable to local query floods"}[cfg.RateLimit.Enabled],
		CanFix:  true,
	})

	configReadable := false
	if info, err := os.Stat(a.configPath); err == nil {
		if runtime.GOOS != "windows" {
			configReadable = info.Mode()&0o044 == 0
		} else {
			configReadable = true
		}
	}
	checks = append(checks, SecurityCheck{
		ID:      "config_perms",
		Name:    "Config file not world-readable",
		Pass:    configReadable,
		Message: map[bool]string{true: "Config file has secure permissions (600)", false: "Config file is world-readable — run: chmod 600 " + a.configPath}[configReadable],
		CanFix:  true,
	})

	checks = append(checks, SecurityCheck{
		ID:      "rebinding",
		Name:    "DNS rebinding protection",
		Pass:    cfg.DNSRebindingProtection,
		Message: map[bool]string{true: "Private IP responses for public domains are rejected", false: "DNS rebinding protection is disabled"}[cfg.DNSRebindingProtection],
		CanFix:  true,
	})

	checks = append(checks, SecurityCheck{
		ID:      "dnssec",
		Name:    "DNSSEC validation enabled",
		Pass:    cfg.DNSSEC,
		Message: map[bool]string{true: "DO bit is set on upstream queries", false: "DNSSEC is disabled"}[cfg.DNSSEC],
		CanFix:  true,
	})

	checks = append(checks, SecurityCheck{
		ID:      "privacy",
		Name:    "Query logging off (privacy mode)",
		Pass:    !cfg.LogQueries,
		Message: map[bool]string{true: "Query domain names are not logged", false: "Full query logging is enabled — domains are stored in memory"}[!cfg.LogQueries],
		CanFix:  true,
	})

	reachable := a.srv.IsRunning()
	checks = append(checks, SecurityCheck{
		ID:      "upstream_reachable",
		Name:    "Upstream servers reachable",
		Pass:    reachable,
		Message: map[bool]string{true: "At least one upstream DoT server is responding", false: "Cannot reach any upstream server"}[reachable],
		CanFix:  false,
	})

	checks = append(checks, SecurityCheck{
		ID:      "api_loopback",
		Name:    "API bound to 127.0.0.1 only",
		Pass:    strings.HasPrefix(cfg.APIListen, "127."),
		Message: map[bool]string{true: "Control Center API is not exposed to the network", false: "API is exposed on a non-loopback address!"}[strings.HasPrefix(cfg.APIListen, "127.")],
		CanFix:  false,
	})

	passed := 0
	for _, c := range checks {
		if c.Pass {
			passed++
		}
	}
	score := passed * 100 / len(checks)

	return securityReport{Score: score, Checks: checks}
}

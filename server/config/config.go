package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ResponseModeNXDomain  = "nxdomain"
	ResponseModeBlockPage = "block_page"
)

func NormalizeDomain(name string) string {
	d := strings.ToLower(strings.TrimSpace(name))
	return strings.TrimSuffix(d, ".")
}

type Config struct {
	Listen                 string            `yaml:"listen"                   json:"listen"`
	APIListen              string            `yaml:"api_listen"               json:"api_listen"`
	UseTLS                 bool              `yaml:"use_tls"                  json:"use_tls"`
	LogQueries             bool              `yaml:"log_queries"              json:"log_queries"`
	DNSRebindingProtection bool              `yaml:"dns_rebinding_protection" json:"dns_rebinding_protection"`
	DNSSEC                 bool              `yaml:"dnssec"                   json:"dnssec"`
	ServiceMode            string            `yaml:"service_mode"             json:"service_mode"`
	Upstream               []string          `yaml:"upstream"                 json:"upstream"`
	Cache                  CacheConfig       `yaml:"cache"                    json:"cache"`
	RateLimit              RateLimitConfig   `yaml:"rate_limit"               json:"rate_limit"`
	Blocklist              BlocklistConfig   `yaml:"blocklist"                json:"blocklist"`
	Hosts                  map[string]string `yaml:"hosts"                    json:"hosts"`
}

type CacheConfig struct {
	Enabled              bool   `yaml:"enabled"                json:"enabled"`
	MaxSize              int    `yaml:"max_size"               json:"max_size"`
	MinTTL               uint32 `yaml:"min_ttl"                json:"min_ttl"`
	StaleWhileRevalidate bool   `yaml:"stale_while_revalidate" json:"stale_while_revalidate"`
}

type RateLimitConfig struct {
	Enabled         bool     `yaml:"enabled"           json:"enabled"`
	MaxRPS          int      `yaml:"max_rps"           json:"max_rps"`
	BurstMultiplier int      `yaml:"burst_multiplier" json:"burst_multiplier"`
	WhitelistIPs    []string `yaml:"whitelist_ips"    json:"whitelist_ips"`
	PerDomainMaxRPS int      `yaml:"per_domain_max_rps" json:"per_domain_max_rps"`
}

type BlocklistConfig struct {
	Enabled      bool            `yaml:"enabled"       json:"enabled"`
	Files        []string        `yaml:"files"         json:"files"`
	Domains      []string        `yaml:"domains"       json:"domains"`
	ResponseMode string          `yaml:"response_mode" json:"response_mode"`
	BlockPage    BlockPageConfig `yaml:"block_page"    json:"block_page"`
}

type BlockPageConfig struct {
	Bind string `yaml:"bind" json:"bind"`
	IPv4 string `yaml:"ipv4" json:"ipv4"`
	IPv6 string `yaml:"ipv6" json:"ipv6"`
	HTML string `yaml:"html" json:"html"`
	CSS  string `yaml:"css"  json:"css"`
	JS   string `yaml:"js"   json:"js"`
}

func Clone(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	next := *cfg
	next.Upstream = append([]string(nil), cfg.Upstream...)
	next.RateLimit.WhitelistIPs = append([]string(nil), cfg.RateLimit.WhitelistIPs...)
	next.Blocklist.Files = append([]string(nil), cfg.Blocklist.Files...)
	next.Blocklist.Domains = append([]string(nil), cfg.Blocklist.Domains...)

	if cfg.Hosts != nil {
		next.Hosts = make(map[string]string, len(cfg.Hosts))
		for domain, ip := range cfg.Hosts {
			next.Hosts[domain] = ip
		}
	}

	return &next
}

func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	info, statErr := os.Stat(path)
	if statErr == nil {
		if runtime.GOOS != "windows" {
			if info.Mode()&0o044 != 0 {
				fmt.Fprintf(os.Stderr, "WARNING: config %s is world-readable (mode %04o)\n", path, info.Mode().Perm())
			}
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if saveErr := Save(cfg, path); saveErr != nil {
				return nil, fmt.Errorf("write default config: %w", saveErr)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := Prepare(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func Save(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func Prepare(cfg *Config) error {
	applyDefaults(cfg)
	cfg.ServiceMode = normaliseServiceMode(cfg.ServiceMode)
	cfg.Listen = normaliseDNSListen(cfg.Listen, cfg.ServiceMode, "53")
	cfg.APIListen = ensureLoopback(cfg.APIListen, "5380")
	cfg.Blocklist.ResponseMode = normaliseBlockedResponse(cfg.Blocklist.ResponseMode)
	if cfg.Blocklist.BlockPage.Bind == "" {
		cfg.Blocklist.BlockPage.Bind = defaultBlockPageBind(cfg.ServiceMode)
	}
	cfg.Blocklist.BlockPage.IPv4 = strings.TrimSpace(cfg.Blocklist.BlockPage.IPv4)
	cfg.Blocklist.BlockPage.IPv6 = strings.TrimSpace(cfg.Blocklist.BlockPage.IPv6)
	if cfg.Blocklist.BlockPage.HTML == "" {
		cfg.Blocklist.BlockPage.HTML = defaultBlockPageHTML()
	}
	if cfg.Blocklist.BlockPage.CSS == "" {
		cfg.Blocklist.BlockPage.CSS = defaultBlockPageCSS()
	}
	if cfg.Blocklist.BlockPage.JS == "" {
		cfg.Blocklist.BlockPage.JS = defaultBlockPageJS()
	}
	return validate(cfg)
}

func validate(cfg *Config) error {
	if cfg.Listen == "" {
		return fmt.Errorf("listen address must not be empty")
	}
	if cfg.APIListen == "" {
		return fmt.Errorf("api_listen address must not be empty")
	}
	if cfg.ServiceMode != "local" && cfg.ServiceMode != "internal" && cfg.ServiceMode != "external" {
		return fmt.Errorf("service_mode must be one of: local, internal, external")
	}
	if len(cfg.Upstream) == 0 {
		return fmt.Errorf("at least one upstream DNS server is required")
	}
	for i, upstream := range cfg.Upstream {
		if _, _, err := net.SplitHostPort(upstream); err != nil {
			return fmt.Errorf("upstream[%d] %q is not a valid host:port: %w", i, upstream, err)
		}
	}
	if cfg.Blocklist.ResponseMode != ResponseModeNXDomain && cfg.Blocklist.ResponseMode != ResponseModeBlockPage {
		return fmt.Errorf("blocklist.response_mode must be one of: %s, %s", ResponseModeNXDomain, ResponseModeBlockPage)
	}
	if cfg.Cache.MaxSize <= 0 {
		cfg.Cache.MaxSize = 10000
	}
	if cfg.RateLimit.MaxRPS <= 0 {
		cfg.RateLimit.MaxRPS = 200
	}
	if cfg.RateLimit.BurstMultiplier <= 0 {
		cfg.RateLimit.BurstMultiplier = 3
	}
	if cfg.RateLimit.WhitelistIPs == nil {
		cfg.RateLimit.WhitelistIPs = []string{}
	}
	if cfg.Blocklist.Files == nil {
		cfg.Blocklist.Files = []string{}
	}
	if cfg.Blocklist.Domains == nil {
		cfg.Blocklist.Domains = []string{}
	}
	if cfg.Blocklist.BlockPage.Bind == "" {
		return fmt.Errorf("blocklist.block_page.bind must not be empty")
	}
	if _, _, err := net.SplitHostPort(cfg.Blocklist.BlockPage.Bind); err != nil {
		return fmt.Errorf("blocklist.block_page.bind must be host:port")
	}
	if ip := cfg.Blocklist.BlockPage.IPv4; ip != "" {
		parsed := net.ParseIP(ip)
		if parsed == nil || parsed.To4() == nil {
			return fmt.Errorf("blocklist.block_page.ipv4 must be a valid IPv4 address")
		}
	}
	if ip := cfg.Blocklist.BlockPage.IPv6; ip != "" {
		parsed := net.ParseIP(ip)
		if parsed == nil || parsed.To4() != nil {
			return fmt.Errorf("blocklist.block_page.ipv6 must be a valid IPv6 address")
		}
	}
	for domain, ip := range cfg.Hosts {
		if strings.TrimSpace(domain) == "" {
			return fmt.Errorf("hosts entry has empty domain")
		}
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("hosts[%q] has invalid IP address %q", domain, ip)
		}
	}
	for _, entry := range cfg.RateLimit.WhitelistIPs {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if net.ParseIP(entry) == nil {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				return fmt.Errorf("rate_limit.whitelist_ips entry %q is not a valid IP or CIDR", entry)
			}
		}
	}
	return nil
}

func defaultConfig() *Config {
	return &Config{
		Listen:                 "127.0.0.1:53",
		APIListen:              "127.0.0.1:5380",
		UseTLS:                 true,
		LogQueries:             false,
		DNSRebindingProtection: true,
		DNSSEC:                 true,
		ServiceMode:            "local",
		Upstream: []string{
			"1.1.1.1:853",
			"8.8.8.8:853",
			"9.9.9.9:853",
			"149.112.112.112:853",
		},
		Cache: CacheConfig{
			Enabled:              true,
			MaxSize:              10000,
			MinTTL:               60,
			StaleWhileRevalidate: true,
		},
		RateLimit: RateLimitConfig{
			Enabled:         true,
			MaxRPS:          200,
			BurstMultiplier: 3,
			WhitelistIPs:    []string{},
			PerDomainMaxRPS: 0,
		},
		Blocklist: BlocklistConfig{
			Enabled:      false,
			Files:        []string{},
			Domains:      []string{},
			ResponseMode: ResponseModeNXDomain,
			BlockPage: BlockPageConfig{
				Bind: defaultBlockPageBind("local"),
				HTML: defaultBlockPageHTML(),
				CSS:  defaultBlockPageCSS(),
				JS:   defaultBlockPageJS(),
			},
		},
		Hosts: map[string]string{
			"myapp.local": "127.0.0.1",
		},
	}
}

func applyDefaults(cfg *Config) {
	if cfg.ServiceMode == "" {
		cfg.ServiceMode = "local"
	}
	if cfg.Blocklist.ResponseMode == "" {
		cfg.Blocklist.ResponseMode = ResponseModeNXDomain
	}
	if cfg.Blocklist.Files == nil {
		cfg.Blocklist.Files = []string{}
	}
	if cfg.Blocklist.Domains == nil {
		cfg.Blocklist.Domains = []string{}
	}
}

func normaliseServiceMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "local":
		return "local"
	case "internal":
		return "internal"
	case "external", "public":
		return "external"
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}

func normaliseBlockedResponse(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", ResponseModeNXDomain:
		return ResponseModeNXDomain
	case ResponseModeBlockPage, "html":
		return ResponseModeBlockPage
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}

func normaliseDNSListen(addr, serviceMode, defaultPort string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = defaultPort
	}
	switch normaliseServiceMode(serviceMode) {
	case "local":
		host = "127.0.0.1"
	default:
		host = "0.0.0.0"
	}
	return net.JoinHostPort(host, port)
}

func ensureLoopback(addr, defaultPort string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = defaultPort
	}
	if !isLoopback(host) {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

func isLoopback(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func defaultBlockPageBind(serviceMode string) string {
	switch normaliseServiceMode(serviceMode) {
	case "local":
		return "127.0.0.1:80"
	default:
		return "0.0.0.0:80"
	}
}

func defaultBlockPageHTML() string {
	return `<main class="shell">
  <section class="card">
    <p class="eyebrow">Blocked by SelfDNS</p>
    <h1>{{DOMAIN}}</h1>
    <p class="lead">This domain is blocked by your DNS server.</p>
    <p class="meta">If you need access, contact the DNS administrator and include the blocked domain shown above.</p>
    <div class="details">
      <div>
        <span class="label">Requested host</span>
        <strong>{{HOST}}</strong>
      </div>
      <div>
        <span class="label">Requested path</span>
        <strong>{{PATH}}</strong>
      </div>
    </div>
  </section>
</main>`
}

func defaultBlockPageCSS() string {
	return `:root {
  color-scheme: dark;
  --bg0: #081120;
  --bg1: #11243b;
  --panel: rgba(10, 18, 31, 0.86);
  --border: rgba(125, 167, 255, 0.16);
  --text: #f8fafc;
  --muted: #b8c4d8;
  --accent: #7dd3fc;
  --accent-2: #f59e0b;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  min-height: 100vh;
  font-family: "Inter", "Segoe UI", sans-serif;
  color: var(--text);
  background:
    radial-gradient(circle at top left, rgba(125, 211, 252, 0.22), transparent 34%),
    radial-gradient(circle at bottom right, rgba(245, 158, 11, 0.18), transparent 26%),
    linear-gradient(135deg, var(--bg0), var(--bg1));
}

.shell {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 32px 18px;
}

.card {
  width: min(720px, 100%);
  padding: 32px;
  border-radius: 28px;
  background: var(--panel);
  border: 1px solid var(--border);
  box-shadow: 0 24px 80px rgba(2, 6, 23, 0.42);
  backdrop-filter: blur(18px);
}

.eyebrow {
  margin: 0 0 14px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  font-size: 12px;
  color: var(--accent);
}

h1 {
  margin: 0;
  font-size: clamp(32px, 7vw, 56px);
  line-height: 0.94;
}

.lead {
  margin: 20px 0 10px;
  font-size: 18px;
  color: var(--text);
}

.meta {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--muted);
}

.details {
  margin-top: 28px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 14px;
}

.details > div {
  padding: 16px 18px;
  border-radius: 18px;
  background: rgba(15, 23, 42, 0.66);
  border: 1px solid rgba(148, 163, 184, 0.14);
}

.label {
  display: block;
  margin-bottom: 8px;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: var(--accent-2);
}`
}

func defaultBlockPageJS() string {
	return `document.documentElement.dataset.blockedDomain = window.SELFDNS_BLOCK_PAGE?.domain ?? "";`
}

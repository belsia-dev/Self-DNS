package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen                 string            `yaml:"listen"                   json:"listen"`
	APIListen              string            `yaml:"api_listen"               json:"api_listen"`
	UseTLS                 bool              `yaml:"use_tls"                  json:"use_tls"`
	LogQueries             bool              `yaml:"log_queries"              json:"log_queries"`
	DNSRebindingProtection bool              `yaml:"dns_rebinding_protection" json:"dns_rebinding_protection"`
	DNSSEC                 bool              `yaml:"dnssec"                   json:"dnssec"`
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
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Files   []string `yaml:"files"   json:"files"`
	Domains []string `yaml:"domains" json:"domains"`
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

	if err := validate(cfg); err != nil {
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

func validate(cfg *Config) error {
	if cfg.Listen == "" {
		return fmt.Errorf("listen address must not be empty")
	}
	if cfg.APIListen == "" {
		return fmt.Errorf("api_listen address must not be empty")
	}
	if len(cfg.Upstream) == 0 {
		return fmt.Errorf("at least one upstream DNS server is required")
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
			Enabled: false,
			Files:   []string{},
			Domains: []string{},
		},
		Hosts: map[string]string{
			"myapp.local": "127.0.0.1",
		},
	}
}

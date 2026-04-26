package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/belsia-dev/Self-DNS/server/cache"
	"github.com/belsia-dev/Self-DNS/server/config"
	"github.com/belsia-dev/Self-DNS/server/stats"
)

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	uptime := time.Since(a.stats.StartTime()).Seconds()
	jsonOK(w, map[string]any{
		"running": a.srv.IsRunning(),
		"uptime":  uptime,
		"version": a.version,
		"listen":  a.currentConfig().Listen,
	})
}

func (a *API) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	snap := a.stats.Snapshot()
	snap2 := struct {
		stats.Snapshot
		QPSHistory []int64 `json:"qps_history"`
	}{
		Snapshot:   snap,
		QPSHistory: a.stats.QPSHistory(),
	}
	jsonOK(w, snap2)
}

func (a *API) handleQueries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	jsonOK(w, a.stats.Queries(500))
}

func (a *API) handleSecurity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	jsonOK(w, a.runSecurityAudit())
}

func (a *API) handleBlocklist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	jsonOK(w, map[string]any{
		"enabled": a.blocker.IsEnabled(),
		"domains": a.blocker.List(),
		"count":   a.blocker.Count(),
		"files":   a.blocker.Files(),
	})
}

func (a *API) handleBlocklistToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	next := !a.blocker.IsEnabled()
	cfg := a.currentConfig()
	cfg.Blocklist.Enabled = next
	if err := config.Save(cfg, a.configPath); err != nil {
		serverError(w, err)
		return
	}
	a.blocker.Toggle(next)
	a.cache.Flush()
	go flushOSDNSCache()
	if err := a.srv.ApplyBlockPageConfig(cfg); err != nil {
		serverError(w, err)
		return
	}
	a.cfg = cfg
	jsonOK(w, map[string]any{"enabled": next})
}

func (a *API) handleBlocklistAdd(w http.ResponseWriter, r *http.Request) {
	a.mutateBlocklistDomain(w, r, true, "added")
}

func (a *API) handleBlocklistRemove(w http.ResponseWriter, r *http.Request) {
	a.mutateBlocklistDomain(w, r, false, "removed")
}

func (a *API) mutateBlocklistDomain(w http.ResponseWriter, r *http.Request, include bool, status string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var body struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Domain == "" {
		badRequest(w, "domain is required")
		return
	}
	domain := config.NormalizeDomain(body.Domain)
	if domain == "" {
		badRequest(w, "domain is required")
		return
	}

	if include {
		a.blocker.Add(domain)
	} else {
		a.blocker.Remove(domain)
	}
	a.cache.Delete(domain)
	go flushOSDNSCache()

	cfg := a.currentConfig()
	filtered := cfg.Blocklist.Domains[:0]
	present := false
	for _, item := range cfg.Blocklist.Domains {
		if config.NormalizeDomain(item) == domain {
			present = true
			if !include {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	if include && !present {
		filtered = append(filtered, domain)
	}
	cfg.Blocklist.Domains = filtered

	if err := config.Save(cfg, a.configPath); err != nil {
		serverError(w, err)
		return
	}
	if err := a.srv.ApplyBlockPageConfig(cfg); err != nil {
		serverError(w, err)
		return
	}
	a.cfg = cfg
	jsonOK(w, map[string]string{"status": status})
}

func (a *API) handleHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cfg := a.currentConfig()
	type hostEntry struct {
		Domain string `json:"domain"`
		IP     string `json:"ip"`
	}
	entries := make([]hostEntry, 0, len(cfg.Hosts))
	for d, ip := range cfg.Hosts {
		entries = append(entries, hostEntry{Domain: d, IP: ip})
	}
	jsonOK(w, entries)
}

func (a *API) handleHostsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var body struct {
		Domain string `json:"domain"`
		IP     string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Domain == "" || body.IP == "" {
		badRequest(w, "domain and ip are required")
		return
	}
	cfg := a.currentConfig()
	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]string)
	}
	cfg.Hosts[strings.ToLower(body.Domain)] = body.IP
	if err := config.Save(cfg, a.configPath); err != nil {
		serverError(w, err)
		return
	}
	if err := a.srv.ApplyBlockPageConfig(cfg); err != nil {
		serverError(w, err)
		return
	}
	a.cfg = cfg
	jsonOK(w, map[string]string{"status": "added"})
}

func (a *API) handleHostsRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var body struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Domain == "" {
		badRequest(w, "domain is required")
		return
	}
	cfg := a.currentConfig()
	delete(cfg.Hosts, strings.ToLower(body.Domain))
	if err := config.Save(cfg, a.configPath); err != nil {
		serverError(w, err)
		return
	}
	if err := a.srv.ApplyBlockPageConfig(cfg); err != nil {
		serverError(w, err)
		return
	}
	a.cfg = cfg
	jsonOK(w, map[string]string{"status": "removed"})
}

func (a *API) handleBlockPageConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var body struct {
		ResponseMode string                 `json:"response_mode"`
		BlockPage    config.BlockPageConfig `json:"block_page"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, "invalid block page config JSON")
		return
	}

	current := a.currentConfig()
	next := config.Clone(current)
	next.Blocklist.ResponseMode = body.ResponseMode
	next.Blocklist.BlockPage = body.BlockPage
	if err := config.Prepare(next); err != nil {
		badRequest(w, err.Error())
		return
	}

	if err := a.srv.ApplyBlockPageConfig(next); err != nil {
		serverError(w, err)
		return
	}
	if err := config.Save(next, a.configPath); err != nil {
		if rollbackErr := a.srv.ApplyBlockPageConfig(current); rollbackErr != nil {
			serverError(w, fmt.Errorf("save live block page config: %w (rollback failed: %v)", err, rollbackErr))
			return
		}
		serverError(w, err)
		return
	}

	a.cfg = next
	jsonOK(w, map[string]any{
		"status":           "applied",
		"response_mode":    next.Blocklist.ResponseMode,
		"block_page":       next.Blocklist.BlockPage,
		"requires_restart": false,
	})
}

func (a *API) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonOK(w, a.currentConfig())
	case http.MethodPost:
		var cfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			badRequest(w, "invalid config JSON")
			return
		}
		if err := config.Prepare(&cfg); err != nil {
			badRequest(w, err.Error())
			return
		}

		if err := config.Save(&cfg, a.configPath); err != nil {
			serverError(w, err)
			return
		}
		if err := a.srv.Reload(&cfg); err != nil {
			serverError(w, err)
			return
		}
		a.blocker.Toggle(cfg.Blocklist.Enabled)
		a.cfg = &cfg
		jsonOK(w, map[string]string{"status": "saved and reloaded"})
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := a.srv.Reload(a.currentConfig()); err != nil {
		serverError(w, err)
		return
	}
	jsonOK(w, map[string]string{"status": "restarted"})
}

func (a *API) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	a.srv.Stop()
	jsonOK(w, map[string]string{"status": "stopped"})
}

func (a *API) currentConfig() *config.Config {
	if cfg := a.srv.ConfigClone(); cfg != nil {
		return cfg
	}
	return config.Clone(a.cfg)
}

func (a *API) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	jsonOK(w, a.cache.Stats())
}

func (a *API) handleCacheFlush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	a.cache.Flush()
	jsonOK(w, map[string]string{"status": "flushed"})
}

func (a *API) handleCacheExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="selfdns-cache.json"`)
	jsonOK(w, a.cache.Export())
}

func (a *API) handleCacheImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var exp cache.Export
	if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
		badRequest(w, "invalid export payload")
		return
	}
	n := a.cache.Import(exp)
	jsonOK(w, map[string]any{"status": "imported", "entries": n})
}

func (a *API) handleCacheHot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	jsonOK(w, a.cache.Hot(20))
}

func (a *API) handleUpstreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	jsonOK(w, a.srv.Resolver().UpstreamsHealth())
}

func (a *API) handlePrefetchRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	go a.srv.Resolver().PrefetchNow()
	jsonOK(w, map[string]string{"status": "triggered"})
}

func (a *API) handleCACert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cert := a.srv.BlockPageCACert()
	if len(cert) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", `attachment; filename="selfdns-ca.crt"`)
	_, _ = w.Write(cert)
}

func (a *API) handleNetworkDNS(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonOK(w, map[string]any{
			"network_dns": a.srv.Resolver().NetworkDNS(),
			"system_dns":  a.srv.Resolver().SystemDNS(),
		})
	case http.MethodPost:
		go a.srv.Resolver().RefreshNetworkDNS()
		go a.srv.Resolver().RefreshSystemDNS()
		jsonOK(w, map[string]string{"status": "refreshing"})
	default:
		methodNotAllowed(w)
	}
}

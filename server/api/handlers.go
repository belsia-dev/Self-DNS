package api

import (
	"encoding/json"
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
		"listen":  a.cfg.Listen,
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
	a.blocker.Toggle(next)
	cfg := a.srv.Config()
	cfg.Blocklist.Enabled = next
	_ = config.Save(cfg, a.configPath)
	jsonOK(w, map[string]any{"enabled": next})
}

func (a *API) handleBlocklistAdd(w http.ResponseWriter, r *http.Request) {
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
	a.blocker.Add(body.Domain)
	jsonOK(w, map[string]string{"status": "added"})
}

func (a *API) handleBlocklistRemove(w http.ResponseWriter, r *http.Request) {
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
	a.blocker.Remove(body.Domain)
	jsonOK(w, map[string]string{"status": "removed"})
}

func (a *API) handleHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	cfg := a.srv.Config()
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
	cfg := a.srv.Config()
	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]string)
	}
	cfg.Hosts[strings.ToLower(body.Domain)] = body.IP
	if err := config.Save(cfg, a.configPath); err != nil {
		serverError(w, err)
		return
	}
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
	cfg := a.srv.Config()
	delete(cfg.Hosts, strings.ToLower(body.Domain))
	if err := config.Save(cfg, a.configPath); err != nil {
		serverError(w, err)
		return
	}
	jsonOK(w, map[string]string{"status": "removed"})
}

func (a *API) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jsonOK(w, a.srv.Config())
	case http.MethodPost:
		var cfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			badRequest(w, "invalid config JSON")
			return
		}
		cfg.Listen = ensureLoopback(cfg.Listen, "53")
		cfg.APIListen = ensureLoopback(cfg.APIListen, "5380")

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
	if err := a.srv.Reload(a.srv.Config()); err != nil {
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

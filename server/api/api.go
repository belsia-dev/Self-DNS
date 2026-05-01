package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/belsia-dev/Self-DNS/server/blocker"
	"github.com/belsia-dev/Self-DNS/server/cache"
	"github.com/belsia-dev/Self-DNS/server/config"
	"github.com/belsia-dev/Self-DNS/server/dns"
	"github.com/belsia-dev/Self-DNS/server/stats"
)

type API struct {
	cfgMu      sync.RWMutex
	cfg        *config.Config
	srv        *dns.Server
	blocker    *blocker.Blocker
	stats      *stats.Stats
	cache      *cache.Cache
	configPath string
	version    string
	httpServer *http.Server
}

func New(
	cfg *config.Config,
	srv *dns.Server,
	bl *blocker.Blocker,
	st *stats.Stats,
	ch *cache.Cache,
	configPath string,
	version string,
) *API {
	return &API{
		cfg:        cfg,
		srv:        srv,
		blocker:    bl,
		stats:      st,
		cache:      ch,
		configPath: configPath,
		version:    version,
	}
}

func (a *API) Start() error {
	mux := http.NewServeMux()
	a.registerRoutes(mux)

	handler := a.loopbackOnly(a.cors(mux))

	a.httpServer = &http.Server{
		Addr:         a.cfg.APIListen,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	host, _, err := net.SplitHostPort(a.cfg.APIListen)
	if err != nil || !isLoopback(host) {
		return fmt.Errorf("API listen address %q must be a loopback address (127.x.x.x)", a.cfg.APIListen)
	}

	log.Printf("API listening on http://%s", a.cfg.APIListen)
	return a.httpServer.ListenAndServe()
}

func (a *API) Stop() {
	if a.httpServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.httpServer.Shutdown(ctx)
}

func (a *API) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/status", a.handleStatus)
	mux.HandleFunc("/api/stats", a.handleStats)
	mux.HandleFunc("/api/queries", a.handleQueries)
	mux.HandleFunc("/api/security", a.handleSecurity)
	mux.HandleFunc("/api/blocklist", a.handleBlocklist)
	mux.HandleFunc("/api/blocklist/toggle", a.handleBlocklistToggle)
	mux.HandleFunc("/api/blocklist/add", a.handleBlocklistAdd)
	mux.HandleFunc("/api/blocklist/remove", a.handleBlocklistRemove)
	mux.HandleFunc("/api/hosts", a.handleHosts)
	mux.HandleFunc("/api/hosts/add", a.handleHostsAdd)
	mux.HandleFunc("/api/hosts/remove", a.handleHostsRemove)
	mux.HandleFunc("/api/config", a.handleConfig)
	mux.HandleFunc("/api/config/block-page", a.handleBlockPageConfig)
	mux.HandleFunc("/api/server/restart", a.handleRestart)
	mux.HandleFunc("/api/server/stop", a.handleStop)
	mux.HandleFunc("/api/cache/stats", a.handleCacheStats)
	mux.HandleFunc("/api/cache/flush", a.handleCacheFlush)
	mux.HandleFunc("/api/cache/export", a.handleCacheExport)
	mux.HandleFunc("/api/cache/import", a.handleCacheImport)
	mux.HandleFunc("/api/cache/hot", a.handleCacheHot)
	mux.HandleFunc("/api/upstreams", a.handleUpstreams)
	mux.HandleFunc("/api/prefetch/run", a.handlePrefetchRun)
	mux.HandleFunc("/api/network-dns", a.handleNetworkDNS)
	mux.HandleFunc("/api/ca-cert", a.handleCACert)
}

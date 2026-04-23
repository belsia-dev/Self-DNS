package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/belsia-dev/Self-DNS/server/api"
	"github.com/belsia-dev/Self-DNS/server/blocker"
	"github.com/belsia-dev/Self-DNS/server/cache"
	"github.com/belsia-dev/Self-DNS/server/config"
	"github.com/belsia-dev/Self-DNS/server/dns"
	"github.com/belsia-dev/Self-DNS/server/resolver"
	"github.com/belsia-dev/Self-DNS/server/stats"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const appVersion = "1.0.0"

type App struct {
	ctx       context.Context
	mu        sync.Mutex
	dnsServer *dns.Server
	apiServer *api.API
	cfgPath   string
	startErr  string
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	if !a.isAdmin() {
		log.Println("[selfdns-app] not running as administrator, triggering elevation...")
		a.Elevate()
		return
	}

	a.ctx = ctx
	a.cfgPath = a.resolveConfigPath()
	log.Printf("[selfdns-app] v%s starting as administrator, config: %s", appVersion, a.cfgPath)
	go a.startDNS()
}

func (a *App) isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func (a *App) shutdown(_ context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.apiServer != nil {
		a.apiServer.Stop()
	}
	if a.dnsServer != nil {
		a.dnsServer.Stop()
	}
	log.Println("[selfdns-app] shutdown complete")
}

func (a *App) startDNS() {
	cfg, err := config.Load(a.cfgPath)
	if err != nil {
		a.setStartErr(fmt.Sprintf("config: %v", err))
		return
	}

	st := stats.New()
	ch := cache.New(cfg.Cache.MaxSize, cfg.Cache.MinTTL)
	bl := blocker.New(cfg.Blocklist)
	res := resolver.New(cfg, ch, st)

	srv, err := dns.New(cfg, res, bl, st, ch)
	if err != nil {
		a.setStartErr(fmt.Sprintf("server init: %v", err))
		return
	}

	if err := srv.Start(); err != nil {
		a.setStartErr(fmt.Sprintf(
			"DNS server failed to bind %s: %v\n\nTip: port 53 requires Administrator. Right-click the app and choose \"Run as administrator\".",
			cfg.Listen, err,
		))
		return
	}

	a.mu.Lock()
	a.dnsServer = srv
	a.mu.Unlock()

	log.Printf("[selfdns-app] DNS ready on %s (UDP+TCP)", cfg.Listen)

	apiSrv := api.New(cfg, srv, bl, st, ch, a.cfgPath, appVersion)

	a.mu.Lock()
	a.apiServer = apiSrv
	a.mu.Unlock()

	log.Printf("[selfdns-app] API ready on http://%s", cfg.APIListen)

	if err := apiSrv.Start(); err != nil {
		log.Printf("[selfdns-app] API stopped: %v", err)
	}
}

func (a *App) setStartErr(msg string) {
	log.Printf("[selfdns-app] ERROR: %s", msg)
	a.mu.Lock()
	a.startErr = msg
	a.mu.Unlock()
}

func (a *App) GetVersion() string { return appVersion }

func (a *App) GetOS() string { return "windows" }

func (a *App) GetConfigPath() string { return a.cfgPath }

func (a *App) GetStartError() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.startErr
}

func (a *App) IsDNSRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dnsServer != nil && a.dnsServer.IsRunning()
}

func (a *App) Elevate() string {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Sprintf("error getting executable: %v", err)
	}
	_ = exec.Command(
		"powershell", "-Command",
		fmt.Sprintf("Start-Process -FilePath '%s' -Verb RunAs", exe),
	).Start()
	os.Exit(0)
	return "ok"
}

func (a *App) OpenFileDialog(title string, filters []wailsruntime.FileFilter) string {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
	if err != nil {
		return ""
	}
	return path
}

func (a *App) FixConfigPermissions(_ string) string {
	return "ok"
}

func (a *App) resolveConfigPath() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "SelfDNS", "config.yaml")
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
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
	if !a.isRoot() {
		log.Println("[selfdns-app] not running as root, triggering elevation...")
		a.Elevate()
		return
	}

	a.ctx = ctx
	a.cfgPath = a.resolveConfigPath()
	log.Printf("[selfdns-app] v%s starting as root, config: %s", appVersion, a.cfgPath)
	go a.startDNS()
}

func (a *App) isRoot() bool {
	if goruntime.GOOS == "windows" {
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	}
	return os.Geteuid() == 0
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
		msg := fmt.Sprintf("DNS server failed to bind %s: %v", cfg.Listen, err)
		if goruntime.GOOS != "windows" {
			msg += "\n\nTip: port 53 requires root. Run: sudo selfdns-app"
		}
		a.setStartErr(msg)
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

func (a *App) GetOS() string { return goruntime.GOOS }

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

	switch goruntime.GOOS {
	case "darwin":
		cmd := fmt.Sprintf("do shell script \"'%s'\" with administrator privileges", exe)
		_ = exec.Command("osascript", "-e", cmd).Start()
		os.Exit(0)
	case "windows":
		_ = exec.Command("powershell", "Start-Process", fmt.Sprintf("'%s'", exe), "-Verb", "runas").Start()
		os.Exit(0)
	default:
		return "Manual restart with sudo is required on your OS."
	}
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

func (a *App) FixConfigPermissions(path string) string {
	if goruntime.GOOS == "windows" {
		return "ok"
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return "ok"
}

func (a *App) resolveConfigPath() string {
	if _, err := os.Stat("/etc/selfdns/config.yaml"); err == nil {
		return "/etc/selfdns/config.yaml"
	}
	switch goruntime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("ProgramData"), "SelfDNS", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "selfdns", "config.yaml")
}

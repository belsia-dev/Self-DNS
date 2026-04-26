package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
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
	ctx         context.Context
	mu          sync.Mutex
	dnsServer   *dns.Server
	apiServer   *api.API
	cfgPath     string
	startErr    string
	originalDNS map[string][]string
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
	a.setSystemDNS()
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
	if a.apiServer != nil {
		a.apiServer.Stop()
	}
	if a.dnsServer != nil {
		a.dnsServer.Stop()
	}
	a.mu.Unlock()
	a.restoreSystemDNS()
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

	srv.SetCADir(filepath.Dir(a.cfgPath))

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

	go a.installBlockPageCA()

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

func (a *App) setSystemDNS() {
	switch goruntime.GOOS {
	case "darwin":
		services, err := a.macNetworkServices()
		if err != nil {
			log.Printf("[selfdns-app] DNS: could not list network services: %v", err)
			return
		}
		orig := make(map[string][]string)
		for _, svc := range services {
			servers, _ := a.macDNSServers(svc)
			orig[svc] = servers
			if err := exec.Command("networksetup", "-setdnsservers", svc, "127.0.0.1").Run(); err != nil {
				log.Printf("[selfdns-app] DNS: could not set %s: %v", svc, err)
			}
		}
		a.mu.Lock()
		a.originalDNS = orig
		a.mu.Unlock()
		_ = exec.Command("dscacheutil", "-flushcache").Run()
		_ = exec.Command("killall", "-HUP", "mDNSResponder").Run()
		log.Printf("[selfdns-app] DNS: set 127.0.0.1 on %d interface(s)", len(services))

	case "linux":
		if out, err := exec.Command("resolvectl", "dns").Output(); err == nil {
			a.mu.Lock()
			a.originalDNS = map[string][]string{"_raw": {string(out)}}
			a.mu.Unlock()
		}
		ifaces, _ := exec.Command("sh", "-c",
			"ip link show up | awk -F': ' '/^[0-9]+:/{print $2}' | grep -v lo").Output()
		for _, iface := range strings.Fields(string(ifaces)) {
			_ = exec.Command("resolvectl", "dns", iface, "127.0.0.1").Run()
		}
		_ = exec.Command("resolvectl", "flush-caches").Run()
		log.Println("[selfdns-app] DNS: set 127.0.0.1 via resolvectl")

	case "windows":
		_ = exec.Command("powershell", "-Command",
			"Get-NetAdapter | Where-Object {$_.Status -eq 'Up'} | "+
				"ForEach-Object { Set-DnsClientServerAddress -InterfaceAlias $_.Name -ServerAddresses '127.0.0.1' }",
		).Run()
		_ = exec.Command("ipconfig", "/flushdns").Run()
		log.Println("[selfdns-app] DNS: set 127.0.0.1 on all active adapters")
	}
}

func (a *App) restoreSystemDNS() {
	switch goruntime.GOOS {
	case "darwin":
		a.mu.Lock()
		orig := a.originalDNS
		a.mu.Unlock()
		if len(orig) == 0 {
			return
		}
		for svc, servers := range orig {
			args := []string{"-setdnsservers", svc}
			if len(servers) == 0 {
				args = append(args, "empty")
			} else {
				args = append(args, servers...)
			}
			_ = exec.Command("networksetup", args...).Run()
		}
		_ = exec.Command("dscacheutil", "-flushcache").Run()
		_ = exec.Command("killall", "-HUP", "mDNSResponder").Run()
		log.Println("[selfdns-app] DNS: original resolvers restored")

	case "linux":
		_ = exec.Command("resolvectl", "revert").Run()
		log.Println("[selfdns-app] DNS: reverted via resolvectl")

	case "windows":
		_ = exec.Command("powershell", "-Command",
			"Get-NetAdapter | Where-Object {$_.Status -eq 'Up'} | "+
				"ForEach-Object { Set-DnsClientServerAddress -InterfaceAlias $_.Name -ResetServerAddresses }",
		).Run()
		_ = exec.Command("ipconfig", "/flushdns").Run()
		log.Println("[selfdns-app] DNS: reset adapters to DHCP DNS")
	}
}

func (a *App) macNetworkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, err
	}
	var services []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "An asterisk") || strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}
	return services, nil
}

func (a *App) macDNSServers(service string) ([]string, error) {
	out, err := exec.Command("networksetup", "-getdnsservers", service).Output()
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(out))
	if strings.Contains(text, "aren't any DNS Servers") {
		return nil, nil
	}
	var servers []string
	for _, line := range strings.Split(text, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			servers = append(servers, s)
		}
	}
	return servers, nil
}

func (a *App) SystemDNSStatus() map[string]any {
	a.mu.Lock()
	orig := a.originalDNS
	a.mu.Unlock()
	return map[string]any{
		"active":       len(orig) > 0,
		"originalDNS":  orig,
	}
}

func (a *App) installBlockPageCA() {
	a.mu.Lock()
	srv := a.dnsServer
	a.mu.Unlock()
	if srv == nil {
		return
	}
	cert := srv.BlockPageCACert()
	if len(cert) == 0 {
		return
	}

	tmp, err := os.CreateTemp("", "selfdns-ca-*.pem")
	if err != nil {
		log.Printf("[selfdns-app] CA install: create temp file: %v", err)
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(cert); err != nil {
		tmp.Close()
		return
	}
	tmp.Close()

	switch goruntime.GOOS {
	case "darwin":
		if err := exec.Command("security", "add-trusted-cert",
			"-d", "-r", "trustRoot",
			"-k", "/Library/Keychains/System.keychain",
			tmpName,
		).Run(); err != nil {
			log.Printf("[selfdns-app] CA install: macOS keychain: %v", err)
		} else {
			log.Println("[selfdns-app] CA: installed in macOS system keychain")
		}
	case "windows":
		if err := exec.Command("powershell", "-Command",
			fmt.Sprintf("Import-Certificate -FilePath '%s' -CertStoreLocation Cert:\\LocalMachine\\Root", tmpName),
		).Run(); err != nil {
			log.Printf("[selfdns-app] CA install: Windows cert store: %v", err)
		} else {
			log.Println("[selfdns-app] CA: installed in Windows LocalMachine Root")
		}
	case "linux":
		// Try Debian/Ubuntu, then RHEL/Fedora — both require root which we have.
		debDest := "/usr/local/share/ca-certificates/selfdns-ca.crt"
		if err := exec.Command("cp", tmpName, debDest).Run(); err == nil {
			_ = exec.Command("update-ca-certificates").Run()
			log.Println("[selfdns-app] CA: installed via update-ca-certificates")
		} else if err := exec.Command("cp", tmpName,
			"/etc/pki/ca-trust/source/anchors/selfdns-ca.pem",
		).Run(); err == nil {
			_ = exec.Command("update-ca-trust").Run()
			log.Println("[selfdns-app] CA: installed via update-ca-trust")
		}
	}
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

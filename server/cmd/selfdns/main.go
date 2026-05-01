package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/belsia-dev/Self-DNS/server/api"
	"github.com/belsia-dev/Self-DNS/server/blocker"
	"github.com/belsia-dev/Self-DNS/server/cache"
	"github.com/belsia-dev/Self-DNS/server/config"
	"github.com/belsia-dev/Self-DNS/server/dns"
	"github.com/belsia-dev/Self-DNS/server/resolver"
	"github.com/belsia-dev/Self-DNS/server/stats"
)

const Version = "1.0.0"

func main() {
	configPath := flag.String("config", defaultConfigPath(), "path to config.yaml")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("selfdns %s\n", Version)
		os.Exit(0)
	}

	log.SetFlags(log.Ldate | log.Ltime)
	log.SetPrefix("[selfdns] ")
	log.Printf("SelfDNS v%s starting", Version)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	st := stats.New()
	ch := cache.New(cfg.Cache.MaxSize, cfg.Cache.MinTTL)
	bl := blocker.New(cfg.Blocklist)
	res := resolver.New(cfg, ch, st)

	srv, err := dns.New(cfg, res, bl, st, ch)
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("server start: %v", err)
	}
	log.Printf("DNS server listening on %s", cfg.Listen)

	apiSrv := api.New(cfg, srv, bl, st, ch, *configPath, Version)
	go func() {
		if err := apiSrv.Start(); err != nil {
			log.Printf("API server stopped: %v", err)
		}
	}()
	log.Printf("Control Center API on http://%s", cfg.APIListen)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down gracefully…")
	apiSrv.Stop()
	srv.Stop()
	res.Stop()
	log.Println("SelfDNS stopped.")
}

func defaultConfigPath() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home + "/.config/selfdns/config.yaml"
	}
	return "config.yaml"
}

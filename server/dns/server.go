package dns

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/belsia-dev/Self-DNS/server/blocker"
	"github.com/belsia-dev/Self-DNS/server/cache"
	"github.com/belsia-dev/Self-DNS/server/config"
	"github.com/belsia-dev/Self-DNS/server/resolver"
	"github.com/belsia-dev/Self-DNS/server/stats"
	"github.com/miekg/dns"
)

type Server struct {
	mu       sync.RWMutex
	cfg      *config.Config
	resolver *resolver.Resolver
	blocker  *blocker.Blocker
	stats    *stats.Stats
	cache    *cache.Cache
	limiter  *rateLimiter

	udp *dns.Server
	tcp *dns.Server

	running   atomic.Bool
	startTime time.Time
}

func New(
	cfg *config.Config,
	res *resolver.Resolver,
	bl *blocker.Blocker,
	st *stats.Stats,
	ch *cache.Cache,
) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	s := &Server{
		cfg:      cfg,
		resolver: res,
		blocker:  bl,
		stats:    st,
		cache:    ch,
		limiter:  newRateLimiter(cfg.RateLimit),
	}
	return s, nil
}

func (s *Server) Resolver() *resolver.Resolver { return s.resolver }

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return fmt.Errorf("server already running")
	}

	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handle)

	s.udp = &dns.Server{Addr: s.cfg.Listen, Net: "udp", Handler: mux, UDPSize: 4096}
	s.tcp = &dns.Server{Addr: s.cfg.Listen, Net: "tcp", Handler: mux}

	udpReady := make(chan struct{})
	tcpReady := make(chan struct{})
	s.udp.NotifyStartedFunc = func() { close(udpReady) }
	s.tcp.NotifyStartedFunc = func() { close(tcpReady) }

	errCh := make(chan error, 2)
	go func() {
		if err := s.udp.ListenAndServe(); err != nil && s.running.Load() {
			errCh <- fmt.Errorf("UDP: %w", err)
		}
	}()
	go func() {
		if err := s.tcp.ListenAndServe(); err != nil && s.running.Load() {
			errCh <- fmt.Errorf("TCP: %w", err)
		}
	}()

	timeout := time.After(5 * time.Second)
	for ready := 0; ready < 2; {
		select {
		case <-udpReady:
			ready++
			udpReady = nil
		case <-tcpReady:
			ready++
			tcpReady = nil
		case err := <-errCh:
			return err
		case <-timeout:
			return fmt.Errorf("listeners did not start within 5s")
		}
	}

	s.startTime = time.Now()
	s.running.Store(true)
	return nil
}

func (s *Server) Stop() {
	s.running.Store(false)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.udp != nil {
		_ = s.udp.Shutdown()
	}
	if s.tcp != nil {
		_ = s.tcp.Shutdown()
	}
}

func (s *Server) IsRunning() bool { return s.running.Load() }

func (s *Server) StartTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startTime
}

func (s *Server) Config() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Server) Reload(cfg *config.Config) error {
	s.Stop()
	s.mu.Lock()
	s.cfg = cfg
	s.limiter = newRateLimiter(cfg.RateLimit)
	s.mu.Unlock()
	s.resolver.UpdateConfig(cfg)
	return s.Start()
}

func (s *Server) handle(w dns.ResponseWriter, req *dns.Msg) {
	start := time.Now()

	if req == nil || len(req.Question) == 0 {
		return
	}

	q := req.Question[0]
	domain := q.Name
	qtype := dns.TypeToString[q.Qtype]

	switch q.Qtype {
	case dns.TypeANY, dns.TypeAXFR, dns.TypeIXFR:
		writeRcode(w, req, dns.RcodeRefused)
		s.record(domain, qtype, stats.ResultError, "", time.Since(start), false)
		return
	}

	s.mu.RLock()
	limitEnabled := s.cfg.RateLimit.Enabled
	s.mu.RUnlock()

	if limitEnabled && !s.limiter.allow(extractIP(w.RemoteAddr()), domain) {
		writeRcode(w, req, dns.RcodeRefused)
		return
	}

	s.mu.RLock()
	hosts := s.cfg.Hosts
	logQ := s.cfg.LogQueries
	s.mu.RUnlock()

	if ip, ok := lookupHost(hosts, domain); ok {
		resp := hostResponse(req, ip)
		_ = w.WriteMsg(resp)
		s.record(domain, qtype, stats.ResultResolved, "", time.Since(start), logQ)
		return
	}

	if s.blocker.IsBlocked(domain) {
		writeRcode(w, req, dns.RcodeNameError)
		s.record(domain, qtype, stats.ResultBlocked, "", time.Since(start), logQ)
		return
	}

	resp, upstream, latency, err := s.resolver.Resolve(req)
	if err != nil {
		log.Printf("resolve %s: %v", domain, err)
		writeRcode(w, req, dns.RcodeServerFailure)
		s.record(domain, qtype, stats.ResultError, "", latency, logQ)
		return
	}

	result := stats.ResultResolved
	if upstream == "cache" {
		result = stats.ResultCached
	}

	_ = w.WriteMsg(resp)
	s.record(domain, qtype, result, upstream, latency, logQ)
}

func (s *Server) record(domain, qtype string, result stats.QueryResult, upstream string, latency time.Duration, logFull bool) {
	d := domain
	if !logFull {
		d = anonymise(domain)
	}
	s.stats.RecordQuery(stats.QueryEntry{
		Timestamp: time.Now(),
		Domain:    d,
		Type:      qtype,
		Result:    result,
		LatencyMs: float64(latency.Microseconds()) / 1000.0,
		Upstream:  upstream,
	})
}

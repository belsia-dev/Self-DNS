package resolver

import (
	"crypto/tls"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/belsia-dev/Self-DNS/server/cache"
	"github.com/belsia-dev/Self-DNS/server/config"
	"github.com/belsia-dev/Self-DNS/server/stats"
	"github.com/miekg/dns"
)

const (
	dialTimeout      = 2 * time.Second
	queryTimeout     = 2 * time.Second
	connMaxAge       = 55 * time.Second
	maxIdlePer       = 8
	prefetchEvery    = 45 * time.Second
	prefetchHeadroom = 30 * time.Second
	prefetchTopN     = 20
	failureCutoff    = 5
)

type upstreamStat struct {
	emaMicro   int64
	failStreak int64
}

type Resolver struct {
	cfg     *config.Config
	cache   *cache.Cache
	stats   *stats.Stats
	dotPool *dotPool

	mu         sync.RWMutex
	systemDNS  []string
	networkDNS []string

	health     sync.Map
	refreshing sync.Map

	stop chan struct{}
}

func New(cfg *config.Config, ch *cache.Cache, st *stats.Stats) *Resolver {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	r := &Resolver{
		cfg:     cfg,
		cache:   ch,
		stats:   st,
		dotPool: newDotPool(tlsCfg),
		stop:    make(chan struct{}),
	}
	r.systemDNS = discoverSystemDNS()
	r.networkDNS = discoverNetworkDNS()
	go r.warmPool()
	go r.prefetchLoop()
	return r
}

func (r *Resolver) warmPool() {
	r.mu.RLock()
	upstreams := append([]string(nil), r.cfg.Upstream...)
	useTLS := r.cfg.UseTLS
	r.mu.RUnlock()

	if !useTLS {
		return
	}
	var wg sync.WaitGroup
	for _, s := range upstreams {
		wg.Add(1)
		go func(srv string) {
			defer wg.Done()
			pc, err := r.dotPool.dial(srv)
			if err == nil {
				r.dotPool.release(pc)
			}
		}(s)
	}
	wg.Wait()
}

func (r *Resolver) Resolve(req *dns.Msg) (*dns.Msg, string, time.Duration, error) {
	if len(req.Question) == 0 {
		return nil, "", 0, fmt.Errorf("empty question section")
	}
	q := req.Question[0]

	r.mu.RLock()
	cacheEnabled := r.cfg.Cache.Enabled
	swr := r.cfg.Cache.StaleWhileRevalidate
	r.mu.RUnlock()

	if cacheEnabled {
		if cached, found, isStale := r.cache.Get(q.Name, q.Qtype, q.Qclass); found {
			cached.Id = req.Id
			if isStale && swr {
				go r.backgroundRefresh(req.Copy())
			}
			return cached, "cache", 0, nil
		}
		r.stats.RecordCacheMiss()
	}

	start := time.Now()
	resp, upstream, err := r.queryUpstreamRace(req)
	latency := time.Since(start)
	if err != nil {
		return nil, "", latency, err
	}

	r.mu.RLock()
	rebind := r.cfg.DNSRebindingProtection
	r.mu.RUnlock()

	if rebind && r.isRebinding(q.Name, resp) {
		return nxDomain(req), upstream, latency, nil
	}

	if cacheEnabled {
		r.cache.Set(resp)
	}

	resp.Id = req.Id
	return resp, upstream, latency, nil
}

func (r *Resolver) UpdateConfig(cfg *config.Config) {
	r.mu.Lock()
	r.cfg = cfg
	r.mu.Unlock()
	go r.warmPool()
}

func (r *Resolver) prefetchLoop() {
	t := time.NewTicker(prefetchEvery)
	defer t.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-t.C:
			r.prefetchHot()
		}
	}
}

func (r *Resolver) PrefetchNow() { r.prefetchHot() }

func (r *Resolver) prefetchHot() {
	r.mu.RLock()
	cacheEnabled := r.cfg.Cache.Enabled
	r.mu.RUnlock()
	if !cacheEnabled {
		return
	}
	hot := r.cache.Hot(prefetchTopN)
	for _, h := range hot {
		if time.Duration(h.TTLSec)*time.Second > prefetchHeadroom {
			continue
		}
		req := new(dns.Msg)
		req.SetQuestion(h.Name, h.Qtype)
		req.Question[0].Qclass = h.Qclass
		go r.backgroundRefresh(req)
	}
}

func (r *Resolver) Stop() {
	select {
	case <-r.stop:
	default:
		close(r.stop)
	}
	if r.dotPool != nil {
		r.dotPool.Stop()
	}
}

func (r *Resolver) sortedUpstreams(upstreams []string) []string {
	type scored struct {
		srv  string
		ema  int64
		fail int64
	}
	all := make([]scored, 0, len(upstreams))
	for _, s := range upstreams {
		v, _ := r.health.LoadOrStore(s, &upstreamStat{})
		st := v.(*upstreamStat)
		all = append(all, scored{srv: s, ema: atomic.LoadInt64(&st.emaMicro), fail: atomic.LoadInt64(&st.failStreak)})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].fail != all[j].fail {
			return all[i].fail < all[j].fail
		}
		if all[i].ema == 0 && all[j].ema != 0 {
			return true
		}
		if all[j].ema == 0 && all[i].ema != 0 {
			return false
		}
		return all[i].ema < all[j].ema
	})
	out := make([]string, 0, len(all))
	for _, s := range all {
		if s.fail >= failureCutoff {
			continue
		}
		out = append(out, s.srv)
	}
	if len(out) == 0 {
		return upstreams
	}
	return out
}

type UpstreamHealth struct {
	Server     string  `json:"server"`
	AvgMs      float64 `json:"avg_ms"`
	FailStreak int64   `json:"fail_streak"`
}

func (r *Resolver) UpstreamsHealth() []UpstreamHealth {
	r.mu.RLock()
	upstreams := append([]string(nil), r.cfg.Upstream...)
	r.mu.RUnlock()
	out := make([]UpstreamHealth, 0, len(upstreams))
	for _, s := range upstreams {
		var avg float64
		var fail int64
		if v, ok := r.health.Load(s); ok {
			st := v.(*upstreamStat)
			avg = float64(atomic.LoadInt64(&st.emaMicro)) / 1000.0
			fail = atomic.LoadInt64(&st.failStreak)
		}
		out = append(out, UpstreamHealth{Server: s, AvgMs: avg, FailStreak: fail})
	}
	return out
}

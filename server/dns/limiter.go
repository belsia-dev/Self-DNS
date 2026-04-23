package dns

import (
	"sync"
	"time"

	"github.com/belsia-dev/Self-DNS/server/config"
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

type rateLimiter struct {
	mu            sync.Mutex
	maxRPS        float64
	burstSize     float64
	whitelist     map[string]struct{}
	ipBuckets     map[string]*tokenBucket
	domainMaxRPS  float64
	domainBuckets map[string]*tokenBucket
}

func newRateLimiter(cfg config.RateLimitConfig) *rateLimiter {
	burst := float64(cfg.MaxRPS) * float64(cfg.BurstMultiplier)
	if burst < float64(cfg.MaxRPS) {
		burst = float64(cfg.MaxRPS)
	}

	whitelist := make(map[string]struct{}, len(cfg.WhitelistIPs))
	for _, ip := range cfg.WhitelistIPs {
		whitelist[ip] = struct{}{}
	}

	r := &rateLimiter{
		maxRPS:        float64(cfg.MaxRPS),
		burstSize:     burst,
		whitelist:     whitelist,
		ipBuckets:     make(map[string]*tokenBucket),
		domainMaxRPS:  float64(cfg.PerDomainMaxRPS),
		domainBuckets: make(map[string]*tokenBucket),
	}
	go r.cleanup()
	return r
}

func (r *rateLimiter) allow(ip, domain string) bool {
	if _, exempt := r.whitelist[ip]; exempt {
		return true
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.consumeToken(r.ipBuckets, ip, r.maxRPS, r.burstSize) {
		return false
	}

	if r.domainMaxRPS > 0 && domain != "" {
		domBurst := r.domainMaxRPS * 2
		if !r.consumeToken(r.domainBuckets, domain, r.domainMaxRPS, domBurst) {
			return false
		}
	}

	return true
}

func (r *rateLimiter) consumeToken(buckets map[string]*tokenBucket, key string, rps, burst float64) bool {
	b, ok := buckets[key]
	if !ok {
		b = &tokenBucket{tokens: burst, lastRefill: time.Now()}
		buckets[key] = b
	}

	now := time.Now()
	b.tokens += now.Sub(b.lastRefill).Seconds() * rps
	if b.tokens > burst {
		b.tokens = burst
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (r *rateLimiter) cleanup() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		cutoff := time.Now().Add(-5 * time.Minute)
		r.mu.Lock()
		for ip, b := range r.ipBuckets {
			if b.lastRefill.Before(cutoff) {
				delete(r.ipBuckets, ip)
			}
		}
		for dom, b := range r.domainBuckets {
			if b.lastRefill.Before(cutoff) {
				delete(r.domainBuckets, dom)
			}
		}
		r.mu.Unlock()
	}
}

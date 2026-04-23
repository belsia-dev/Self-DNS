package resolver

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

func (r *Resolver) backgroundRefresh(req *dns.Msg) {
	if len(req.Question) == 0 {
		return
	}
	q := req.Question[0]
	key := q.Name + "\x00" + dns.TypeToString[q.Qtype]

	if _, already := r.refreshing.LoadOrStore(key, struct{}{}); already {
		return
	}
	defer r.refreshing.Delete(key)

	resp, _, err := r.queryUpstreamRace(req)
	if err != nil {
		return
	}

	r.mu.RLock()
	cacheEnabled := r.cfg.Cache.Enabled
	rebind := r.cfg.DNSRebindingProtection
	r.mu.RUnlock()

	if rebind && r.isRebinding(q.Name, resp) {
		return
	}
	if cacheEnabled {
		r.cache.Set(resp)
	}
}

func (r *Resolver) queryUpstreamRace(req *dns.Msg) (*dns.Msg, string, error) {
	r.mu.RLock()
	configuredUpstreams := append([]string(nil), r.cfg.Upstream...)
	systemDNS := append([]string(nil), r.systemDNS...)
	networkDNS := append([]string(nil), r.networkDNS...)
	useTLS := r.cfg.UseTLS
	dnssec := r.cfg.DNSSEC
	r.mu.RUnlock()

	udpReq := req.Copy()

	dotReq := req.Copy()
	if dnssec {
		dotReq.SetEdns0(4096, true)
	}

	var lastErr error

	for _, s := range systemDNS {
		c := &dns.Client{Net: "udp", Timeout: queryTimeout}
		resp, _, err := c.Exchange(udpReq, s)
		if err == nil && resp != nil {
			return resp, "system:" + s, nil
		}
		lastErr = err
	}

	for _, s := range networkDNS {
		c := &dns.Client{Net: "udp", Timeout: queryTimeout}
		resp, _, err := c.Exchange(udpReq, s)
		if err == nil && resp != nil {
			return resp, "network:" + s, nil
		}
		lastErr = err
	}

	upstreams := r.sortedUpstreams(configuredUpstreams)
	if len(upstreams) == 0 {
		if lastErr == nil {
			lastErr = fmt.Errorf("no DNS servers available")
		}
		return nil, "", fmt.Errorf("all tiers failed: %w", lastErr)
	}

	type result struct {
		resp   *dns.Msg
		server string
		err    error
	}

	ch := make(chan result, len(upstreams))

	for _, s := range upstreams {
		go func(srv string) {
			t0 := time.Now()
			resp, err := r.exchange(srv, dotReq.Copy(), useTLS)
			lat := time.Since(t0)
			ch <- result{resp, srv, err}
			if err != nil {
				r.bumpFailures(srv)
			} else {
				r.recordSuccess(srv, lat)
			}
		}(s)
	}

	timer := time.NewTimer(queryTimeout)
	defer timer.Stop()

	for received := 0; received < len(upstreams); received++ {
		select {
		case res := <-ch:
			if res.err == nil && res.resp != nil {
				return res.resp, res.server, nil
			}
			lastErr = res.err
		case <-timer.C:
			if lastErr == nil {
				lastErr = fmt.Errorf("upstream timeout")
			}
			return nil, "", fmt.Errorf("all tiers failed: %w", lastErr)
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no upstreams available")
	}
	return nil, "", fmt.Errorf("all tiers failed: %w", lastErr)
}

func (r *Resolver) exchange(server string, req *dns.Msg, useTLS bool) (*dns.Msg, error) {
	if useTLS {
		return r.exchangeDoT(server, req)
	}
	c := &dns.Client{Net: "udp", Timeout: queryTimeout}
	resp, _, err := c.Exchange(req, server)
	return resp, err
}

func (r *Resolver) exchangeDoT(server string, req *dns.Msg) (*dns.Msg, error) {
	for attempt := 0; attempt < 2; attempt++ {
		pc, err := r.dotPool.acquire(server)
		if err != nil {
			return nil, err
		}

		_ = pc.conn.SetDeadline(time.Now().Add(queryTimeout))

		if err := pc.conn.WriteMsg(req); err != nil {
			r.dotPool.discard(pc)
			continue
		}
		resp, err := pc.conn.ReadMsg()
		if err != nil {
			r.dotPool.discard(pc)
			continue
		}

		r.dotPool.release(pc)
		return resp, nil
	}
	return nil, fmt.Errorf("DoT %s: failed after retry", server)
}

func (r *Resolver) bumpFailures(server string) {
	v, _ := r.health.LoadOrStore(server, &upstreamStat{})
	atomic.AddInt64(&v.(*upstreamStat).failStreak, 1)
}

func (r *Resolver) recordSuccess(server string, latency time.Duration) {
	v, _ := r.health.LoadOrStore(server, &upstreamStat{})
	st := v.(*upstreamStat)
	atomic.StoreInt64(&st.failStreak, 0)
	micro := latency.Microseconds()
	for {
		cur := atomic.LoadInt64(&st.emaMicro)
		var next int64
		if cur == 0 {
			next = micro
		} else {
			next = (cur*7 + micro) / 8
		}
		if atomic.CompareAndSwapInt64(&st.emaMicro, cur, next) {
			return
		}
	}
}

func (r *Resolver) Failures(server string) int64 {
	if v, ok := r.health.Load(server); ok {
		return atomic.LoadInt64(&v.(*upstreamStat).failStreak)
	}
	return 0
}

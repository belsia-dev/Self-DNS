package stats

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxQueryLog  = 500
	qpsWindowSec = 60
)

type QueryResult string

const (
	ResultResolved QueryResult = "RESOLVED"
	ResultBlocked  QueryResult = "BLOCKED"
	ResultCached   QueryResult = "CACHED"
	ResultError    QueryResult = "ERROR"
)

type QueryEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	Domain    string      `json:"domain"`
	Type      string      `json:"type"`
	Result    QueryResult `json:"result"`
	LatencyMs float64     `json:"latency_ms"`
	Upstream  string      `json:"upstream"`
}

type TopEntry struct {
	Domain string `json:"domain"`
	Count  int64  `json:"count"`
}

type Snapshot struct {
	TotalQueries  int64      `json:"total_queries"`
	TotalBlocked  int64      `json:"total_blocked"`
	TotalResolved int64      `json:"total_resolved"`
	TotalCached   int64      `json:"total_cached"`
	TotalErrors   int64      `json:"total_errors"`
	QueriesPerSec float64    `json:"queries_per_sec"`
	CacheHitRate  float64    `json:"cache_hit_rate"`
	AvgLatencyMs  float64    `json:"avg_latency_ms"`
	TopDomains    []TopEntry `json:"top_domains"`
	TopBlocked    []TopEntry `json:"top_blocked"`
	StartTime     time.Time  `json:"start_time"`
}

type UpstreamStats struct {
	mu       sync.Mutex
	totalMs  float64
	count    int64
	failures int64
}

func (u *UpstreamStats) Record(latencyMs float64, ok bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if ok {
		u.totalMs += latencyMs
		u.count++
	} else {
		u.failures++
	}
}

func (u *UpstreamStats) AvgMs() float64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.count == 0 {
		return 0
	}
	return u.totalMs / float64(u.count)
}

type domainCounter struct {
	mu     sync.RWMutex
	counts map[string]int64
}

func newDomainCounter() *domainCounter {
	return &domainCounter{counts: make(map[string]int64)}
}

func (d *domainCounter) Inc(domain string) {
	d.mu.Lock()
	d.counts[domain]++
	d.mu.Unlock()
}

func (d *domainCounter) Top(n int) []TopEntry {
	d.mu.RLock()
	entries := make([]TopEntry, 0, len(d.counts))
	for domain, count := range d.counts {
		entries = append(entries, TopEntry{Domain: domain, Count: count})
	}
	d.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})
	if len(entries) > n {
		entries = entries[:n]
	}
	return entries
}

type Stats struct {
	totalQueries   atomic.Int64
	totalBlocked   atomic.Int64
	totalResolved  atomic.Int64
	totalCached    atomic.Int64
	totalErrors    atomic.Int64
	totalCacheMiss atomic.Int64

	latencySum   atomic.Int64
	latencyCount atomic.Int64

	qpsRing [qpsWindowSec]int64
	qpsMu   sync.Mutex
	qpsIdx  int
	qpsLast time.Time

	domainHits  *domainCounter
	blockedHits *domainCounter

	upstreams sync.Map

	queryLog    []QueryEntry
	queryLogMu  sync.RWMutex
	queryLogPos int

	startTime time.Time
}

func New() *Stats {
	return &Stats{
		queryLog:    make([]QueryEntry, 0, maxQueryLog),
		domainHits:  newDomainCounter(),
		blockedHits: newDomainCounter(),
		startTime:   time.Now(),
		qpsLast:     time.Now(),
	}
}

func (s *Stats) RecordQuery(entry QueryEntry) {
	s.totalQueries.Add(1)
	s.tickQPS()

	switch entry.Result {
	case ResultBlocked:
		s.totalBlocked.Add(1)
		s.blockedHits.Inc(entry.Domain)
	case ResultCached:
		s.totalCached.Add(1)
		s.totalResolved.Add(1)
	case ResultResolved:
		s.totalResolved.Add(1)
	case ResultError:
		s.totalErrors.Add(1)
	}

	if entry.Result != ResultBlocked && entry.Result != ResultError {
		s.domainHits.Inc(entry.Domain)
	}

	latencyUs := int64(entry.LatencyMs * 1000)
	s.latencySum.Add(latencyUs)
	s.latencyCount.Add(1)

	if entry.Upstream != "" {
		v, _ := s.upstreams.LoadOrStore(entry.Upstream, &UpstreamStats{})
		v.(*UpstreamStats).Record(entry.LatencyMs, entry.Result != ResultError)
	}

	s.queryLogMu.Lock()
	if len(s.queryLog) < maxQueryLog {
		s.queryLog = append(s.queryLog, entry)
	} else {
		s.queryLog[s.queryLogPos%maxQueryLog] = entry
		s.queryLogPos++
	}
	s.queryLogMu.Unlock()
}

func (s *Stats) RecordCacheMiss() {
	s.totalCacheMiss.Add(1)
}

func (s *Stats) tickQPS() {
	now := time.Now()
	s.qpsMu.Lock()
	defer s.qpsMu.Unlock()

	elapsed := int(now.Sub(s.qpsLast).Seconds())
	if elapsed > 0 {
		for i := 1; i <= elapsed && i <= qpsWindowSec; i++ {
			s.qpsRing[(s.qpsIdx+i)%qpsWindowSec] = 0
		}
		s.qpsIdx = (s.qpsIdx + elapsed) % qpsWindowSec
		s.qpsLast = s.qpsLast.Add(time.Duration(elapsed) * time.Second)
	}
	s.qpsRing[s.qpsIdx]++
}

func (s *Stats) QPS() float64 {
	s.qpsMu.Lock()
	defer s.qpsMu.Unlock()
	var total int64
	for _, v := range s.qpsRing {
		total += v
	}
	return float64(total) / float64(qpsWindowSec)
}

func (s *Stats) QPSHistory() []int64 {
	s.qpsMu.Lock()
	defer s.qpsMu.Unlock()
	out := make([]int64, qpsWindowSec)
	for i := 0; i < qpsWindowSec; i++ {
		out[i] = s.qpsRing[(s.qpsIdx+1+i)%qpsWindowSec]
	}
	return out
}

func (s *Stats) Snapshot() Snapshot {
	total := s.totalQueries.Load()
	cached := s.totalCached.Load()
	missed := s.totalCacheMiss.Load()

	var hitRate float64
	if total := cached + missed; total > 0 {
		hitRate = float64(cached) / float64(total) * 100
	}

	var avgMs float64
	if cnt := s.latencyCount.Load(); cnt > 0 {
		avgMs = float64(s.latencySum.Load()) / float64(cnt) / 1000.0
	}

	return Snapshot{
		TotalQueries:  total,
		TotalBlocked:  s.totalBlocked.Load(),
		TotalResolved: s.totalResolved.Load(),
		TotalCached:   cached,
		TotalErrors:   s.totalErrors.Load(),
		QueriesPerSec: s.QPS(),
		CacheHitRate:  hitRate,
		AvgLatencyMs:  avgMs,
		TopDomains:    s.domainHits.Top(10),
		TopBlocked:    s.blockedHits.Top(10),
		StartTime:     s.startTime,
	}
}

func (s *Stats) Queries(n int) []QueryEntry {
	s.queryLogMu.RLock()
	defer s.queryLogMu.RUnlock()

	log := make([]QueryEntry, len(s.queryLog))
	copy(log, s.queryLog)

	for i, j := 0, len(log)-1; i < j; i, j = i+1, j-1 {
		log[i], log[j] = log[j], log[i]
	}
	if n > 0 && len(log) > n {
		log = log[:n]
	}
	return log
}

func (s *Stats) StartTime() time.Time {
	return s.startTime
}

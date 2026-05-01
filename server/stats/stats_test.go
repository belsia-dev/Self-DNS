package stats

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func makeEntry(domain string, result QueryResult, latencyMs float64, upstream string) QueryEntry {
	return QueryEntry{
		Timestamp: time.Now(),
		Domain:    domain,
		Type:      "A",
		Result:    result,
		LatencyMs: latencyMs,
		Upstream:  upstream,
	}
}

func TestNewCreatesZeroState(t *testing.T) {
	s := New()

	if s.totalQueries.Load() != 0 {
		t.Error("totalQueries should be 0")
	}
	if s.totalBlocked.Load() != 0 {
		t.Error("totalBlocked should be 0")
	}
	if s.totalCached.Load() != 0 {
		t.Error("totalCached should be 0")
	}
	if s.totalErrors.Load() != 0 {
		t.Error("totalErrors should be 0")
	}
	if s.totalResolved.Load() != 0 {
		t.Error("totalResolved should be 0")
	}
	if s.startTime.IsZero() {
		t.Error("startTime should be set")
	}
}

func TestRecordIncrementsCounters(t *testing.T) {
	s := New()

	s.RecordQuery(makeEntry("blocked.com", ResultBlocked, 0, ""))
	s.RecordQuery(makeEntry("cached.com", ResultCached, 0.5, ""))
	s.RecordQuery(makeEntry("resolved.com", ResultResolved, 10, "1.1.1.1"))
	s.RecordQuery(makeEntry("error.com", ResultError, 0, "8.8.8.8"))

	if s.totalQueries.Load() != 4 {
		t.Errorf("totalQueries = %d, want 4", s.totalQueries.Load())
	}
	if s.totalBlocked.Load() != 1 {
		t.Errorf("totalBlocked = %d, want 1", s.totalBlocked.Load())
	}
	if s.totalCached.Load() != 1 {
		t.Errorf("totalCached = %d, want 1", s.totalCached.Load())
	}
	if s.totalResolved.Load() != 2 { // Cached + Resolved
		t.Errorf("totalResolved = %d, want 2", s.totalResolved.Load())
	}
	if s.totalErrors.Load() != 1 {
		t.Errorf("totalErrors = %d, want 1", s.totalErrors.Load())
	}
}

func TestRecordDomainHitsExcludesBlockedAndError(t *testing.T) {
	s := New()

	s.RecordQuery(makeEntry("ok.com", ResultResolved, 1, ""))
	s.RecordQuery(makeEntry("blocked.com", ResultBlocked, 0, ""))
	s.RecordQuery(makeEntry("err.com", ResultError, 0, ""))

	top := s.domainHits.Top(10)
	names := make(map[string]bool)
	for _, e := range top {
		names[e.Domain] = true
	}
	if !names["ok.com"] {
		t.Error("ok.com should be in domain hits")
	}
	if names["blocked.com"] {
		t.Error("blocked.com should NOT be in domain hits")
	}
	if names["err.com"] {
		t.Error("err.com should NOT be in domain hits")
	}
}

func TestQueriesEmpty(t *testing.T) {
	s := New()
	queries := s.Queries(0)
	if len(queries) != 0 {
		t.Errorf("expected 0 queries, got %d", len(queries))
	}
}

func TestQueriesBeforeWrap(t *testing.T) {
	s := New()

	for i := 0; i < 10; i++ {
		s.RecordQuery(makeEntry(fmt.Sprintf("test%d.com", i), ResultResolved, 1, ""))
	}

	queries := s.Queries(0)
	if len(queries) != 10 {
		t.Fatalf("expected 10 queries, got %d", len(queries))
	}

	// Newest first.
	if queries[0].Domain != "test9.com" {
		t.Errorf("first entry = %q, want test9.com (newest first)", queries[0].Domain)
	}
	if queries[9].Domain != "test0.com" {
		t.Errorf("last entry = %q, want test0.com (oldest last)", queries[9].Domain)
	}
}

func TestQueriesBeforeWrapWithLimit(t *testing.T) {
	s := New()

	for i := 0; i < 20; i++ {
		s.RecordQuery(makeEntry(fmt.Sprintf("test%d.com", i), ResultResolved, 1, ""))
	}

	queries := s.Queries(5)
	if len(queries) != 5 {
		t.Fatalf("expected 5 queries, got %d", len(queries))
	}
	if queries[0].Domain != "test19.com" {
		t.Errorf("first = %q, want test19.com", queries[0].Domain)
	}
}

func TestQueriesAfterWrap(t *testing.T) {
	s := New()

	totalRecords := 600
	for i := 0; i < totalRecords; i++ {
		s.RecordQuery(makeEntry(fmt.Sprintf("test%d.com", i), ResultResolved, 1, ""))
	}

	queries := s.Queries(0)
	if len(queries) != maxQueryLog {
		t.Fatalf("expected %d queries, got %d", maxQueryLog, len(queries))
	}

	// First entry should be the newest (test599.com).
	if !strings.HasPrefix(queries[0].Domain, "test599") {
		t.Errorf("expected newest first (test599.com), got %s", queries[0].Domain)
	}

	// Last entry should be the oldest among the retained 500 (test100.com).
	if !strings.HasPrefix(queries[maxQueryLog-1].Domain, "test100") {
		t.Errorf("expected oldest last (test100.com), got %s", queries[maxQueryLog-1].Domain)
	}

	// Verify descending order.
	for i := 1; i < len(queries); i++ {
		prev := queries[i-1].Domain
		curr := queries[i].Domain
		if prev == curr {
			continue // shouldn't happen, but avoid false positive
		}
		if prev < curr {
			t.Errorf("queries not in descending order: %s before %s at index %d", prev, curr, i)
			break
		}
	}
}

func TestQueriesAfterWrapWithLimit(t *testing.T) {
	s := New()

	for i := 0; i < 600; i++ {
		s.RecordQuery(makeEntry(fmt.Sprintf("test%d.com", i), ResultResolved, 1, ""))
	}

	queries := s.Queries(10)
	if len(queries) != 10 {
		t.Fatalf("expected 10 queries, got %d", len(queries))
	}
	if !strings.HasPrefix(queries[0].Domain, "test599") {
		t.Errorf("expected test599.com first, got %s", queries[0].Domain)
	}
}

func TestSnapshot(t *testing.T) {
	s := New()

	s.RecordQuery(makeEntry("a.com", ResultResolved, 10, "1.1.1.1"))
	s.RecordQuery(makeEntry("b.com", ResultBlocked, 0, ""))
	s.RecordQuery(makeEntry("c.com", ResultCached, 1, ""))
	s.RecordQuery(makeEntry("d.com", ResultError, 0, "8.8.8.8"))
	s.RecordCacheMiss()

	snap := s.Snapshot()

	if snap.TotalQueries != 4 {
		t.Errorf("TotalQueries = %d, want 4", snap.TotalQueries)
	}
	if snap.TotalBlocked != 1 {
		t.Errorf("TotalBlocked = %d, want 1", snap.TotalBlocked)
	}
	if snap.TotalCached != 1 {
		t.Errorf("TotalCached = %d, want 1", snap.TotalCached)
	}
	if snap.TotalErrors != 1 {
		t.Errorf("TotalErrors = %d, want 1", snap.TotalErrors)
	}
	if snap.TotalResolved != 2 {
		t.Errorf("TotalResolved = %d, want 2", snap.TotalResolved)
	}

	// AvgLatencyMs: (10 + 0 + 1 + 0) / 4 = 2.75
	if snap.AvgLatencyMs < 2.7 || snap.AvgLatencyMs > 2.8 {
		t.Errorf("AvgLatencyMs = %f, want ~2.75", snap.AvgLatencyMs)
	}

	// CacheHitRate: cached=1 / (cached+miss) = 1/(1+1) = 50%
	if snap.CacheHitRate < 49.9 || snap.CacheHitRate > 50.1 {
		t.Errorf("CacheHitRate = %f, want ~50", snap.CacheHitRate)
	}

	if snap.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestSnapshotTopDomains(t *testing.T) {
	s := New()

	s.RecordQuery(makeEntry("popular.com", ResultResolved, 1, ""))
	s.RecordQuery(makeEntry("popular.com", ResultResolved, 1, ""))
	s.RecordQuery(makeEntry("popular.com", ResultResolved, 1, ""))
	s.RecordQuery(makeEntry("other.com", ResultResolved, 1, ""))

	snap := s.Snapshot()
	if len(snap.TopDomains) == 0 {
		t.Fatal("TopDomains should not be empty")
	}
	if snap.TopDomains[0].Domain != "popular.com" {
		t.Errorf("top domain = %q, want popular.com", snap.TopDomains[0].Domain)
	}
	if snap.TopDomains[0].Count != 3 {
		t.Errorf("top domain count = %d, want 3", snap.TopDomains[0].Count)
	}
}

func TestSnapshotTopBlocked(t *testing.T) {
	s := New()

	s.RecordQuery(makeEntry("ad.com", ResultBlocked, 0, ""))
	s.RecordQuery(makeEntry("ad.com", ResultBlocked, 0, ""))
	s.RecordQuery(makeEntry("tracker.com", ResultBlocked, 0, ""))

	snap := s.Snapshot()
	if len(snap.TopBlocked) == 0 {
		t.Fatal("TopBlocked should not be empty")
	}
	if snap.TopBlocked[0].Domain != "ad.com" {
		t.Errorf("top blocked = %q, want ad.com", snap.TopBlocked[0].Domain)
	}
}

func TestUpstreams(t *testing.T) {
	s := New()

	s.RecordQuery(makeEntry("a.com", ResultResolved, 10, "1.1.1.1:853"))
	s.RecordQuery(makeEntry("b.com", ResultResolved, 20, "1.1.1.1:853"))
	s.RecordQuery(makeEntry("c.com", ResultError, 0, "8.8.8.8:853"))

	v, ok := s.upstreams.Load("1.1.1.1:853")
	if !ok {
		t.Fatal("upstream 1.1.1.1:853 not found")
	}
	us := v.(*UpstreamStats)
	if us.count != 2 {
		t.Errorf("upstream count = %d, want 2", us.count)
	}
	avg := us.AvgMs()
	if avg < 14.9 || avg > 15.1 {
		t.Errorf("upstream avg = %f, want ~15", avg)
	}

	v, ok = s.upstreams.Load("8.8.8.8:853")
	if !ok {
		t.Fatal("upstream 8.8.8.8:853 not found")
	}
	us = v.(*UpstreamStats)
	if us.failures != 1 {
		t.Errorf("upstream failures = %d, want 1", us.failures)
	}
}

func TestUpstreamAvgMsZero(t *testing.T) {
	u := &UpstreamStats{}
	if u.AvgMs() != 0 {
		t.Errorf("AvgMs() = %f, want 0 for zero-count upstream", u.AvgMs())
	}
}

func TestStartTime(t *testing.T) {
	s := New()
	before := time.Now().Add(-1 * time.Second)
	st := s.StartTime()
	after := time.Now()

	if st.Before(before) || st.After(after) {
		t.Errorf("StartTime = %v, expected between %v and %v", st, before, after)
	}
}

func TestRecordCacheMiss(t *testing.T) {
	s := New()
	s.RecordCacheMiss()
	s.RecordCacheMiss()
	if s.totalCacheMiss.Load() != 2 {
		t.Errorf("totalCacheMiss = %d, want 2", s.totalCacheMiss.Load())
	}
}

func TestQPS(t *testing.T) {
	s := New()

	for i := 0; i < 10; i++ {
		s.RecordQuery(makeEntry("qps.com", ResultResolved, 1, ""))
	}

	qps := s.QPS()
	if qps <= 0 {
		t.Errorf("QPS = %f, want > 0", qps)
	}
}

func TestConcurrentRecordAndSnapshot(t *testing.T) {
	s := New()

	var wg sync.WaitGroup
	const writers = 20
	const recordsPerWriter = 50

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < recordsPerWriter; j++ {
				result := ResultResolved
				if j%4 == 0 {
					result = ResultBlocked
				} else if j%4 == 1 {
					result = ResultCached
				} else if j%4 == 2 {
					result = ResultError
				}
				s.RecordQuery(makeEntry(
					fmt.Sprintf("g%d-r%d.com", id, j),
					result,
					float64(j),
					"1.1.1.1",
				))
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < recordsPerWriter; j++ {
				_ = s.Snapshot()
				_ = s.Queries(10)
			}
		}()
	}

	wg.Wait()

	snap := s.Snapshot()
	expectedTotal := int64(writers * recordsPerWriter)
	if snap.TotalQueries != expectedTotal {
		t.Errorf("TotalQueries = %d, want %d", snap.TotalQueries, expectedTotal)
	}

	queries := s.Queries(0)
	if len(queries) == 0 {
		t.Error("Queries() returned empty after concurrent writes")
	}
}

func TestQueriesReturnsCopy(t *testing.T) {
	s := New()

	for i := 0; i < 5; i++ {
		s.RecordQuery(makeEntry(fmt.Sprintf("copy%d.com", i), ResultResolved, 1, ""))
	}

	q1 := s.Queries(0)
	q1[0].Domain = "mutated"

	q2 := s.Queries(0)
	if q2[0].Domain == "mutated" {
		t.Error("mutation of returned slice affected internal state")
	}
}

func TestDomainCounterTop(t *testing.T) {
	d := newDomainCounter()

	d.Inc("a.com")
	d.Inc("a.com")
	d.Inc("a.com")
	d.Inc("b.com")
	d.Inc("b.com")
	d.Inc("c.com")

	top := d.Top(2)
	if len(top) != 2 {
		t.Fatalf("Top(2) returned %d entries", len(top))
	}
	if top[0].Domain != "a.com" || top[0].Count != 3 {
		t.Errorf("top[0] = {%q, %d}, want {a.com, 3}", top[0].Domain, top[0].Count)
	}
	if top[1].Domain != "b.com" || top[1].Count != 2 {
		t.Errorf("top[1] = {%q, %d}, want {b.com, 2}", top[1].Domain, top[1].Count)
	}
}

func TestDomainCounterTopExceedsSize(t *testing.T) {
	d := newDomainCounter()
	for i := 0; i < 20; i++ {
		d.Inc(fmt.Sprintf("d%d.com", i))
	}

	top := d.Top(5)
	if len(top) != 5 {
		t.Errorf("Top(5) returned %d, want 5", len(top))
	}
}

package cache

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// helper: create a minimal DNS message with one A-record answer.
func makeMsg(domain string, rcode int, ttl uint32) *dns.Msg {
	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.Rcode = rcode
	if rcode == dns.RcodeSuccess {
		m.Answer = []dns.RR{&dns.A{
			Hdr: dns.RR_Header{
				Name:   dns.Fqdn(domain),
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: net.ParseIP("1.2.3.4"),
		}}
	}
	return m
}

// helper: create an NXDOMAIN response with SOA for negative TTL.
func makeNXMsg(domain string, soaTTL uint32) *dns.Msg {
	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.Rcode = dns.RcodeNameError
	m.Ns = []dns.RR{&dns.SOA{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(domain),
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    soaTTL,
		},
		Ns:      dns.Fqdn("ns1.example.com."),
		Mbox:    dns.Fqdn("hostmaster.example.com."),
		Serial:  2024010100,
		Refresh: 3600,
		Retry:   900,
		Expire:  604800,
		Minttl:  soaTTL,
	}}
	return m
}

// -------------------------------------------------------------------
// New
// -------------------------------------------------------------------

func TestNewCreatesCache(t *testing.T) {
	c := New(1024, 10)
	defer c.Stop()

	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.maxSize != 1024 {
		t.Errorf("maxSize = %d, want 1024", c.maxSize)
	}
	if c.minTTL != 10 {
		t.Errorf("minTTL = %d, want 10", c.minTTL)
	}
	// Each shard should have maxSize/numShards capacity
	perShard := 1024 / numShards
	for i := range c.shards {
		if c.shards[i].maxSize != perShard {
			t.Errorf("shard[%d].maxSize = %d, want %d", i, c.shards[i].maxSize, perShard)
		}
	}
}

func TestNewDefaultMaxSize(t *testing.T) {
	c := New(0, 5)
	defer c.Stop()
	if c.maxSize != 10000 {
		t.Errorf("expected default maxSize 10000, got %d", c.maxSize)
	}
}

// -------------------------------------------------------------------
// Get/Put basic
// -------------------------------------------------------------------

func TestSetAndGetBasic(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	msg := makeMsg("example.com.", dns.RcodeSuccess, 300)
	c.Set(msg)

	got, found, isStale := c.Get("example.com.", dns.TypeA, dns.ClassINET)
	if !found {
		t.Fatal("expected to find cached entry")
	}
	if isStale {
		t.Error("entry should not be stale")
	}
	if got.Rcode != dns.RcodeSuccess {
		t.Errorf("Rcode = %d, want %d", got.Rcode, dns.RcodeSuccess)
	}
	if len(got.Answer) != 1 {
		t.Fatalf("len(Answer) = %d, want 1", len(got.Answer))
	}
	a := got.Answer[0].(*dns.A)
	if !a.A.Equal(net.ParseIP("1.2.3.4")) {
		t.Errorf("A record = %v, want 1.2.3.4", a.A)
	}
}

// -------------------------------------------------------------------
// Cache miss
// -------------------------------------------------------------------

func TestGetMiss(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	_, found, _ := c.Get("nonexistent.com.", dns.TypeA, dns.ClassINET)
	if found {
		t.Error("expected cache miss, got hit")
	}
}

// -------------------------------------------------------------------
// Set ignores nil and messages without questions
// -------------------------------------------------------------------

func TestSetNilMessage(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	c.Set(nil)
	if c.Len() != 0 {
		t.Error("nil message should not be stored")
	}

	m := &dns.Msg{}
	c.Set(m)
	if c.Len() != 0 {
		t.Error("message without questions should not be stored")
	}
}

// -------------------------------------------------------------------
// Set ignores non-Success/non-NXDOMAIN rcodes
// -------------------------------------------------------------------

func TestSetIgnoresServFail(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	msg := makeMsg("servfail.com.", dns.RcodeServerFailure, 300)
	c.Set(msg)
	if c.Len() != 0 {
		t.Error("SERVFAIL should not be cached")
	}
}

// -------------------------------------------------------------------
// TTL expiry
// -------------------------------------------------------------------

func testTTLExpiry(t *testing.T, ttl uint32, minTTL uint32) {
	c := New(1024, minTTL)
	defer c.Stop()

	msg := makeMsg("expire.com.", dns.RcodeSuccess, ttl)
	c.Set(msg)
	if c.Len() != 1 {
		t.Fatalf("Len = %d, want 1 after Set", c.Len())
	}

	// Wait until after staleUntil expires.
	// staleUntil = expires + staleFraction*TTL (clamped to [10s, 60s]).
	// Use a generous sleep to ensure expiry.
	staleWin := time.Duration(float64(minTTL)*staleFraction) * time.Second
	if staleWin < minStale {
		staleWin = minStale
	}
	if staleWin > maxStale {
		staleWin = maxStale
	}
	wait := time.Duration(minTTL)*time.Second + staleWin + 500*time.Millisecond
	time.Sleep(wait)

	_, found, _ := c.Get("expire.com.", dns.TypeA, dns.ClassINET)
	if found {
		t.Error("entry should have expired")
	}
}

func TestTTLExpiry(t *testing.T) {
	// Use minTTL=1 so the test doesn't take forever.
	// actual TTL will be minTTL (1 second) since msg TTL is 0 and minTTL floors it.
	testTTLExpiry(t, 0, 1)
}

// -------------------------------------------------------------------
// LRU eviction
// -------------------------------------------------------------------

func TestLRUEviction(t *testing.T) {
	// Small cache: 2 per shard * 16 shards = 32 total.
	// We'll target a single shard by using keys that hash to the same shard.
	// Simpler approach: just fill beyond total capacity.
	const totalSize = 32
	c := New(totalSize, 1)
	defer c.Stop()

	// Fill the cache completely.
	for i := 0; i < totalSize; i++ {
		msg := makeMsg(dns.Fqdn("domain"+string(rune('a'+i%26))+string(rune('0'+i/26))+".com"), dns.RcodeSuccess, 300)
		c.Set(msg)
	}
	if c.Len() != totalSize {
		t.Logf("Len after fill = %d (capacity %d), some domains may share shards", c.Len(), totalSize)
	}

	// Add more entries to trigger evictions.
	for i := 0; i < totalSize; i++ {
		msg := makeMsg(dns.Fqdn("extra"+string(rune('a'+i%26))+string(rune('0'+i/26))+".com"), dns.RcodeSuccess, 300)
		c.Set(msg)
	}

	stats := c.Stats()
	if stats.Evictions == 0 {
		t.Error("expected some evictions after overfilling")
	}
}

// -------------------------------------------------------------------
// LRU eviction within a single shard (deterministic)
// -------------------------------------------------------------------

func TestLRUEvictionSingleShard(t *testing.T) {
	c := New(numShards*2, 1) // 2 per shard
	defer c.Stop()

	shard := &c.shards[0]

	msg1 := makeMsg("shard0-a.com.", dns.RcodeSuccess, 300)
	k1 := cacheKey("shard0-a.com.", dns.TypeA, dns.ClassINET)
	shard.set(k1, msg1, 300)

	msg2 := makeMsg("shard0-b.com.", dns.RcodeSuccess, 300)
	k2 := cacheKey("shard0-b.com.", dns.TypeA, dns.ClassINET)
	shard.set(k2, msg2, 300)

	if shard.len() != 2 {
		t.Fatalf("shard len = %d, want 2", shard.len())
	}

	msg3 := makeMsg("shard0-c.com.", dns.RcodeSuccess, 300)
	k3 := cacheKey("shard0-c.com.", dns.TypeA, dns.ClassINET)
	shard.set(k3, msg3, 300)

	shard.mu.Lock()
	evictions := shard.evictions
	items := len(shard.items)
	shard.mu.Unlock()

	if evictions != 1 {
		t.Errorf("evictions = %d, want 1", evictions)
	}
	if items != 2 {
		t.Errorf("items = %d, want 2 (maxSize)", items)
	}
}

// -------------------------------------------------------------------
// Flush
// -------------------------------------------------------------------

func TestFlush(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	for i := 0; i < 50; i++ {
		msg := makeMsg(dns.Fqdn("flush.com"), dns.RcodeSuccess, 300)
		msg.Question[0].Name = dns.Fqdn("flush" + string(rune('a'+i%26)) + ".com")
		c.Set(msg)
	}

	if c.Len() == 0 {
		t.Fatal("expected some entries before flush")
	}

	c.Flush()

	if c.Len() != 0 {
		t.Errorf("Len after Flush = %d, want 0", c.Len())
	}
}

// -------------------------------------------------------------------
// Stop (no panic)
// -------------------------------------------------------------------

func TestStop(t *testing.T) {
	c := New(1024, 1)
	c.Stop()
	// Calling Stop again should not panic (double close protected).
	c.Stop()
}

// -------------------------------------------------------------------
// Len
// -------------------------------------------------------------------

func TestLen(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	if c.Len() != 0 {
		t.Errorf("Len = %d, want 0 initially", c.Len())
	}

	msg := makeMsg("len-test.com.", dns.RcodeSuccess, 300)
	c.Set(msg)
	if c.Len() != 1 {
		t.Errorf("Len = %d, want 1 after one Set", c.Len())
	}
}

// -------------------------------------------------------------------
// Stats
// -------------------------------------------------------------------

func TestStatsCounters(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	// Miss.
	c.Get("miss.com.", dns.TypeA, dns.ClassINET)
	stats := c.Stats()
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}

	// Hit.
	msg := makeMsg("hit.com.", dns.RcodeSuccess, 300)
	c.Set(msg)
	c.Get("hit.com.", dns.TypeA, dns.ClassINET)
	stats = c.Stats()
	if stats.Hits != 1 {
		t.Errorf("Hits = %d, want 1", stats.Hits)
	}
	if stats.Size != 1 {
		t.Errorf("Size = %d, want 1", stats.Size)
	}
	if stats.MaxSize != 1024 {
		t.Errorf("MaxSize = %d, want 1024", stats.MaxSize)
	}

	wantRate := float64(1) / float64(2) * 100 // 1 hit / (1 hit + 1 miss)
	if stats.HitRate < wantRate-0.01 || stats.HitRate > wantRate+0.01 {
		t.Errorf("HitRate = %f, want ~%f", stats.HitRate, wantRate)
	}
}

// -------------------------------------------------------------------
// Delete
// -------------------------------------------------------------------

func TestDelete(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	msg := makeMsg("delete-me.com.", dns.RcodeSuccess, 300)
	c.Set(msg)

	_, found, _ := c.Get("delete-me.com.", dns.TypeA, dns.ClassINET)
	if !found {
		t.Fatal("entry should exist before delete")
	}

	c.Delete("delete-me.com")

	_, found, _ = c.Get("delete-me.com.", dns.TypeA, dns.ClassINET)
	if found {
		t.Error("entry should be gone after delete")
	}
}

func TestDeleteEmpty(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	// Should not panic.
	c.Delete("")
	c.Delete("   ")
}

func TestDeleteNormalizesName(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	msg := makeMsg("Normalize.com.", dns.RcodeSuccess, 300)
	c.Set(msg)

	// Delete without trailing dot — should still work.
	c.Delete("Normalize.com")

	_, found, _ := c.Get("Normalize.com.", dns.TypeA, dns.ClassINET)
	if found {
		t.Error("entry should be removed after normalized delete")
	}
}

// -------------------------------------------------------------------
// NXDOMAIN caching (negative cache)
// -------------------------------------------------------------------

func TestNXDomainCaching(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	msg := makeNXMsg("nx.example.com.", 300)
	c.Set(msg)

	_, found, _ := c.Get("nx.example.com.", dns.TypeA, dns.ClassINET)
	if !found {
		t.Error("NXDOMAIN should be cached")
	}
}

// -------------------------------------------------------------------
// Export / Import round-trip
// -------------------------------------------------------------------

func TestExportImport(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	msg := makeMsg("export.com.", dns.RcodeSuccess, 300)
	c.Set(msg)

	exp := c.Export()
	if exp.Version != 1 {
		t.Errorf("Version = %d, want 1", exp.Version)
	}
	if len(exp.Entries) == 0 {
		t.Fatal("Export returned no entries")
	}

	// Import into a fresh cache.
	c2 := New(1024, 1)
	defer c2.Stop()

	imported := c2.Import(exp)
	if imported == 0 {
		t.Error("expected at least 1 imported entry")
	}

	_, found, _ := c2.Get("export.com.", dns.TypeA, dns.ClassINET)
	if !found {
		t.Error("imported entry should be retrievable")
	}
}

// -------------------------------------------------------------------
// Hot entries
// -------------------------------------------------------------------

func TestHot(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	// Store an entry, access it multiple times.
	msg := makeMsg("hot.com.", dns.RcodeSuccess, 300)
	c.Set(msg)
	for i := 0; i < 5; i++ {
		c.Get("hot.com.", dns.TypeA, dns.ClassINET)
	}

	hot := c.Hot(10)
	if len(hot) == 0 {
		t.Fatal("Hot() returned empty")
	}
	if hot[0].Name != "hot.com." {
		t.Errorf("hot[0].Name = %q, want %q", hot[0].Name, "hot.com.")
	}
	// 5 Get hits.
	if hot[0].Hits < 5 {
		t.Errorf("hot[0].Hits = %d, want >= 5", hot[0].Hits)
	}
}

// -------------------------------------------------------------------
// Set overwrites existing entry
// -------------------------------------------------------------------

func TestSetOverwrites(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	msg1 := makeMsg("overwrite.com.", dns.RcodeSuccess, 300)
	c.Set(msg1)

	msg2 := makeMsg("overwrite.com.", dns.RcodeSuccess, 600)
	msg2.Answer[0].(*dns.A).A = net.ParseIP("5.6.7.8")
	c.Set(msg2)

	got, found, _ := c.Get("overwrite.com.", dns.TypeA, dns.ClassINET)
	if !found {
		t.Fatal("expected to find overwritten entry")
	}
	a := got.Answer[0].(*dns.A)
	if !a.A.Equal(net.ParseIP("5.6.7.8")) {
		t.Errorf("after overwrite A = %v, want 5.6.7.8", a.A)
	}
}

// -------------------------------------------------------------------
// Get returns a copy (mutation isolation)
// -------------------------------------------------------------------

func TestGetReturnsCopy(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	msg := makeMsg("copy.com.", dns.RcodeSuccess, 300)
	c.Set(msg)

	got1, _, _ := c.Get("copy.com.", dns.TypeA, dns.ClassINET)
	got1.Rcode = dns.RcodeServerFailure // mutate

	got2, _, _ := c.Get("copy.com.", dns.TypeA, dns.ClassINET)
	if got2.Rcode == dns.RcodeServerFailure {
		t.Error("mutation of returned message affected cache")
	}
}

// -------------------------------------------------------------------
// Concurrent access (race detector)
// -------------------------------------------------------------------

func TestConcurrentAccess(t *testing.T) {
	c := New(1024, 1)
	defer c.Stop()

	var wg sync.WaitGroup
	const goroutines = 50
	const opsPerGoroutine = 100

	// Concurrent writers.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				domain := dns.Fqdn("concurrent.com.")
				msg := makeMsg(domain, dns.RcodeSuccess, 300)
				c.Set(msg)
			}
		}(i)
	}

	// Concurrent readers.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				c.Get("concurrent.com.", dns.TypeA, dns.ClassINET)
			}
		}()
	}

	// Concurrent Flush.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 10; j++ {
			c.Flush()
			time.Sleep(time.Millisecond)
		}
	}()

	// Concurrent Stats.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < opsPerGoroutine; j++ {
			_ = c.Stats()
		}
	}()

	// Concurrent Delete.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < opsPerGoroutine; j++ {
			c.Delete("concurrent.com")
		}
	}()

	wg.Wait()
}

// -------------------------------------------------------------------
// cacheKey
// -------------------------------------------------------------------

func TestCacheKeyFormat(t *testing.T) {
	k := cacheKey("Example.COM.", dns.TypeA, dns.ClassINET)
	if k != "example.com.\x00A\x00IN" {
		t.Errorf("cacheKey = %q, unexpected format", k)
	}
}

// -------------------------------------------------------------------
// minTTL enforcement
// -------------------------------------------------------------------

func TestMinTTLEnforced(t *testing.T) {
	c := New(1024, 300) // minTTL = 300s
	defer c.Stop()

	// Message with TTL=1, but minTTL floors it to 300.
	msg := makeMsg("minttl.com.", dns.RcodeSuccess, 1)
	c.Set(msg)

	got, found, _ := c.Get("minttl.com.", dns.TypeA, dns.ClassINET)
	if !found {
		t.Fatal("entry should be cached")
	}
	_ = got

	// Should still be present after 2 seconds (minTTL=300).
	time.Sleep(2 * time.Second)
	_, found, _ = c.Get("minttl.com.", dns.TypeA, dns.ClassINET)
	if !found {
		t.Error("entry should still be valid (minTTL not elapsed)")
	}
}

// -------------------------------------------------------------------
// shard.len
// -------------------------------------------------------------------

func TestShardLen(t *testing.T) {
	s := newShard(10)
	if s.len() != 0 {
		t.Errorf("empty shard len = %d, want 0", s.len())
	}

	msg := makeMsg("shardlen.com.", dns.RcodeSuccess, 300)
	k := cacheKey("shardlen.com.", dns.TypeA, dns.ClassINET)
	s.set(k, msg, 300)
	if s.len() != 1 {
		t.Errorf("shard len = %d, want 1", s.len())
	}
}

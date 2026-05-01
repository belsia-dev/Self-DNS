package cache

import (
	"container/list"
	"encoding/base64"
	"hash/fnv"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

const numShards = 16
const staleFraction = 0.25
const (
	minStale = 10 * time.Second
	maxStale = 60 * time.Second
)

type entry struct {
	msg        *dns.Msg
	expires    time.Time
	staleUntil time.Time
	key        string
	hits       int64
	elem       *list.Element
}

type shard struct {
	mu        sync.Mutex
	items     map[string]*entry
	lru       *list.List
	maxSize   int
	evictions int64
}

func newShard(maxSize int) shard {
	return shard{
		items:   make(map[string]*entry, maxSize),
		lru:     list.New(),
		maxSize: maxSize,
	}
}

func (s *shard) get(k string) (*dns.Msg, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.items[k]
	if !ok {
		return nil, false, false
	}

	now := time.Now()
	if now.After(e.staleUntil) {
		s.remove(e)
		return nil, false, false
	}

	isStale := now.After(e.expires)
	e.hits++
	s.lru.MoveToFront(e.elem)
	return e.msg.Copy(), true, isStale
}

func (s *shard) set(k string, msg *dns.Msg, ttl uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expires := time.Now().Add(time.Duration(ttl) * time.Second)
	staleWin := time.Duration(float64(ttl)*staleFraction) * time.Second
	if staleWin < minStale {
		staleWin = minStale
	}
	if staleWin > maxStale {
		staleWin = maxStale
	}
	staleUntil := expires.Add(staleWin)

	if e, ok := s.items[k]; ok {
		e.msg = msg.Copy()
		e.expires = expires
		e.staleUntil = staleUntil
		s.lru.MoveToFront(e.elem)
		return
	}
	if len(s.items) >= s.maxSize {
		s.evictLRU()
	}
	e := &entry{
		msg:        msg.Copy(),
		expires:    expires,
		staleUntil: staleUntil,
		key:        k,
	}
	e.elem = s.lru.PushFront(e)
	s.items[k] = e
}

func (s *shard) flush() {
	s.mu.Lock()
	s.items = make(map[string]*entry, s.maxSize)
	s.lru.Init()
	s.mu.Unlock()
}

func (s *shard) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

func (s *shard) remove(e *entry) {
	delete(s.items, e.key)
	s.lru.Remove(e.elem)
}

func (s *shard) evictLRU() {
	elem := s.lru.Back()
	if elem == nil {
		return
	}
	s.remove(elem.Value.(*entry))
	s.evictions++
}

func (s *shard) purgeExpired() {
	s.mu.Lock()
	now := time.Now()
	for _, e := range s.items {
		if now.After(e.staleUntil) {
			s.remove(e)
		}
	}
	s.mu.Unlock()
}

func (s *shard) snapshot() []snapshotEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]snapshotEntry, 0, len(s.items))
	for _, e := range s.items {
		raw, err := e.msg.Pack()
		if err != nil {
			continue
		}
		out = append(out, snapshotEntry{
			Key:        e.key,
			ExpiresAt:  e.expires.UnixMilli(),
			StaleUntil: e.staleUntil.UnixMilli(),
			Hits:       e.hits,
			MsgB64:     base64.StdEncoding.EncodeToString(raw),
		})
	}
	return out
}

type snapshotEntry struct {
	Key        string `json:"key"`
	ExpiresAt  int64  `json:"expires_at"`
	StaleUntil int64  `json:"stale_until"`
	Hits       int64  `json:"hits"`
	MsgB64     string `json:"msg_b64"`
}

type Stats struct {
	Size      int     `json:"size"`
	MaxSize   int     `json:"max_size"`
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	HitRate   float64 `json:"hit_rate"`
	Evictions int64   `json:"evictions"`
}

type HotEntry struct {
	Name   string `json:"name"`
	Qtype  uint16 `json:"qtype"`
	Qclass uint16 `json:"qclass"`
	Hits   int64  `json:"hits"`
	TTLSec int64  `json:"ttl_sec"`
}

type Cache struct {
	shards  [numShards]shard
	minTTL  uint32
	maxSize int
	hits    atomic.Int64
	misses  atomic.Int64
	stopCh  chan struct{}
}

func New(maxSize int, minTTL uint32) *Cache {
	if maxSize <= 0 {
		maxSize = 10000
	}
	perShard := max(1, maxSize/numShards)
	c := &Cache{maxSize: maxSize, minTTL: minTTL, stopCh: make(chan struct{})}
	for i := range c.shards {
		c.shards[i] = newShard(perShard)
	}
	go c.janitor()
	return c
}

func (c *Cache) Get(name string, qtype, qclass uint16) (*dns.Msg, bool, bool) {
	k := cacheKey(name, qtype, qclass)
	msg, found, isStale := c.pickShard(k).get(k)
	if found {
		c.hits.Add(1)
	} else {
		c.misses.Add(1)
	}
	return msg, found, isStale
}

func (c *Cache) Set(msg *dns.Msg) {
	if msg == nil || len(msg.Question) == 0 {
		return
	}
	q := msg.Question[0]
	k := cacheKey(q.Name, q.Qtype, q.Qclass)

	var ttl uint32
	switch msg.Rcode {
	case dns.RcodeSuccess:
		ttl = minTTLFromMsg(msg)
	case dns.RcodeNameError:
		ttl = negativeTTL(msg)
	default:
		return
	}

	if ttl < c.minTTL {
		ttl = c.minTTL
	}
	if ttl == 0 {
		return
	}
	c.pickShard(k).set(k, msg, ttl)
}

func (c *Cache) Flush() {
	for i := range c.shards {
		c.shards[i].flush()
	}
}

var targetedDeleteQtypes = []uint16{
	dns.TypeA,
	dns.TypeAAAA,
	dns.TypeHTTPS,
	dns.TypeSVCB,
	dns.TypeCNAME,
	dns.TypeMX,
	dns.TypeTXT,
}

func (c *Cache) Delete(name string) {
	norm := strings.ToLower(strings.TrimSpace(name))
	if norm == "" {
		return
	}
	if !strings.HasSuffix(norm, ".") {
		norm += "."
	}
	for _, qt := range targetedDeleteQtypes {
		k := cacheKey(norm, qt, dns.ClassINET)
		s := c.pickShard(k)
		s.mu.Lock()
		if e, ok := s.items[k]; ok {
			s.remove(e)
		}
		s.mu.Unlock()
	}
}

func (c *Cache) Len() int {
	total := 0
	for i := range c.shards {
		total += c.shards[i].len()
	}
	return total
}

func (c *Cache) Stats() Stats {
	h, m := c.hits.Load(), c.misses.Load()
	var rate float64
	if tot := h + m; tot > 0 {
		rate = float64(h) / float64(tot) * 100
	}
	var evictions int64
	for i := range c.shards {
		c.shards[i].mu.Lock()
		evictions += c.shards[i].evictions
		c.shards[i].mu.Unlock()
	}
	return Stats{
		Size:      c.Len(),
		MaxSize:   c.maxSize,
		Hits:      h,
		Misses:    m,
		HitRate:   rate,
		Evictions: evictions,
	}
}

func (c *Cache) Hot(n int) []HotEntry {
	if n <= 0 {
		n = 20
	}
	type scored struct {
		key     string
		hits    int64
		ttl     int64
	}
	all := make([]scored, 0, 256)
	now := time.Now()
	for i := range c.shards {
		c.shards[i].mu.Lock()
		for _, e := range c.shards[i].items {
			all = append(all, scored{
				key:  e.key,
				hits: e.hits,
				ttl:  int64(e.expires.Sub(now).Seconds()),
			})
		}
		c.shards[i].mu.Unlock()
	}
	sort.Slice(all, func(i, j int) bool { return all[i].hits > all[j].hits })
	if len(all) > n {
		all = all[:n]
	}
	out := make([]HotEntry, 0, len(all))
	for _, s := range all {
		name, qt, qc, ok := parseKey(s.key)
		if !ok {
			continue
		}
		out = append(out, HotEntry{
			Name:   name,
			Qtype:  qt,
			Qclass: qc,
			Hits:   s.hits,
			TTLSec: s.ttl,
		})
	}
	return out
}

type Export struct {
	Version int             `json:"version"`
	SavedAt int64           `json:"saved_at"`
	Entries []snapshotEntry `json:"entries"`
}

func (c *Cache) Export() Export {
	var entries []snapshotEntry
	for i := range c.shards {
		entries = append(entries, c.shards[i].snapshot()...)
	}
	return Export{Version: 1, SavedAt: time.Now().Unix(), Entries: entries}
}

func (c *Cache) Import(e Export) int {
	imported := 0
	now := time.Now()
	for _, se := range e.Entries {
		raw, err := base64.StdEncoding.DecodeString(se.MsgB64)
		if err != nil {
			continue
		}
		msg := new(dns.Msg)
		if err := msg.Unpack(raw); err != nil {
			continue
		}
		expires := time.UnixMilli(se.ExpiresAt)
		if expires.Before(now) {
			continue
		}
		ttl := uint32(expires.Sub(now).Seconds())
		if ttl == 0 {
			continue
		}
		c.pickShard(se.Key).set(se.Key, msg, ttl)
		imported++
	}
	return imported
}

func (c *Cache) pickShard(k string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(k))
	return &c.shards[h.Sum32()%numShards]
}

func (c *Cache) janitor() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C:
			for i := range c.shards {
				c.shards[i].purgeExpired()
			}
		}
	}
}

func (c *Cache) Stop() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}

func cacheKey(name string, qtype, qclass uint16) string {
	return strings.ToLower(name) + "\x00" + dns.TypeToString[qtype] + "\x00" + dns.ClassToString[qclass]
}

func parseKey(k string) (string, uint16, uint16, bool) {
	parts := strings.Split(k, "\x00")
	if len(parts) != 3 {
		return "", 0, 0, false
	}
	qt, ok1 := dns.StringToType[parts[1]]
	qc, ok2 := dns.StringToClass[parts[2]]
	if !ok1 || !ok2 {
		return "", 0, 0, false
	}
	return parts[0], qt, qc, true
}

func minTTLFromMsg(msg *dns.Msg) uint32 {
	var min uint32 = ^uint32(0)
	for _, rr := range msg.Answer {
		if t := rr.Header().Ttl; t < min {
			min = t
		}
	}
	for _, rr := range msg.Ns {
		if t := rr.Header().Ttl; t < min {
			min = t
		}
	}
	if min == ^uint32(0) {
		return 0
	}
	return min
}

func negativeTTL(msg *dns.Msg) uint32 {
	for _, rr := range msg.Ns {
		if soa, ok := rr.(*dns.SOA); ok {
			ttl := soa.Hdr.Ttl
			if soa.Minttl < ttl {
				ttl = soa.Minttl
			}
			return ttl
		}
	}
	return 300
}



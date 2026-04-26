package blocker

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"github.com/belsia-dev/Self-DNS/server/config"
)

var normalise = config.NormalizeDomain

type FileInfo struct {
	Path   string `json:"path"`
	Count  int    `json:"count"`
	Loaded bool   `json:"loaded"`
}

type Blocker struct {
	mu      sync.RWMutex
	enabled bool
	domains map[string]struct{}
	files   []FileInfo
}

func New(cfg config.BlocklistConfig) *Blocker {
	b := &Blocker{
		enabled: cfg.Enabled,
		domains: make(map[string]struct{}),
	}

	for _, d := range cfg.Domains {
		b.domains[normalise(d)] = struct{}{}
	}

	for _, path := range cfg.Files {
		b.loadFile(path)
	}

	return b
}

func (b *Blocker) IsBlocked(domain string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if !b.enabled {
		return false
	}
	domain = normalise(domain)
	for {
		if _, ok := b.domains[domain]; ok {
			return true
		}
		idx := strings.IndexByte(domain, '.')
		if idx < 0 {
			return false
		}
		domain = domain[idx+1:]
	}
}

func (b *Blocker) Add(domain string) {
	b.mu.Lock()
	b.domains[normalise(domain)] = struct{}{}
	b.mu.Unlock()
}

func (b *Blocker) Remove(domain string) {
	b.mu.Lock()
	delete(b.domains, normalise(domain))
	b.mu.Unlock()
}

func (b *Blocker) List() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.domains))
	for d := range b.domains {
		out = append(out, d)
	}
	return out
}

func (b *Blocker) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.domains)
}

func (b *Blocker) Toggle(enabled bool) {
	b.mu.Lock()
	b.enabled = enabled
	b.mu.Unlock()
}

func (b *Blocker) IsEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.enabled
}

func (b *Blocker) Files() []FileInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]FileInfo, len(b.files))
	copy(out, b.files)
	return out
}

func (b *Blocker) AddFile(path string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.loadFile(path)
}

func (b *Blocker) RemoveFile(path string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, f := range b.files {
		if f.Path == path {
			b.files = append(b.files[:i], b.files[i+1:]...)
			break
		}
	}
	b.rebuildDomains()
}

func (b *Blocker) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		b.files = append(b.files, FileInfo{Path: path, Loaded: false})
		return err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		domain := parseLine(line)
		if domain == "" {
			continue
		}
		b.domains[normalise(domain)] = struct{}{}
		count++
	}

	for i, fi := range b.files {
		if fi.Path == path {
			b.files[i] = FileInfo{Path: path, Count: count, Loaded: true}
			return scanner.Err()
		}
	}
	b.files = append(b.files, FileInfo{Path: path, Count: count, Loaded: true})
	return scanner.Err()
}

func (b *Blocker) rebuildDomains() {
	files := make([]string, 0, len(b.files))
	for _, f := range b.files {
		files = append(files, f.Path)
	}
	b.domains = make(map[string]struct{})
	b.files = nil
	for _, path := range files {
		_ = b.loadFile(path)
	}
}

func parseLine(line string) string {
	if idx := strings.IndexByte(line, '#'); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	if len(fields) >= 2 {
		addr := fields[0]
		if addr == "0.0.0.0" || addr == "127.0.0.1" || addr == "::" {
			return fields[1]
		}
	}
	if len(fields) == 1 {
		return fields[0]
	}
	return ""
}


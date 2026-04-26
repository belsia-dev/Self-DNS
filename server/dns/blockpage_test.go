package dns

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/belsia-dev/Self-DNS/server/config"
	"github.com/miekg/dns"
)

type fakeResponseWriter struct {
	local  net.Addr
	remote net.Addr
}

func (f fakeResponseWriter) LocalAddr() net.Addr       { return f.local }
func (f fakeResponseWriter) RemoteAddr() net.Addr      { return f.remote }
func (f fakeResponseWriter) WriteMsg(*dns.Msg) error   { return nil }
func (f fakeResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (f fakeResponseWriter) Close() error              { return nil }
func (f fakeResponseWriter) TsigStatus() error         { return nil }
func (f fakeResponseWriter) TsigTimersOnly(bool)       {}
func (f fakeResponseWriter) Hijack()                   {}

func TestResolveBlockedIPUsesLoopbackInLocalMode(t *testing.T) {
	cfg := &config.Config{ServiceMode: "local"}
	page := config.BlockPageConfig{Bind: "127.0.0.1:80"}

	got := resolveBlockedIP(cfg, &net.UDPAddr{IP: net.IPv4zero, Port: 53}, page, false)
	if got == nil || !got.Equal(net.ParseIP("127.0.0.1")) {
		t.Fatalf("expected 127.0.0.1, got %v", got)
	}
}

func TestBlockedResponseReturnsEmptyForHTTPSQueries(t *testing.T) {
	srv := &Server{
		cfg: &config.Config{
			ServiceMode: "local",
			Blocklist: config.BlocklistConfig{
				ResponseMode: "block_page",
				BlockPage: config.BlockPageConfig{
					Bind: "127.0.0.1:80",
				},
			},
		},
	}
	req := new(dns.Msg)
	req.SetQuestion("blocked.example.", dns.TypeHTTPS)

	resp := srv.blockedResponse(fakeResponseWriter{
		local:  &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53},
		remote: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50000},
	}, req)

	if resp.Rcode != dns.RcodeSuccess {
		t.Fatalf("expected NOERROR for HTTPS query, got %d", resp.Rcode)
	}
	if len(resp.Answer) != 0 {
		t.Fatalf("expected no answers for HTTPS query, got %d", len(resp.Answer))
	}
}

func TestBlockedResponseReturnsARecordForAQueries(t *testing.T) {
	srv := &Server{
		cfg: &config.Config{
			ServiceMode: "local",
			Blocklist: config.BlocklistConfig{
				ResponseMode: "block_page",
				BlockPage: config.BlockPageConfig{
					Bind: "127.0.0.1:80",
				},
			},
		},
	}
	req := new(dns.Msg)
	req.SetQuestion("blocked.example.", dns.TypeA)

	resp := srv.blockedResponse(fakeResponseWriter{
		local:  &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 53},
		remote: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50000},
	}, req)

	if resp.Rcode != dns.RcodeSuccess {
		t.Fatalf("expected NOERROR, got %d", resp.Rcode)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
	}
	record, ok := resp.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", resp.Answer[0])
	}
	if !record.A.Equal(net.ParseIP("127.0.0.1")) {
		t.Fatalf("expected 127.0.0.1 answer, got %v", record.A)
	}
}

func stubBlockPageServerStart(t *testing.T) {
	t.Helper()

	previous := newBlockPageHTTPServer
	newBlockPageHTTPServer = func(addr string, _ http.Handler, _ func() bool) (*httpServerAdapter, error) {
		return &httpServerAdapter{
			addr: addr,
			shutdown: func(context.Context) error {
				return nil
			},
		}, nil
	}
	t.Cleanup(func() {
		newBlockPageHTTPServer = previous
	})
}

func TestApplyBlockPageConfigUpdatesHTMLWithoutRestartingHTTPServer(t *testing.T) {
	stubBlockPageServerStart(t)

	srv := &Server{
		cfg: &config.Config{
			ServiceMode: "local",
			Blocklist: config.BlocklistConfig{
				ResponseMode: "block_page",
				BlockPage: config.BlockPageConfig{
					Bind: "127.0.0.1:0",
					HTML: "<h1>old</h1>",
				},
			},
		},
	}
	srv.running.Store(true)

	srv.mu.Lock()
	if err := srv.startBlockPageServerLocked(); err != nil {
		srv.mu.Unlock()
		t.Fatalf("start block page server: %v", err)
	}
	original := srv.blockPageServer
	srv.mu.Unlock()
	defer srv.Stop()

	next := config.Clone(srv.cfg)
	next.Blocklist.BlockPage.HTML = "<h1>new</h1>"

	if err := srv.ApplyBlockPageConfig(next); err != nil {
		t.Fatalf("apply block page config: %v", err)
	}

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if srv.cfg.Blocklist.BlockPage.HTML != "<h1>new</h1>" {
		t.Fatalf("expected updated HTML, got %q", srv.cfg.Blocklist.BlockPage.HTML)
	}
	if srv.blockPageServer != original {
		t.Fatalf("expected HTTP server to stay running without restart")
	}
}

func TestApplyBlockPageConfigStartsHTTPServerWhenModeChanges(t *testing.T) {
	stubBlockPageServerStart(t)

	srv := &Server{
		cfg: &config.Config{
			ServiceMode: "local",
			Blocklist: config.BlocklistConfig{
				ResponseMode: "nxdomain",
				BlockPage: config.BlockPageConfig{
					Bind: "127.0.0.1:0",
					HTML: "<h1>blocked</h1>",
				},
			},
		},
	}
	srv.running.Store(true)
	defer srv.Stop()

	next := config.Clone(srv.cfg)
	next.Blocklist.ResponseMode = "block_page"

	if err := srv.ApplyBlockPageConfig(next); err != nil {
		t.Fatalf("apply block page config: %v", err)
	}

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if srv.blockPageServer == nil {
		t.Fatalf("expected block page HTTP server to start")
	}
	if srv.cfg.Blocklist.ResponseMode != "block_page" {
		t.Fatalf("expected response mode to be block_page, got %q", srv.cfg.Blocklist.ResponseMode)
	}
}

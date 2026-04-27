package dns

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/belsia-dev/Self-DNS/server/config"
	"github.com/miekg/dns"
)

const blockPageShutdownTimeout = 5 * time.Second

var newBlockPageHTTPServer = func(addr string, handler http.Handler, isRunning func() bool) (*httpServerAdapter, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed && isRunning() {
			log.Printf("block page server stopped: %v", err)
		}
	}()
	return &httpServerAdapter{addr: ln.Addr().String(), shutdown: srv.Shutdown}, nil
}

var newBlockPageHTTPSServer = func(addr string, handler http.Handler, tlsCfg *tls.Config, isRunning func() bool) (*httpServerAdapter, error) {
	ln, err := tls.Listen("tcp", addr, tlsCfg)
	if err != nil {
		return nil, err
	}
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		TLSConfig:    tlsCfg,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed && isRunning() {
			log.Printf("block page HTTPS server stopped: %v", err)
		}
	}()
	return &httpServerAdapter{addr: ln.Addr().String(), shutdown: srv.Shutdown}, nil
}


type blockPageState struct {
	Domain      string `json:"domain"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	RequestURI  string `json:"request_uri"`
	Method      string `json:"method"`
	GeneratedAt string `json:"generated_at"`
}

func (s *Server) blockedResponse(w dns.ResponseWriter, req *dns.Msg) *dns.Msg {
	s.mu.RLock()
	cfg := s.cfg
	mode := s.cfg.Blocklist.ResponseMode
	page := s.cfg.Blocklist.BlockPage
	s.mu.RUnlock()

	if mode != config.ResponseModeBlockPage {
		return rcodeResponse(req, dns.RcodeNameError)
	}

	q := req.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		ip := resolveBlockedIP(cfg, w.LocalAddr(), page, false)
		if ip == nil {
			return rcodeResponse(req, dns.RcodeNameError)
		}
		return blockedHostResponse(req, ip.String())
	case dns.TypeAAAA:
		ip := resolveBlockedIP(cfg, w.LocalAddr(), page, true)
		if ip == nil {
			return emptyResponse(req)
		}
		return blockedHostResponse(req, ip.String())
	default:
		return emptyResponse(req)
	}
}

func (s *Server) startBlockPageServerLocked() error {
	s.stopBlockPageServerLocked()

	if s.cfg.Blocklist.ResponseMode != config.ResponseModeBlockPage {
		return nil
	}

	addr := s.cfg.Blocklist.BlockPage.Bind
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleBlockPageHTTP)

	httpSrv, err := newBlockPageHTTPServer(addr, mux, s.running.Load)
	if err != nil {
		return fmt.Errorf("block page server listen %s: %w", addr, err)
	}
	s.blockPageServer = httpSrv
	log.Printf("Block page server listening on http://%s", httpSrv.addr)

	if host, _, splitErr := net.SplitHostPort(addr); splitErr == nil && s.blockPageCA != nil {
		httpsAddr := net.JoinHostPort(host, "443")
		tlsCfg := s.blockPageCA.tlsConfig()
		if httpsSrv, httpsErr := newBlockPageHTTPSServer(httpsAddr, mux, tlsCfg, s.running.Load); httpsErr == nil {
			s.blockPageHTTPSServer = httpsSrv
			log.Printf("Block page HTTPS server listening on https://%s", httpsSrv.addr)
		} else {
			log.Printf("Block page HTTPS server could not start on %s: %v", httpsAddr, httpsErr)
		}
	}

	return nil
}

func (s *Server) stopBlockPageServerLocked() {
	ctx, cancel := context.WithTimeout(context.Background(), blockPageShutdownTimeout)
	defer cancel()
	if s.blockPageServer != nil {
		_ = s.blockPageServer.shutdown(ctx)
		s.blockPageServer = nil
	}
	if s.blockPageHTTPSServer != nil {
		_ = s.blockPageHTTPSServer.shutdown(ctx)
		s.blockPageHTTPSServer = nil
	}
	if s.blockPageCA != nil {
		s.blockPageCA.mu.Lock()
		s.blockPageCA.cache = make(map[string]*tls.Certificate)
		s.blockPageCA.mu.Unlock()
	}
}

func (s *Server) handleBlockPageHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	page := s.cfg.Blocklist.BlockPage
	s.mu.RUnlock()

	host := requestHost(r.Host)
	if host == "" {
		host = "blocked-domain"
	}

	state := blockPageState{
		Domain:      host,
		Host:        host,
		Path:        r.URL.Path,
		RequestURI:  r.URL.RequestURI(),
		Method:      r.Method,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(state)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	w.WriteHeader(http.StatusUnavailableForLegalReasons)

	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Blocked by SelfDNS</title>
  <style>%s</style>
</head>
<body>
%s
  <script>window.SELFDNS_BLOCK_PAGE = %s;</script>
  <script>%s</script>
</body>
</html>`, page.CSS, replaceBlockTokens(page.HTML, state), string(payload), page.JS)
}

func resolveBlockedIP(cfg *config.Config, localAddr net.Addr, page config.BlockPageConfig, wantIPv6 bool) net.IP {
	candidates := []string{}
	if wantIPv6 {
		candidates = append(candidates, page.IPv6)
	} else {
		candidates = append(candidates, page.IPv4)
	}
	candidates = append(candidates, bindHost(page.Bind))
	if localAddr != nil {
		candidates = append(candidates, localAddr.String())
	}

	for _, candidate := range candidates {
		if ip := parseBlockedIP(candidate, wantIPv6); isUsableReplyIP(ip, wantIPv6) {
			return ip
		}
	}

	if cfg != nil && cfg.ServiceMode == "local" {
		if wantIPv6 {
			return net.ParseIP("::1")
		}
		return net.ParseIP("127.0.0.1")
	}

	return discoverInterfaceIP(wantIPv6)
}

func parseBlockedIP(raw string, wantIPv6 bool) net.IP {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")

	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	if wantIPv6 {
		if ip.To4() == nil {
			return ip
		}
		return nil
	}
	return ip.To4()
}

func bindHost(bind string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(bind))
	if err != nil {
		return ""
	}
	return host
}

func isUsableReplyIP(ip net.IP, wantIPv6 bool) bool {
	if ip == nil || ip.IsUnspecified() {
		return false
	}
	if wantIPv6 {
		return ip.To4() == nil && ip.IsGlobalUnicast()
	}
	ip = ip.To4()
	return ip != nil && ip.IsGlobalUnicast()
}

func discoverInterfaceIP(wantIPv6 bool) net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var fallback net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := ipFromAddr(addr, wantIPv6)
			if !isUsableReplyIP(ip, wantIPv6) {
				continue
			}
			if ip.IsPrivate() {
				return ip
			}
			if fallback == nil {
				fallback = ip
			}
		}
	}
	return fallback
}

func ipFromAddr(addr net.Addr, wantIPv6 bool) net.IP {
	switch value := addr.(type) {
	case *net.IPNet:
		if wantIPv6 {
			if value.IP.To4() == nil {
				return value.IP
			}
			return nil
		}
		return value.IP.To4()
	case *net.IPAddr:
		if wantIPv6 {
			if value.IP.To4() == nil {
				return value.IP
			}
			return nil
		}
		return value.IP.To4()
	default:
		return parseBlockedIP(addr.String(), wantIPv6)
	}
}

func requestHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	return host
}

func replaceBlockTokens(htmlSnippet string, state blockPageState) string {
	replacer := strings.NewReplacer(
		"{{DOMAIN}}", html.EscapeString(state.Domain),
		"{{HOST}}", html.EscapeString(state.Host),
		"{{PATH}}", html.EscapeString(state.Path),
		"{{REQUEST_URI}}", html.EscapeString(state.RequestURI),
		"{{METHOD}}", html.EscapeString(state.Method),
		"{{GENERATED_AT}}", html.EscapeString(state.GeneratedAt),
	)
	return replacer.Replace(htmlSnippet)
}

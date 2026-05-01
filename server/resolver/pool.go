package resolver

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type pooledConn struct {
	conn    *dns.Conn
	server  string
	created time.Time
}

type dotPool struct {
	tlsCfg   *tls.Config
	sessions tls.ClientSessionCache
	mu       sync.Mutex
	idle     map[string][]*pooledConn
	stopCh   chan struct{}
}

func newDotPool(tlsCfg *tls.Config) *dotPool {
	cfg := tlsCfg.Clone()
	cfg.ClientSessionCache = tls.NewLRUClientSessionCache(64)
	p := &dotPool{
		tlsCfg:   cfg,
		sessions: cfg.ClientSessionCache,
		idle:     make(map[string][]*pooledConn),
		stopCh:   make(chan struct{}),
	}
	go p.janitor()
	return p
}

func (p *dotPool) acquire(server string) (*pooledConn, error) {
	p.mu.Lock()
	stack := p.idle[server]
	for len(stack) > 0 {
		last := len(stack) - 1
		pc := stack[last]
		p.idle[server] = stack[:last]
		p.mu.Unlock()

		if time.Since(pc.created) < connMaxAge {
			return pc, nil
		}
		_ = pc.conn.Close()

		p.mu.Lock()
		stack = p.idle[server]
	}
	p.mu.Unlock()

	return p.dial(server)
}

func (p *dotPool) release(pc *pooledConn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(pc.created) >= connMaxAge || len(p.idle[pc.server]) >= maxIdlePer {
		_ = pc.conn.Close()
		return
	}
	p.idle[pc.server] = append(p.idle[pc.server], pc)
}

func (p *dotPool) discard(pc *pooledConn) { _ = pc.conn.Close() }

func (p *dotPool) dial(server string) (*pooledConn, error) {
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		host = server
	}

	cfg := p.tlsCfg.Clone()
	cfg.ServerName = host
	cfg.ClientSessionCache = p.sessions

	tcpConn, err := net.DialTimeout("tcp", server, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", server, err)
	}

	if tcp, ok := tcpConn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(30 * time.Second)
		_ = tcp.SetNoDelay(true)
	}

	tlsConn := tls.Client(tcpConn, cfg)
	if err := tlsConn.SetDeadline(time.Now().Add(dialTimeout)); err != nil {
		_ = tcpConn.Close()
		return nil, err
	}
	if err := tlsConn.Handshake(); err != nil {
		_ = tcpConn.Close()
		return nil, fmt.Errorf("TLS handshake %s: %w", server, err)
	}
	_ = tlsConn.SetDeadline(time.Time{})

	return &pooledConn{
		conn:    &dns.Conn{Conn: tlsConn},
		server:  server,
		created: time.Now(),
	}, nil
}

func (p *dotPool) janitor() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case <-t.C:
			p.mu.Lock()
			for server, stack := range p.idle {
				alive := stack[:0]
				for _, pc := range stack {
					if time.Since(pc.created) < connMaxAge {
						alive = append(alive, pc)
					} else {
						_ = pc.conn.Close()
					}
				}
				p.idle[server] = alive
			}
			p.mu.Unlock()
		}
	}
}

func (p *dotPool) Stop() {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
}

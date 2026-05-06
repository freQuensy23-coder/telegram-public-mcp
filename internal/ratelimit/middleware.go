package ratelimit

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	GlobalLimit int
	IPLimit     int
	Window      time.Duration
	Clock       func() time.Time
}

type Middleware struct {
	global *fixedWindow
	ips    map[string]*fixedWindow
	cfg    Config
	mu     sync.Mutex
}

func New(cfg Config) *Middleware {
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &Middleware{
		global: newFixedWindow(cfg.GlobalLimit, cfg.Window, cfg.Clock),
		ips:    map[string]*fixedWindow{},
		cfg:    cfg,
	}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if retryAfter, ok := m.global.Allow(); !ok {
			writeLimited(w, retryAfter)
			return
		}
		if retryAfter, ok := m.allowIP(clientIP(r)); !ok {
			writeLimited(w, retryAfter)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) allowIP(ip string) (time.Duration, bool) {
	if m.cfg.IPLimit <= 0 {
		return 0, true
	}
	m.mu.Lock()
	limiter := m.ips[ip]
	if limiter == nil {
		limiter = newFixedWindow(m.cfg.IPLimit, m.cfg.Window, m.cfg.Clock)
		m.ips[ip] = limiter
	}
	m.cleanupLocked()
	m.mu.Unlock()
	return limiter.Allow()
}

func (m *Middleware) cleanupLocked() {
	now := m.cfg.Clock()
	for ip, limiter := range m.ips {
		if now.Sub(limiter.startedAt()) > 2*m.cfg.Window {
			delete(m.ips, ip)
		}
	}
}

func writeLimited(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int(retryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

type fixedWindow struct {
	limit  int
	window time.Duration
	clock  func() time.Time
	mu     sync.Mutex
	start  time.Time
	count  int
}

func newFixedWindow(limit int, window time.Duration, clock func() time.Time) *fixedWindow {
	return &fixedWindow{
		limit:  limit,
		window: window,
		clock:  clock,
		start:  clock(),
	}
}

func (w *fixedWindow) Allow() (time.Duration, bool) {
	if w.limit <= 0 {
		return 0, true
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	now := w.clock()
	elapsed := now.Sub(w.start)
	if elapsed >= w.window {
		w.start = now
		w.count = 0
		elapsed = 0
	}
	if w.count >= w.limit {
		return w.window - elapsed, false
	}
	w.count++
	return 0, true
}

func (w *fixedWindow) startedAt() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.start
}

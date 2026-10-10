package auth

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// PerIPLimiter limits login attempts per IP within a sliding window.
type PerIPLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
	max      int
	window   time.Duration
}

func NewPerIPLimiter(max int, window time.Duration) *PerIPLimiter {
	return &PerIPLimiter{failures: map[string][]time.Time{}, max: max, window: window}
}

// Allow reports whether the IP may attempt a login right now.
func (l *PerIPLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := time.Now().Add(-l.window)
	kept := l.failures[ip][:0]
	for _, t := range l.failures[ip] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.failures[ip] = kept
	return len(kept) < l.max
}

// Fail records a failed attempt for the IP.
func (l *PerIPLimiter) Fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures[ip] = append(l.failures[ip], time.Now())
}

// ClientIP extracts the real client IP, honoring proxy headers like the
// original (Cloudflare first, then X-Forwarded-For).
func ClientIP(r *http.Request) string {
	if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
		return cf
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

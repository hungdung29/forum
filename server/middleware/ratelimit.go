package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements token bucket algorithm for rate limiting
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
}

type visitor struct {
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
	}
	
	// Cleanup old visitors every 10 minutes
	go rl.cleanupLoop()
	
	return rl
}

// Allow checks if a request should be allowed based on rate limits
// maxTokens: maximum number of requests allowed
// refillRate: how often to add 1 token back
func (rl *RateLimiter) Allow(key string, maxTokens int, refillRate time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	v, exists := rl.visitors[key]
	if !exists {
		// First request from this visitor
		v = &visitor{
			tokens:     maxTokens - 1,
			lastRefill: time.Now(),
		}
		rl.visitors[key] = v
		return true
	}
	
	// Refill tokens based on time passed
	now := time.Now()
	elapsed := now.Sub(v.lastRefill)
	tokensToAdd := int(elapsed / refillRate)
	
	if tokensToAdd > 0 {
		v.tokens = min(v.tokens+tokensToAdd, maxTokens)
		v.lastRefill = now
	}
	
	// Check if request allowed
	if v.tokens > 0 {
		v.tokens--
		return true
	}
	
	return false // Rate limited
}

// cleanupLoop removes inactive visitors to prevent memory leaks
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, v := range rl.visitors {
			// Remove visitors inactive for > 1 hour
			if now.Sub(v.lastRefill) > 1*time.Hour {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit middleware wrapper
func RateLimit(limiter *RateLimiter, maxRequests int, window time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Use IP as key (or user ID if authenticated)
			key := getClientIP(r)
			
			// Calculate refill rate: window / maxRequests
			refillRate := window / time.Duration(maxRequests)
			
			if !limiter.Allow(key, maxRequests, refillRate) {
				http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
				return
			}
			
			next(w, r)
		}
	}
}

// getClientIP extracts the real client IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (if behind proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Use RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

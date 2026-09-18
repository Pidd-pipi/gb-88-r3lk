package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/util"
)

// Simple in-memory sliding-window rate limiter (per client IP). It is
// intentionally process-local: for a single-instance deployment this is
// sufficient, while a multi-replica deployment should swap this for a shared
// Redis-backed limiter.
type ipCounter struct {
	mu       sync.Mutex
	requests map[string]*window
}

type window struct {
	count   int
	resetAt time.Time
}

var limiter = &ipCounter{requests: map[string]*window{}}

// RateLimit caps requests per IP at maxRequests per minute.
func RateLimit(maxRequests int, logger *slog.Logger) gin.HandlerFunc {
	if maxRequests <= 0 {
		maxRequests = 120
	}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		limiter.mu.Lock()
		w, ok := limiter.requests[ip]
		if !ok || now.After(w.resetAt) {
			w = &window{count: 0, resetAt: now.Add(time.Minute)}
			limiter.requests[ip] = w
			// Opportunistically clean stale entries so the map cannot grow
			// without bound under a high-cardinality client IP workload.
			if len(limiter.requests) > 10_000 {
				for key, val := range limiter.requests {
					if now.After(val.resetAt) {
						delete(limiter.requests, key)
					}
				}
			}
		}
		w.count++
		exceeded := w.count > maxRequests
		limiter.mu.Unlock()

		if exceeded {
			if logger != nil {
				logger.Warn("rate limit exceeded", "client_ip", ip)
			}
			c.AbortWithStatusJSON(http.StatusTooManyRequests, util.Response{
				Code:    constants.CodeTooManyRequests,
				Message: constants.MsgTooManyRequests,
			})
			return
		}
		c.Next()
	}
}

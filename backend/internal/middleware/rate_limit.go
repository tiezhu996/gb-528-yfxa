package middleware

import (
	"sync"
	"time"

	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

type visitorWindow struct {
	count   int
	started time.Time
}

type LocalRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]visitorWindow
	lastGC  time.Time
}

func NewLocalRateLimiter(limit int) *LocalRateLimiter {
	return &LocalRateLimiter{limit: limit, window: time.Minute, clients: make(map[string]visitorWindow), lastGC: time.Now()}
}

func (l *LocalRateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()
		l.mu.Lock()
		entry := l.clients[key]
		if entry.started.IsZero() || now.Sub(entry.started) >= l.window {
			entry = visitorWindow{started: now}
		}
		entry.count++
		l.clients[key] = entry
		allowed := entry.count <= l.limit
		if now.Sub(l.lastGC) > 5*l.window {
			for client, candidate := range l.clients {
				if now.Sub(candidate.started) > 2*l.window {
					delete(l.clients, client)
				}
			}
			l.lastGC = now
		}
		l.mu.Unlock()
		if !allowed {
			c.Header("Retry-After", "60")
			util.Fail(c, &util.AppError{Status: 429, Code: "RATE_LIMITED", Message: "local request limit exceeded"})
			return
		}
		c.Next()
	}
}

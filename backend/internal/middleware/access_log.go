package middleware

import (
	"log/slog"
	"time"

	"stage-rigging-cue-interlock/backend/internal/auth"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		_, username, role := auth.Actor(c)
		logger.Info("http request",
			"request_id", util.RequestID(c),
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"response_bytes", c.Writer.Size(),
			"actor", username,
			"role", role,
			"client_ip", c.ClientIP(),
		)
	}
}

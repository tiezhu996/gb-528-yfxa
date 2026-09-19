package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "request_id", util.RequestID(c), "error", fmt.Sprint(recovered), "stack", string(debug.Stack()))
				util.Fail(c, util.Internal(fmt.Errorf("panic recovered: %v", recovered)))
			}
		}()
		c.Next()
	}
}

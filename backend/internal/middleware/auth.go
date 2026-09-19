package middleware

import (
	"strings"

	"stage-rigging-cue-interlock/backend/internal/auth"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func Auth(service *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			util.Fail(c, util.Unauthorized("AUTH_REQUIRED", "a bearer token is required"))
			return
		}
		claims, err := service.Parse(strings.TrimSpace(parts[1]))
		if err != nil {
			util.Fail(c, err)
			return
		}
		c.Set(auth.ContextUserID, claims.UserID)
		c.Set(auth.ContextUsername, claims.Username)
		c.Set(auth.ContextRole, claims.Role)
		c.Next()
	}
}

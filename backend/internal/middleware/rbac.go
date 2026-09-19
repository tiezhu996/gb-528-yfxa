package middleware

import (
	"stage-rigging-cue-interlock/backend/internal/auth"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

func RBAC(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		_, _, role := auth.Actor(c)
		if _, ok := allowed[role]; !ok {
			util.Fail(c, util.Forbidden("FORBIDDEN", "the current role cannot perform this action"))
			return
		}
		c.Next()
	}
}

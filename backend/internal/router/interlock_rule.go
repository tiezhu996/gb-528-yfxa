package router

import (
	"stage-rigging-cue-interlock/backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterInterlockRuleRoutes(group *gin.RouterGroup, h *handler.InterlockRuleHandler, write, review gin.HandlerFunc) {
	group.GET("/rules", h.List)
	group.GET("/rules/:id", h.Get)
	group.POST("/rules", write, h.Create)
	group.PUT("/rules/:id", write, h.Update)
	group.POST("/rules/:id/toggle", review, h.Toggle)
	group.POST("/rules/:id/test", h.Test)
}

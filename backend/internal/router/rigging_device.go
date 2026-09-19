package router

import (
	"stage-rigging-cue-interlock/backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRiggingDeviceRoutes(group *gin.RouterGroup, h *handler.RiggingDeviceHandler, write gin.HandlerFunc) {
	group.GET("/devices", h.List)
	group.GET("/devices/:id", h.Get)
	group.POST("/devices", write, h.Create)
	group.PUT("/devices/:id", write, h.Update)
}

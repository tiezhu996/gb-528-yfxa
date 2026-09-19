package router

import (
	"stage-rigging-cue-interlock/backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterCueDefinitionRoutes(group *gin.RouterGroup, h *handler.CueDefinitionHandler, write, review gin.HandlerFunc) {
	group.GET("/cues", h.List)
	group.GET("/cues/:id", h.Get)
	group.POST("/cues", write, h.Create)
	group.PUT("/cues/:id", write, h.Update)
	group.POST("/cues/:id/submit", write, h.Submit)
	group.POST("/cues/:id/approve", review, h.Approve)
	group.POST("/cues/:id/reject", review, h.Reject)
	group.POST("/cues/:id/lock", review, h.Lock)
	group.POST("/cues/:id/archive", review, h.Archive)
}

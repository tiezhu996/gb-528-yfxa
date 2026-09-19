package router

import (
	"stage-rigging-cue-interlock/backend/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRehearsalRunRoutes(group *gin.RouterGroup, h *handler.RehearsalRunHandler, write, review gin.HandlerFunc) {
	group.GET("/rehearsals", h.List)
	group.GET("/rehearsals/:id", h.Get)
	group.GET("/rehearsals/:id/compare", h.Compare)
	group.POST("/rehearsals/run", write, h.Run)
	group.POST("/rehearsals/:id/submit", write, h.Submit)
	group.POST("/rehearsals/:id/review", review, h.Review)
}

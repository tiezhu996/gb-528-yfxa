package handler

import (
	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/service"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

type CueDefinitionHandler struct{ service *service.CueDefinitionService }

func NewCueDefinitionHandler(cueService *service.CueDefinitionService) *CueDefinitionHandler {
	return &CueDefinitionHandler{service: cueService}
}

func (h *CueDefinitionHandler) List(c *gin.Context) {
	page, pageSize := util.Pagination(c)
	items, total, err := h.service.List(page, pageSize, c.Query("status"), c.Query("search"))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Page(c, items, page, pageSize, total)
}

func (h *CueDefinitionHandler) Get(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	item, err := h.service.Get(id)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, item)
}

func (h *CueDefinitionHandler) Create(c *gin.Context) {
	var request dto.CreateCueRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "cue timing, actions, or dependencies are invalid", err.Error()))
		return
	}
	item, err := h.service.Create(request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Created(c, item)
}

func (h *CueDefinitionHandler) Update(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.UpdateCueRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "cue fields or version are invalid", err.Error()))
		return
	}
	item, err := h.service.Update(id, request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, item)
}

func (h *CueDefinitionHandler) Submit(c *gin.Context)  { h.transition(c, constants.CuePendingReview) }
func (h *CueDefinitionHandler) Approve(c *gin.Context) { h.transition(c, constants.CueApproved) }
func (h *CueDefinitionHandler) Reject(c *gin.Context)  { h.transition(c, constants.CueDraft) }
func (h *CueDefinitionHandler) Lock(c *gin.Context)    { h.transition(c, constants.CueLocked) }
func (h *CueDefinitionHandler) Archive(c *gin.Context) { h.transition(c, constants.CueArchived) }

func (h *CueDefinitionHandler) transition(c *gin.Context, target constants.CueStatus) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.CueTransitionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "version and review reason are required", err.Error()))
		return
	}
	item, err := h.service.Transition(id, request, target, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, item)
}

package handler

import (
	"strconv"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/service"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

type RehearsalRunHandler struct{ service *service.RehearsalRunService }

func NewRehearsalRunHandler(runService *service.RehearsalRunService) *RehearsalRunHandler {
	return &RehearsalRunHandler{service: runService}
}

func (h *RehearsalRunHandler) List(c *gin.Context) {
	page, pageSize := util.Pagination(c)
	items, total, err := h.service.List(page, pageSize, c.Query("status"), c.Query("severity"), c.Query("search"))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Page(c, items, page, pageSize, total)
}

func (h *RehearsalRunHandler) Get(c *gin.Context) {
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

func (h *RehearsalRunHandler) Run(c *gin.Context) {
	var request dto.RunRehearsalRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "cue_ids must identify one or more locked cues", err.Error()))
		return
	}
	item, err := h.service.Run(request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Created(c, item)
}

func (h *RehearsalRunHandler) Submit(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.RunTransitionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "version and submission reason are required", err.Error()))
		return
	}
	item, err := h.service.Submit(id, request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, item)
}

func (h *RehearsalRunHandler) Review(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.ReviewRunRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "decision, reason, and version are required", err.Error()))
		return
	}
	item, err := h.service.Review(id, request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, item)
}

func (h *RehearsalRunHandler) Compare(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	otherRaw := c.Query("other_id")
	otherID, parseErr := strconv.ParseUint(otherRaw, 10, 64)
	if parseErr != nil || otherID == 0 {
		util.Fail(c, util.BadRequest("INVALID_COMPARE_TARGET", "other_id must be a positive run id", nil))
		return
	}
	result, err := h.service.Compare(id, uint(otherID))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, result)
}

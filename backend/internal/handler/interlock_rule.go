package handler

import (
	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/service"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

type InterlockRuleHandler struct{ service *service.InterlockRuleService }

func NewInterlockRuleHandler(ruleService *service.InterlockRuleService) *InterlockRuleHandler {
	return &InterlockRuleHandler{service: ruleService}
}

func (h *InterlockRuleHandler) List(c *gin.Context) {
	page, pageSize := util.Pagination(c)
	items, total, err := h.service.List(page, pageSize, c.Query("rule_type"), c.Query("severity"), c.Query("search"))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Page(c, items, page, pageSize, total)
}

func (h *InterlockRuleHandler) Get(c *gin.Context) {
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

func (h *InterlockRuleHandler) Create(c *gin.Context) {
	var request dto.CreateInterlockRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "rule scope, threshold, or severity is invalid", err.Error()))
		return
	}
	item, err := h.service.Create(request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Created(c, item)
}

func (h *InterlockRuleHandler) Update(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.UpdateInterlockRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "rule fields or version are invalid", err.Error()))
		return
	}
	item, err := h.service.Update(id, request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, item)
}

func (h *InterlockRuleHandler) Toggle(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.ToggleRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "enabled and rule_version are required", err.Error()))
		return
	}
	item, err := h.service.Toggle(id, request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, item)
}

func (h *InterlockRuleHandler) Test(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.TestRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "actual_value is invalid", err.Error()))
		return
	}
	result, err := h.service.Test(id, request.ActualValue)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, result)
}

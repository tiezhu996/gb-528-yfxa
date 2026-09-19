package handler

import (
	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/service"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
)

type RiggingDeviceHandler struct{ service *service.RiggingDeviceService }

func NewRiggingDeviceHandler(deviceService *service.RiggingDeviceService) *RiggingDeviceHandler {
	return &RiggingDeviceHandler{service: deviceService}
}

func (h *RiggingDeviceHandler) List(c *gin.Context) {
	page, pageSize := util.Pagination(c)
	items, total, err := h.service.List(page, pageSize, c.Query("status"), c.Query("search"))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Page(c, items, page, pageSize, total)
}

func (h *RiggingDeviceHandler) Get(c *gin.Context) {
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

func (h *RiggingDeviceHandler) Create(c *gin.Context) {
	var request dto.CreateRiggingDeviceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "device fields are invalid", err.Error()))
		return
	}
	item, err := h.service.Create(request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Created(c, item)
}

func (h *RiggingDeviceHandler) Update(c *gin.Context) {
	id, err := util.ParseID(c, "id")
	if err != nil {
		util.Fail(c, err)
		return
	}
	var request dto.UpdateRiggingDeviceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "device fields and version are invalid", err.Error()))
		return
	}
	item, err := h.service.Update(id, request, audit.ActorFromContext(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, item)
}

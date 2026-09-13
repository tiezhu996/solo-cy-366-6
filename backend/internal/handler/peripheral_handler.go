package handler

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/esportsbar/backend/internal/constants"
	"github.com/esportsbar/backend/internal/dto"
	"github.com/esportsbar/backend/internal/service"
	"github.com/esportsbar/backend/internal/util"
	"github.com/esportsbar/backend/pkg/response"
)

// PeripheralHandler 外设设备接口处理器。
type PeripheralHandler struct {
	peripheralService *service.PeripheralService
	logger            *slog.Logger
}

// NewPeripheralHandler 构造外设设备接口处理器。
func NewPeripheralHandler(peripheralService *service.PeripheralService, logger *slog.Logger) *PeripheralHandler {
	return &PeripheralHandler{peripheralService: peripheralService, logger: logger}
}

// Create 登记外设设备。
func (h *PeripheralHandler) Create(c *gin.Context) {
	var req dto.CreatePeripheralReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "外设设备参数校验失败："+err.Error())
		return
	}
	p, err := h.peripheralService.Create(&req)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OKMessage(c, constants.MsgPeripheralCreateOK, p)
}

// UpdateStatus 设备状态流转（可借/维护）。
func (h *PeripheralHandler) UpdateStatus(c *gin.Context) {
	var idReq dto.IDReq
	if err := c.ShouldBindUri(&idReq); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "外设设备 ID 无效")
		return
	}
	var req dto.UpdatePeripheralStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "设备状态参数校验失败："+err.Error())
		return
	}
	username, _ := c.Get("username")
	opName, _ := username.(string)
	p, err := h.peripheralService.UpdateStatus(idReq.ID, &req, opName)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OKMessage(c, constants.MsgUpdateSuccess, p)
}

// List 分页查询设备（status=available 时即租借登记表单的可借设备下拉数据）。
func (h *PeripheralHandler) List(c *gin.Context) {
	var query dto.PeripheralQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "设备查询参数校验失败："+err.Error())
		return
	}
	list, total, err := h.peripheralService.List(&query)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, dto.PageResult{List: list, Total: total, Page: query.Page, PageSize: query.PageSize})
}

// abort 统一错误处理。
func (h *PeripheralHandler) abort(c *gin.Context, err error) {
	var appErr *util.AppError
	if errors.As(err, &appErr) {
		response.Fail(c, httpStatusFor(appErr.Code), appErr.Code, appErr.Message)
		return
	}
	h.logger.Error(fmt.Sprintf("peripheral handler error: %v", err))
	response.Fail(c, 500, constants.CodeInternal, "服务器内部错误，请稍后重试")
}

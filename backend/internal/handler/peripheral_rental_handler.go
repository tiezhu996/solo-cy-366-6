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

// PeripheralRentalHandler 外设租借接口处理器。
type PeripheralRentalHandler struct {
	rentalService *service.PeripheralRentalService
	logger        *slog.Logger
}

// NewPeripheralRentalHandler 构造外设租借接口处理器。
func NewPeripheralRentalHandler(rentalService *service.PeripheralRentalService, logger *slog.Logger) *PeripheralRentalHandler {
	return &PeripheralRentalHandler{rentalService: rentalService, logger: logger}
}

// Create 店员为到场会员登记租借。
func (h *PeripheralRentalHandler) Create(c *gin.Context) {
	var req dto.CreateRentalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "租借登记参数校验失败："+err.Error())
		return
	}
	operatorID, _ := c.Get("user_id")
	opID, _ := operatorID.(uint)
	rental, err := h.rentalService.Create(opID, &req)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OKMessage(c, constants.MsgRentalCreateOK, rental)
}

// Return 完好归还：结束租借并释放押金。
func (h *PeripheralRentalHandler) Return(c *gin.Context) {
	var idReq dto.IDReq
	if err := c.ShouldBindUri(&idReq); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "租借记录 ID 无效")
		return
	}
	opID, opName := rentalOperator(c)
	rental, err := h.rentalService.Return(idReq.ID, opID, opName)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OKMessage(c, constants.MsgRentalReturnOK, rental)
}

// Damage 损坏归还登记：记录说明、赔偿金额与处理人，押金不足部分计入会员欠款。
func (h *PeripheralRentalHandler) Damage(c *gin.Context) {
	var idReq dto.IDReq
	if err := c.ShouldBindUri(&idReq); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "租借记录 ID 无效")
		return
	}
	var req dto.DamageRentalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "损坏赔付参数校验失败："+err.Error())
		return
	}
	opID, opName := rentalOperator(c)
	rental, err := h.rentalService.Damage(idReq.ID, opID, opName, &req)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OKMessage(c, constants.MsgRentalDamageOK, rental)
}

// List 分页查询租借记录（店员/管理员）。
func (h *PeripheralRentalHandler) List(c *gin.Context) {
	var query dto.RentalQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "租借查询参数校验失败："+err.Error())
		return
	}
	list, total, err := h.rentalService.List(&query)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, dto.PageResult{List: list, Total: total, Page: query.Page, PageSize: query.PageSize})
}

// ListMine 会员查看自己的租借记录。
func (h *PeripheralRentalHandler) ListMine(c *gin.Context) {
	var query dto.RentalQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, 400, constants.CodeValidation, "租借查询参数校验失败："+err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	list, total, err := h.rentalService.ListMine(uid, &query)
	if err != nil {
		h.abort(c, err)
		return
	}
	response.OK(c, dto.PageResult{List: list, Total: total, Page: query.Page, PageSize: query.PageSize})
}

// rentalOperator 从登录上下文取处理人 ID 与姓名。
func rentalOperator(c *gin.Context) (uint, string) {
	operatorID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	opID, _ := operatorID.(uint)
	opName, _ := username.(string)
	return opID, opName
}

// abort 统一错误处理。
func (h *PeripheralRentalHandler) abort(c *gin.Context, err error) {
	var appErr *util.AppError
	if errors.As(err, &appErr) {
		response.Fail(c, httpStatusFor(appErr.Code), appErr.Code, appErr.Message)
		return
	}
	h.logger.Error(fmt.Sprintf("peripheral rental handler error: %v", err))
	response.Fail(c, 500, constants.CodeInternal, "服务器内部错误，请稍后重试")
}

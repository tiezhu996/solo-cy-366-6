package router

import (
	"github.com/gin-gonic/gin"

	"github.com/esportsbar/backend/internal/constants"
	"github.com/esportsbar/backend/internal/handler"
	"github.com/esportsbar/backend/internal/middleware"
)

// RegisterPeripheral 注册外设设备路由。
func RegisterPeripheral(rg *gin.RouterGroup, h *handler.PeripheralHandler, jwtSecret string) {
	peripherals := rg.Group("/peripherals", middleware.Auth(jwtSecret))
	{
		peripherals.GET("", h.List)
		peripherals.POST("", middleware.RBAC(constants.RoleAdmin, constants.RoleStaff), h.Create)
		peripherals.PUT("/:id/status", middleware.RBAC(constants.RoleAdmin, constants.RoleStaff), h.UpdateStatus)
	}
}

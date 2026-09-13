package router

import (
	"github.com/gin-gonic/gin"

	"github.com/esportsbar/backend/internal/constants"
	"github.com/esportsbar/backend/internal/handler"
	"github.com/esportsbar/backend/internal/middleware"
)

// RegisterPeripheralRental 注册外设租借路由。
func RegisterPeripheralRental(rg *gin.RouterGroup, h *handler.PeripheralRentalHandler, jwtSecret string) {
	rentals := rg.Group("/rentals", middleware.Auth(jwtSecret))
	{
		rentals.GET("", middleware.RBAC(constants.RoleAdmin, constants.RoleStaff), h.List)
		rentals.GET("/mine", h.ListMine)
		rentals.POST("", middleware.RBAC(constants.RoleAdmin, constants.RoleStaff), h.Create)
		rentals.POST("/:id/return", middleware.RBAC(constants.RoleAdmin, constants.RoleStaff), h.Return)
		rentals.POST("/:id/damage", middleware.RBAC(constants.RoleAdmin, constants.RoleStaff), h.Damage)
	}
}

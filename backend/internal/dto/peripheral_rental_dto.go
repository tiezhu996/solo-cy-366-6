package dto

import "time"

// CreateRentalReq 登记外设租借请求（店员为到场会员办理）。
type CreateRentalReq struct {
	UserID           uint      `json:"user_id" binding:"required"`
	PeripheralID     uint      `json:"peripheral_id" binding:"required"`
	Deposit          float64   `json:"deposit" binding:"required,gte=0"`
	ExpectedReturnAt time.Time `json:"expected_return_at" binding:"required"`
	Remark           string    `json:"remark" binding:"omitempty,max=255"`
}

// DamageRentalReq 损坏归还登记请求（说明、赔偿金额，处理人取当前登录店员）。
type DamageRentalReq struct {
	DamageDesc   string  `json:"damage_desc" binding:"required,max=255"`
	Compensation float64 `json:"compensation" binding:"required,gte=0"`
}

// RentalQuery 外设租借记录查询参数。
type RentalQuery struct {
	Page         int    `form:"page" binding:"omitempty,min=1"`
	PageSize     int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status       string `form:"status" binding:"omitempty,oneof=renting returned damaged"`
	UserID       uint   `form:"user_id"`
	PeripheralID uint   `form:"peripheral_id"`
}

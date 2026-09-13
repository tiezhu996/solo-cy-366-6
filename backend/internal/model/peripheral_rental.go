package model

import "time"

// PeripheralRental 外设租借与损坏赔付记录。
type PeripheralRental struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	RentalNo         string     `gorm:"size:40;uniqueIndex;not null" json:"rental_no"`
	UserID           uint       `gorm:"index;not null" json:"user_id"`
	PeripheralID     uint       `gorm:"index;not null" json:"peripheral_id"`
	DeviceNo         string     `gorm:"size:32;not null" json:"device_no"`
	DeviceType       string     `gorm:"size:16;not null" json:"device_type"`
	Deposit          float64    `gorm:"type:decimal(12,2);default:0" json:"deposit"`
	ExpectedReturnAt time.Time  `gorm:"not null" json:"expected_return_at"`
	ReturnedAt       *time.Time `json:"returned_at"`
	Status           string     `gorm:"size:16;default:renting" json:"status"` // renting/returned/damaged
	DamageDesc       string     `gorm:"size:255" json:"damage_desc"`
	Compensation     float64    `gorm:"type:decimal(12,2);default:0" json:"compensation"`
	HandlerID        uint       `gorm:"default:0" json:"handler_id"`
	HandlerName      string     `gorm:"size:64" json:"handler_name"`
	DebtAmount       float64    `gorm:"type:decimal(12,2);default:0" json:"debt_amount"`
	RefundAmount     float64    `gorm:"type:decimal(12,2);default:0" json:"refund_amount"`
	Remark           string     `gorm:"size:255" json:"remark"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName 指定表名。
func (PeripheralRental) TableName() string { return "peripheral_rentals" }

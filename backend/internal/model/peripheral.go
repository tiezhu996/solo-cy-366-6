package model

import "time"

// Peripheral 电竞馆外设设备（键盘/鼠标/耳机）。
type Peripheral struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DeviceNo   string    `gorm:"size:32;uniqueIndex;not null" json:"device_no"`
	DeviceType string    `gorm:"size:16;not null" json:"device_type"` // keyboard/mouse/headset
	Name       string    `gorm:"size:64;not null" json:"name"`
	Status     string    `gorm:"size:16;default:available" json:"status"` // available/rented/maintenance
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (Peripheral) TableName() string { return "peripherals" }

package dto

// CreatePeripheralReq 登记外设设备请求。
type CreatePeripheralReq struct {
	DeviceNo   string `json:"device_no" binding:"required,max=32"`
	DeviceType string `json:"device_type" binding:"required,oneof=keyboard mouse headset"`
	Name       string `json:"name" binding:"required,max=64"`
}

// UpdatePeripheralStatusReq 外设设备状态变更请求。
type UpdatePeripheralStatusReq struct {
	Status string `json:"status" binding:"required,oneof=available rented maintenance"`
}

// PeripheralQuery 外设设备查询参数。
type PeripheralQuery struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	DeviceType string `form:"device_type" binding:"omitempty,oneof=keyboard mouse headset"`
	Status     string `form:"status" binding:"omitempty,oneof=available rented maintenance"`
}

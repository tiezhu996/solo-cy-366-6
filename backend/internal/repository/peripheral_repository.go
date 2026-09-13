package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/esportsbar/backend/internal/model"
)

// PeripheralRepository 外设设备仓储。
type PeripheralRepository struct {
	db *gorm.DB
}

// NewPeripheralRepository 构造外设设备仓储。
func NewPeripheralRepository(db *gorm.DB) *PeripheralRepository {
	return &PeripheralRepository{db: db}
}

// Create 创建设备。
func (r *PeripheralRepository) Create(p *model.Peripheral) error {
	return r.db.Create(p).Error
}

// FindByID 查询设备。
func (r *PeripheralRepository) FindByID(id uint) (*model.Peripheral, error) {
	var p model.Peripheral
	err := r.db.First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

// FindByDeviceNo 按设备编号查询设备。
func (r *PeripheralRepository) FindByDeviceNo(deviceNo string) (*model.Peripheral, error) {
	var p model.Peripheral
	err := r.db.Where("device_no = ?", deviceNo).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

// Update 更新设备。
func (r *PeripheralRepository) Update(p *model.Peripheral) error {
	return r.db.Save(p).Error
}

// List 分页查询设备，支持类型与状态筛选（租借登记表单的可借设备下拉复用 status=available）。
func (r *PeripheralRepository) List(page, pageSize int, deviceType, status string) ([]model.Peripheral, int64, error) {
	var list []model.Peripheral
	var total int64
	query := r.db.Model(&model.Peripheral{})
	if deviceType != "" {
		query = query.Where("device_type = ?", deviceType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("device_type, device_no").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// LockByID 行锁查询设备（并发租借防重复借出使用）。
func (r *PeripheralRepository) LockByID(tx *gorm.DB, id uint) (*model.Peripheral, error) {
	var p model.Peripheral
	err := tx.Clauses(clauseLocking()).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, err
}

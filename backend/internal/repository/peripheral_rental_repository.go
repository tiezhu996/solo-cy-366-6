package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/esportsbar/backend/internal/model"
)

// PeripheralRentalRepository 外设租借记录仓储。
type PeripheralRentalRepository struct {
	db *gorm.DB
}

// NewPeripheralRentalRepository 构造外设租借记录仓储。
func NewPeripheralRentalRepository(db *gorm.DB) *PeripheralRentalRepository {
	return &PeripheralRentalRepository{db: db}
}

// CreateTx 事务内创建租借记录。
func (r *PeripheralRentalRepository) CreateTx(tx *gorm.DB, rental *model.PeripheralRental) error {
	return tx.Create(rental).Error
}

// FindByID 查询租借记录。
func (r *PeripheralRentalRepository) FindByID(id uint) (*model.PeripheralRental, error) {
	var rental model.PeripheralRental
	err := r.db.First(&rental, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &rental, err
}

// LockByID 行锁查询租借记录（归还/损坏赔付防重复处理使用）。
func (r *PeripheralRentalRepository) LockByID(tx *gorm.DB, id uint) (*model.PeripheralRental, error) {
	var rental model.PeripheralRental
	err := tx.Clauses(clauseLocking()).First(&rental, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &rental, err
}

// UpdateTx 事务内更新租借记录。
func (r *PeripheralRentalRepository) UpdateTx(tx *gorm.DB, rental *model.PeripheralRental) error {
	return tx.Save(rental).Error
}

// List 分页查询租借记录（店员列表与会员“我的租借”复用，会员侧固定传入 userID）。
func (r *PeripheralRentalRepository) List(page, pageSize int, status string, userID, peripheralID uint) ([]model.PeripheralRental, int64, error) {
	var list []model.PeripheralRental
	var total int64
	query := r.db.Model(&model.PeripheralRental{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if peripheralID > 0 {
		query = query.Where("peripheral_id = ?", peripheralID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// CountRentingByPeripheralTx 事务内统计设备在借记录数（重复借出兜底校验）。
func (r *PeripheralRentalRepository) CountRentingByPeripheralTx(tx *gorm.DB, peripheralID uint) (int64, error) {
	var cnt int64
	err := tx.Model(&model.PeripheralRental{}).
		Where("peripheral_id = ? AND status = ?", peripheralID, "renting").
		Count(&cnt).Error
	return cnt, err
}

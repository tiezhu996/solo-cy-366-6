package service

import (
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/esportsbar/backend/internal/constants"
	"github.com/esportsbar/backend/internal/dto"
	"github.com/esportsbar/backend/internal/model"
	"github.com/esportsbar/backend/internal/repository"
	"github.com/esportsbar/backend/internal/util"
)

// PeripheralService 外设设备服务。
type PeripheralService struct {
	peripheralRepo *repository.PeripheralRepository
	logger         *slog.Logger
}

// NewPeripheralService 构造外设设备服务。
func NewPeripheralService(peripheralRepo *repository.PeripheralRepository, logger *slog.Logger) *PeripheralService {
	return &PeripheralService{peripheralRepo: peripheralRepo, logger: logger}
}

// Create 登记外设设备：设备编号全局唯一。
func (s *PeripheralService) Create(req *dto.CreatePeripheralReq) (*model.Peripheral, error) {
	if _, err := s.peripheralRepo.FindByDeviceNo(req.DeviceNo); err == nil {
		return nil, util.NewAppError(constants.CodeConflict, fmt.Sprintf("设备编号 %s 已存在，请更换编号", req.DeviceNo))
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("peripheral find by device no: %w", err)
	}
	p := &model.Peripheral{
		DeviceNo:   req.DeviceNo,
		DeviceType: req.DeviceType,
		Name:       req.Name,
		Status:     constants.PeripheralAvailable,
	}
	if err := s.peripheralRepo.Create(p); err != nil {
		return nil, fmt.Errorf("peripheral create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTemplates["peripheral_create_ok"], p.DeviceNo, p.DeviceType))
	return p, nil
}

// UpdateStatus 设备状态流转：available<->maintenance 可手动变更，rented 只能由租借流程流转。
func (s *PeripheralService) UpdateStatus(id uint, req *dto.UpdatePeripheralStatusReq, operator string) (*model.Peripheral, error) {
	p, err := s.peripheralRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, "外设设备不存在")
		}
		return nil, fmt.Errorf("peripheral status find: %w", err)
	}
	from := p.Status
	if !allowedPeripheralTransition(from, req.Status) {
		return nil, util.NewAppError(constants.CodePeripheralBusy,
			fmt.Sprintf("设备状态不允许从 %s 变更为 %s，租借中的设备需先归还", util.StatusText(from), util.StatusText(req.Status)))
	}
	p.Status = req.Status
	if err := s.peripheralRepo.Update(p); err != nil {
		return nil, fmt.Errorf("peripheral status update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTemplates["peripheral_status_change"], p.ID, from, req.Status, operator))
	return p, nil
}

// List 分页查询设备（设备列表与租借登记“可借设备”下拉复用，status=available 即可借设备）。
func (s *PeripheralService) List(query *dto.PeripheralQuery) ([]model.Peripheral, int64, error) {
	page := query.Page
	pageSize := query.PageSize
	if page <= 0 {
		page = constants.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = constants.DefaultPageSize
	}
	return s.peripheralRepo.List(page, pageSize, query.DeviceType, query.Status)
}

// GetByID 查询设备详情。
func (s *PeripheralService) GetByID(id uint) (*model.Peripheral, error) {
	p, err := s.peripheralRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, "外设设备不存在")
		}
		return nil, fmt.Errorf("peripheral get: %w", err)
	}
	return p, nil
}

// LockForUpdate 事务内行锁设备，供租借服务复用。
func (s *PeripheralService) LockForUpdate(tx *gorm.DB, id uint) (*model.Peripheral, error) {
	return s.peripheralRepo.LockByID(tx, id)
}

// allowedPeripheralTransition 设备状态机：rented 状态只能由租借/归还流程改变，不允许手动流转。
func allowedPeripheralTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case constants.PeripheralAvailable:
		return to == constants.PeripheralMaintenance
	case constants.PeripheralMaintenance:
		return to == constants.PeripheralAvailable
	}
	return false
}

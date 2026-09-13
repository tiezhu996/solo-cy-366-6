package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/esportsbar/backend/internal/constants"
	"github.com/esportsbar/backend/internal/dto"
	"github.com/esportsbar/backend/internal/model"
	"github.com/esportsbar/backend/internal/repository"
	"github.com/esportsbar/backend/internal/util"
)

// PeripheralRentalService 外设租借与损坏赔付服务。
type PeripheralRentalService struct {
	rentalRepo        *repository.PeripheralRentalRepository
	peripheralService *PeripheralService
	userRepo          *repository.UserRepository
	db                *gorm.DB
	logger            *slog.Logger
}

// NewPeripheralRentalService 构造外设租借服务。
func NewPeripheralRentalService(
	rentalRepo *repository.PeripheralRentalRepository,
	peripheralService *PeripheralService,
	userRepo *repository.UserRepository,
	db *gorm.DB,
	logger *slog.Logger,
) *PeripheralRentalService {
	return &PeripheralRentalService{rentalRepo: rentalRepo, peripheralService: peripheralService, userRepo: userRepo, db: db, logger: logger}
}

// Create 登记租借：事务内锁定设备防重复借出、冻结会员押金、设备置为已借出。
func (s *PeripheralRentalService) Create(operatorID uint, req *dto.CreateRentalReq) (*model.PeripheralRental, error) {
	if !req.ExpectedReturnAt.After(time.Now()) {
		return nil, util.NewAppError(constants.CodeValidation, "预计归还时间必须晚于当前时间")
	}
	member, err := s.userRepo.FindByID(req.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeUserNotFound, "会员不存在，无法登记租借")
		}
		return nil, fmt.Errorf("rental find member: %w", err)
	}
	if member.Role != constants.RoleMember {
		return nil, util.NewAppError(constants.CodeValidation,
			fmt.Sprintf("租借对象必须是会员，用户 %s 的角色为 %s，不能作为租借对象", member.Username, util.RoleText(member.Role)))
	}
	peripheral, err := s.peripheralService.GetByID(req.PeripheralID)
	if err != nil {
		return nil, err
	}
	rental := &model.PeripheralRental{
		RentalNo:         genRentalNo(),
		UserID:           member.ID,
		PeripheralID:     peripheral.ID,
		DeviceNo:         peripheral.DeviceNo,
		DeviceType:       peripheral.DeviceType,
		Deposit:          req.Deposit,
		ExpectedReturnAt: req.ExpectedReturnAt,
		Status:           constants.RentalRenting,
		Remark:           req.Remark,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		locked, err := s.peripheralService.LockForUpdate(tx, req.PeripheralID)
		if err != nil {
			return err
		}
		if locked.Status != constants.PeripheralAvailable {
			return util.NewAppError(constants.CodePeripheralBusy,
				fmt.Sprintf("设备 %s 当前状态为 %s，不能重复借出", locked.DeviceNo, util.StatusText(locked.Status)))
		}
		renting, err := s.rentalRepo.CountRentingByPeripheralTx(tx, req.PeripheralID)
		if err != nil {
			return err
		}
		if renting > 0 {
			return util.NewAppError(constants.CodePeripheralBusy, fmt.Sprintf("设备 %s 存在在借记录，不能重复借出", locked.DeviceNo))
		}
		if req.Deposit > 0 {
			if err := s.userRepo.UpdateBalanceTx(tx, member.ID, -req.Deposit); err != nil {
				if errors.Is(err, repository.ErrConflict) {
					return util.NewAppError(constants.CodeInsufficient, "会员余额不足，无法冻结押金，请先充值")
				}
				return err
			}
		}
		locked.Status = constants.PeripheralRented
		if err := tx.Save(locked).Error; err != nil {
			return err
		}
		return s.rentalRepo.CreateTx(tx, rental)
	})
	if err != nil {
		return nil, fmt.Errorf("rental create tx: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTemplates["rental_create_ok"], rental.ID, member.ID, peripheral.DeviceNo, req.Deposit, operatorID))
	return rental, nil
}

// Return 完好归还：事务内校验在借状态，释放押金回会员余额，设备恢复可借。
func (s *PeripheralRentalService) Return(id uint, operatorID uint, operatorName string) (*model.PeripheralRental, error) {
	now := time.Now()
	var rental *model.PeripheralRental
	err := s.db.Transaction(func(tx *gorm.DB) error {
		locked, err := s.rentalRepo.LockByID(tx, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, "租借记录不存在")
			}
			return err
		}
		if locked.Status != constants.RentalRenting {
			return util.NewAppError(constants.CodeRentalState,
				fmt.Sprintf("租借记录当前状态为 %s，已归还的记录不能再次处理", util.StatusText(locked.Status)))
		}
		if locked.Deposit > 0 {
			if err := s.userRepo.UpdateBalanceTx(tx, locked.UserID, locked.Deposit); err != nil {
				return err
			}
		}
		device, err := s.peripheralService.LockForUpdate(tx, locked.PeripheralID)
		if err != nil {
			return err
		}
		device.Status = constants.PeripheralAvailable
		if err := tx.Save(device).Error; err != nil {
			return err
		}
		locked.Status = constants.RentalReturned
		locked.ReturnedAt = &now
		locked.RefundAmount = locked.Deposit
		locked.HandlerID = operatorID
		locked.HandlerName = operatorName
		if err := s.rentalRepo.UpdateTx(tx, locked); err != nil {
			return err
		}
		rental = locked
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("rental return tx: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTemplates["rental_return_ok"], id, rental.RefundAmount, operatorName))
	return rental, nil
}

// Damage 损坏归还登记：事务内校验在借状态，押金抵扣赔偿，不足部分计入会员欠款，设备转维护。
func (s *PeripheralRentalService) Damage(id uint, operatorID uint, operatorName string, req *dto.DamageRentalReq) (*model.PeripheralRental, error) {
	now := time.Now()
	var rental *model.PeripheralRental
	err := s.db.Transaction(func(tx *gorm.DB) error {
		locked, err := s.rentalRepo.LockByID(tx, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, "租借记录不存在")
			}
			return err
		}
		if locked.Status != constants.RentalRenting {
			return util.NewAppError(constants.CodeRentalState,
				fmt.Sprintf("租借记录当前状态为 %s，赔偿已完成不能重复扣款", util.StatusText(locked.Status)))
		}
		refund, debt := splitDamageCompensation(locked.Deposit, req.Compensation)
		if refund > 0 {
			if err := s.userRepo.UpdateBalanceTx(tx, locked.UserID, refund); err != nil {
				return err
			}
		}
		if debt > 0 {
			if err := s.userRepo.UpdateDebtTx(tx, locked.UserID, debt); err != nil {
				return err
			}
		}
		device, err := s.peripheralService.LockForUpdate(tx, locked.PeripheralID)
		if err != nil {
			return err
		}
		device.Status = constants.PeripheralMaintenance
		if err := tx.Save(device).Error; err != nil {
			return err
		}
		locked.Status = constants.RentalDamaged
		locked.ReturnedAt = &now
		locked.DamageDesc = req.DamageDesc
		locked.Compensation = req.Compensation
		locked.HandlerID = operatorID
		locked.HandlerName = operatorName
		locked.DebtAmount = debt
		locked.RefundAmount = refund
		if err := s.rentalRepo.UpdateTx(tx, locked); err != nil {
			return err
		}
		rental = locked
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("rental damage tx: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogTemplates["rental_damage_ok"], id, rental.Compensation, rental.DebtAmount, operatorName))
	return rental, nil
}

// List 分页查询租借记录（店员/管理员）。
func (s *PeripheralRentalService) List(query *dto.RentalQuery) ([]model.PeripheralRental, int64, error) {
	page, pageSize := normalizeRentalPage(query)
	return s.rentalRepo.List(page, pageSize, query.Status, query.UserID, query.PeripheralID)
}

// ListMine 会员查看自己的租借记录（与 List 复用同一仓储查询，强制当前会员过滤）。
func (s *PeripheralRentalService) ListMine(userID uint, query *dto.RentalQuery) ([]model.PeripheralRental, int64, error) {
	page, pageSize := normalizeRentalPage(query)
	return s.rentalRepo.List(page, pageSize, query.Status, userID, 0)
}

// normalizeRentalPage 归一化分页参数。
func normalizeRentalPage(query *dto.RentalQuery) (int, int) {
	page := query.Page
	pageSize := query.PageSize
	if page <= 0 {
		page = constants.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = constants.DefaultPageSize
	}
	return page, pageSize
}

// genRentalNo 生成租借单号。
func genRentalNo() string {
	return fmt.Sprintf("PR%s%06d", time.Now().Format("20060102150405"), time.Now().UnixNano()%1000000)
}

// splitDamageCompensation 损坏赔付拆分：押金先抵扣赔偿，剩余押金退还会员，不足部分计入会员欠款。
func splitDamageCompensation(deposit, compensation float64) (refund, debt float64) {
	refund = deposit - compensation
	if refund < 0 {
		refund = 0
	}
	debt = compensation - deposit
	if debt < 0 {
		debt = 0
	}
	return refund, debt
}

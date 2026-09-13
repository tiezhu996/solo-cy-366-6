package service

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/esportsbar/backend/internal/constants"
	"github.com/esportsbar/backend/internal/dto"
	"github.com/esportsbar/backend/internal/repository"
	"github.com/esportsbar/backend/internal/util"
)

// newRentalMockService 基于 sqlmock 构造完整租借服务依赖链。
func newRentalMockService(t *testing.T) (*PeripheralRentalService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new error: %v", err)
	}
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm open error: %v", err)
	}
	rentalRepo := repository.NewPeripheralRentalRepository(gdb)
	peripheralRepo := repository.NewPeripheralRepository(gdb)
	userRepo := repository.NewUserRepository(gdb)
	peripheralSvc := NewPeripheralService(peripheralRepo, newTestLogger())
	svc := NewPeripheralRentalService(rentalRepo, peripheralSvc, userRepo, gdb, newTestLogger())
	return svc, mock
}

func rentalRow(id, userID, peripheralID uint, deposit float64, status string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "rental_no", "user_id", "peripheral_id", "device_no", "device_type", "deposit", "expected_return_at", "status"}).
		AddRow(id, "PR20260913001", userID, peripheralID, "KB-001", "keyboard", deposit, time.Now().Add(24*time.Hour), status)
}

// TestRentalCreateOK 登记租借完整事务：锁设备、查在借、冻结押金、设备置租出、写记录。
func TestRentalCreateOK(t *testing.T) {
	svc, mock := newRentalMockService(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE `users`.`id` = ? ORDER BY `users`.`id` LIMIT ?")).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "balance"}).AddRow(2, "member", 200))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `peripherals` WHERE `peripherals`.`id` = ? ORDER BY `peripherals`.`id` LIMIT ?")).
		WithArgs(3, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "device_no", "device_type", "name", "status"}).AddRow(3, "KB-001", "keyboard", "机械键盘", "available"))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `peripherals` WHERE .* FOR UPDATE").
		WithArgs(3, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "device_no", "device_type", "name", "status"}).AddRow(3, "KB-001", "keyboard", "机械键盘", "available"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `peripheral_rentals` WHERE peripheral_id = ? AND status = ?")).
		WithArgs(3, "renting").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT \\* FROM `users` WHERE .* FOR UPDATE").
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "balance"}).AddRow(2, "member", 200))
	mock.ExpectExec("UPDATE `users` SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `peripherals` SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO `peripheral_rentals`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	rental, err := svc.Create(1, &dto.CreateRentalReq{
		UserID:           2,
		PeripheralID:     3,
		Deposit:          100,
		ExpectedReturnAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if rental.Status != constants.RentalRenting || rental.DeviceNo != "KB-001" || rental.ID != 1 {
		t.Fatalf("unexpected rental: %+v", rental)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

// TestRentalCreateDeviceRented 同一设备不能重复借出：设备非可借状态时事务回滚。
func TestRentalCreateDeviceRented(t *testing.T) {
	svc, mock := newRentalMockService(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE `users`.`id` = ? ORDER BY `users`.`id` LIMIT ?")).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "balance"}).AddRow(2, "member", 200))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `peripherals` WHERE `peripherals`.`id` = ? ORDER BY `peripherals`.`id` LIMIT ?")).
		WithArgs(3, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "device_no", "device_type", "name", "status"}).AddRow(3, "KB-001", "keyboard", "机械键盘", "rented"))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `peripherals` WHERE .* FOR UPDATE").
		WithArgs(3, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "device_no", "device_type", "name", "status"}).AddRow(3, "KB-001", "keyboard", "机械键盘", "rented"))
	mock.ExpectRollback()

	_, err := svc.Create(1, &dto.CreateRentalReq{
		UserID:           2,
		PeripheralID:     3,
		Deposit:          100,
		ExpectedReturnAt: time.Now().Add(24 * time.Hour),
	})
	assertAppErrorCode(t, err, constants.CodePeripheralBusy)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

// TestRentalReturnAlreadyProcessed 已归还记录不能再次处理。
func TestRentalReturnAlreadyProcessed(t *testing.T) {
	svc, mock := newRentalMockService(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `peripheral_rentals` WHERE .* FOR UPDATE").
		WithArgs(9, 1).
		WillReturnRows(rentalRow(9, 2, 3, 100, constants.RentalReturned))
	mock.ExpectRollback()

	_, err := svc.Return(9, 1, "admin")
	assertAppErrorCode(t, err, constants.CodeRentalState)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

// TestRentalDamageSettleOK 损坏赔付完整事务：押金抵扣、不足计欠款、设备转维护、记录结案。
func TestRentalDamageSettleOK(t *testing.T) {
	svc, mock := newRentalMockService(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `peripheral_rentals` WHERE .* FOR UPDATE").
		WithArgs(9, 1).
		WillReturnRows(rentalRow(9, 2, 3, 100, constants.RentalRenting))
	mock.ExpectQuery("SELECT \\* FROM `users` WHERE .* FOR UPDATE").
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "balance", "debt"}).AddRow(2, "member", 0, 0))
	mock.ExpectExec("UPDATE `users` SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT \\* FROM `peripherals` WHERE .* FOR UPDATE").
		WithArgs(3, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "device_no", "device_type", "name", "status"}).AddRow(3, "KB-001", "keyboard", "机械键盘", "rented"))
	mock.ExpectExec("UPDATE `peripherals` SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `peripheral_rentals` SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	rental, err := svc.Damage(9, 1, "admin", &dto.DamageRentalReq{DamageDesc: "键盘进水", Compensation: 150})
	if err != nil {
		t.Fatalf("Damage error: %v", err)
	}
	if rental.Status != constants.RentalDamaged {
		t.Fatalf("unexpected status: %s", rental.Status)
	}
	if rental.DebtAmount != 50 || rental.RefundAmount != 0 {
		t.Fatalf("unexpected split: debt=%v refund=%v", rental.DebtAmount, rental.RefundAmount)
	}
	if rental.HandlerName != "admin" || rental.ReturnedAt == nil {
		t.Fatalf("unexpected handler/returned_at: %+v", rental)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

// TestRentalDamageAlreadySettled 赔偿完成不能重复扣款。
func TestRentalDamageAlreadySettled(t *testing.T) {
	svc, mock := newRentalMockService(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT \\* FROM `peripheral_rentals` WHERE .* FOR UPDATE").
		WithArgs(9, 1).
		WillReturnRows(rentalRow(9, 2, 3, 100, constants.RentalDamaged))
	mock.ExpectRollback()

	_, err := svc.Damage(9, 1, "admin", &dto.DamageRentalReq{DamageDesc: "重复登记", Compensation: 150})
	assertAppErrorCode(t, err, constants.CodeRentalState)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

// assertAppErrorCode 断言错误为指定业务错误码。
func assertAppErrorCode(t *testing.T, err error, code int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d, got nil", code)
	}
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %v", err)
	}
	if appErr.Code != code {
		t.Fatalf("expected code %d, got %d (%s)", code, appErr.Code, appErr.Message)
	}
}

package repository

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPeripheralRepositoryFindByDeviceNo(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewPeripheralRepository(gdb)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `peripherals` WHERE device_no = ? ORDER BY `peripherals`.`id` LIMIT ?")).
		WithArgs("KB-001", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "device_no", "device_type", "status"}).AddRow(1, "KB-001", "keyboard", "available"))
	p, err := repo.FindByDeviceNo("KB-001")
	if err != nil {
		t.Fatalf("FindByDeviceNo error: %v", err)
	}
	if p.ID != 1 || p.DeviceNo != "KB-001" {
		t.Fatalf("unexpected peripheral: %+v", p)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPeripheralRepositoryFindByDeviceNoNotFound(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewPeripheralRepository(gdb)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `peripherals` WHERE device_no = ? ORDER BY `peripherals`.`id` LIMIT ?")).
		WithArgs("XX-999", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	if _, err := repo.FindByDeviceNo("XX-999"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPeripheralRepositoryList(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewPeripheralRepository(gdb)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `peripherals` WHERE device_type = ? AND status = ?")).
		WithArgs("keyboard", "available").
		WillReturnRows(sqlmockRowsCount(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `peripherals` WHERE device_type = ? AND status = ? ORDER BY device_type, device_no LIMIT ?")).
		WithArgs("keyboard", "available", 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "device_no", "device_type", "status"}).AddRow(1, "KB-001", "keyboard", "available"))
	list, total, err := repo.List(1, 10, "keyboard", "available")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("unexpected result total=%d len=%d", total, len(list))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPeripheralRentalRepositoryListMine(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewPeripheralRentalRepository(gdb)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `peripheral_rentals` WHERE status = ? AND user_id = ?")).
		WithArgs("renting", 7).
		WillReturnRows(sqlmockRowsCount(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `peripheral_rentals` WHERE status = ? AND user_id = ? ORDER BY id DESC LIMIT ?")).
		WithArgs("renting", 7, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_no", "status"}).AddRow(9, 7, "MS-001", "renting"))
	list, total, err := repo.List(1, 10, "renting", 7, 0)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].UserID != 7 {
		t.Fatalf("unexpected result total=%d list=%+v", total, list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

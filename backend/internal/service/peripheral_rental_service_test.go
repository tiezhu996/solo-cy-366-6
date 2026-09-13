package service

import (
	"testing"

	"github.com/esportsbar/backend/internal/constants"
)

// TestPeripheralStatusTransition 外设设备状态机白盒测试：rented 只能由租借流程流转。
func TestPeripheralStatusTransition(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{name: "available_to_maintenance", from: constants.PeripheralAvailable, to: constants.PeripheralMaintenance, want: true},
		{name: "maintenance_to_available", from: constants.PeripheralMaintenance, to: constants.PeripheralAvailable, want: true},
		{name: "available_to_rented_manual", from: constants.PeripheralAvailable, to: constants.PeripheralRented, want: false},
		{name: "rented_to_available_manual", from: constants.PeripheralRented, to: constants.PeripheralAvailable, want: false},
		{name: "rented_to_maintenance_manual", from: constants.PeripheralRented, to: constants.PeripheralMaintenance, want: false},
		{name: "available_to_available", from: constants.PeripheralAvailable, to: constants.PeripheralAvailable, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allowedPeripheralTransition(tc.from, tc.to); got != tc.want {
				t.Fatalf("allowedPeripheralTransition(%s->%s) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

// TestSplitDamageCompensation 损坏赔付拆分：押金抵扣赔偿，剩余退还、不足计欠款。
func TestSplitDamageCompensation(t *testing.T) {
	cases := []struct {
		name         string
		deposit      float64
		compensation float64
		wantRefund   float64
		wantDebt     float64
	}{
		{name: "compensation_exceeds_deposit", deposit: 100, compensation: 150, wantRefund: 0, wantDebt: 50},
		{name: "compensation_below_deposit", deposit: 100, compensation: 60, wantRefund: 40, wantDebt: 0},
		{name: "compensation_equals_deposit", deposit: 100, compensation: 100, wantRefund: 0, wantDebt: 0},
		{name: "zero_compensation_full_refund", deposit: 100, compensation: 0, wantRefund: 100, wantDebt: 0},
		{name: "zero_deposit_full_debt", deposit: 0, compensation: 80, wantRefund: 0, wantDebt: 80},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refund, debt := splitDamageCompensation(tc.deposit, tc.compensation)
			if refund != tc.wantRefund || debt != tc.wantDebt {
				t.Fatalf("splitDamageCompensation(%v, %v) = (%v, %v), want (%v, %v)",
					tc.deposit, tc.compensation, refund, debt, tc.wantRefund, tc.wantDebt)
			}
		})
	}
}

// TestRentalStatusEnum 租借状态枚举合法性。
func TestRentalStatusEnum(t *testing.T) {
	for _, s := range []string{constants.RentalRenting, constants.RentalReturned, constants.RentalDamaged} {
		if !constants.IsValidRentalStatus(s) {
			t.Fatalf("IsValidRentalStatus(%q) = false, want true", s)
		}
	}
	if constants.IsValidRentalStatus("settled") {
		t.Fatalf("IsValidRentalStatus(settled) = true, want false")
	}
}

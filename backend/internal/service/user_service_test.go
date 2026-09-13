package service

import (
	"testing"

	"github.com/esportsbar/backend/internal/model"
)

// TestToUserItemMapsDebt 用户响应对象必须携带欠款字段（会员资料/登录/注册均复用该映射）。
func TestToUserItemMapsDebt(t *testing.T) {
	u := &model.User{
		ID:       2,
		Username: "member",
		Nickname: "体验会员",
		Role:     "member",
		Balance:  40,
		Debt:     50,
		Status:   "active",
	}
	item := toUserItem(u)
	if item.Debt != 50 {
		t.Fatalf("toUserItem Debt = %v, want 50", item.Debt)
	}
	if item.Balance != 40 {
		t.Fatalf("toUserItem Balance = %v, want 40", item.Balance)
	}
}

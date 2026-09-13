package apitest

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestMemberDebtShownAfterProfileRefresh 会员页欠款显示：损坏赔付后，资料刷新（GET /auth/profile）返回真实欠款，
// 且会员页租借记录（GET /rentals/mine）同步可见赔付结果。
func TestMemberDebtShownAfterProfileRefresh(t *testing.T) {
	app := newTestApp(t)
	staff := app.seedUser("staff_debt", "staff", 0)
	member := app.seedUser("member_debt", "member", 0)
	staffToken := app.login(staff.Username)
	memberToken := app.login(member.Username)

	app.rechargeOK(staffToken, member.ID, 200)
	peripheralID := app.createPeripheral(staffToken, "T-KB-DEBT")
	rentalID := app.createRentalOK(staffToken, member.ID, peripheralID, 100)

	// 押金冻结后：余额 200-100=100，欠款 0
	p := app.profile(memberToken)
	if p.Balance != 100 || p.Debt != 0 {
		t.Fatalf("会员页账户区显示异常: 押金冻结后 GET /api/v1/auth/profile 期望 balance=100 debt=0, 实际 balance=%v debt=%v", p.Balance, p.Debt)
	}

	// 损坏赔付：赔偿 150，押金 100 全额抵扣，差额 50 计入欠款
	status, resp, raw := app.do("POST", fmt.Sprintf("/api/v1/rentals/%d/damage", rentalID), staffToken, map[string]any{
		"damage_desc": "键盘进水，轴体损坏", "compensation": 150,
	})
	if status != 200 || resp.Code != 0 {
		t.Fatalf("损坏赔付登记接口异常: POST /api/v1/rentals/%d/damage status=%d body=%s", rentalID, status, raw)
	}

	// 资料刷新：欠款 50，押金全额抵扣不退回（余额仍为 100）
	p = app.profile(memberToken)
	if p.Debt != 50 {
		t.Fatalf("会员页欠款显示异常: 损坏赔付150(押金100)后 GET /api/v1/auth/profile 期望 debt=50, 实际 debt=%v (balance=%v)", p.Debt, p.Balance)
	}
	if p.Balance != 100 {
		t.Fatalf("会员页余额显示异常: 押金全额抵扣赔偿不应退款, GET /api/v1/auth/profile 期望 balance=100, 实际 balance=%v", p.Balance)
	}

	// 会员页租借记录同步可见赔付结果
	status, resp, raw = app.do("GET", "/api/v1/rentals/mine?page=1&page_size=10", memberToken, nil)
	if status != 200 || resp.Code != 0 {
		t.Fatalf("我的租借记录接口异常: GET /api/v1/rentals/mine status=%d body=%s", status, raw)
	}
	var mine struct {
		List []struct {
			ID         uint    `json:"id"`
			Status     string  `json:"status"`
			DebtAmount float64 `json:"debt_amount"`
		} `json:"list"`
	}
	if err := json.Unmarshal(resp.Data, &mine); err != nil {
		t.Fatalf("我的租借记录响应解析失败: %v, body=%s", err, raw)
	}
	found := false
	for _, r := range mine.List {
		if r.ID == rentalID {
			found = true
			if r.Status != "damaged" || r.DebtAmount != 50 {
				t.Fatalf("会员页租借记录显示异常: GET /api/v1/rentals/mine 记录#%d 期望 status=damaged debt_amount=50, 实际 status=%s debt_amount=%v", rentalID, r.Status, r.DebtAmount)
			}
		}
	}
	if !found {
		t.Fatalf("会员页租借记录缺失: GET /api/v1/rentals/mine 未返回记录#%d, body=%s", rentalID, raw)
	}
}

// TestRentalRejectsNonMemberTarget 租借对象必须限于会员：管理员/店员不能被登记为租借人。
func TestRentalRejectsNonMemberTarget(t *testing.T) {
	app := newTestApp(t)
	staff := app.seedUser("staff_role", "staff", 0)
	admin := app.seedUser("admin_role", "admin", 0)
	colleague := app.seedUser("staff_role2", "staff", 0)
	staffToken := app.login(staff.Username)
	peripheralID := app.createPeripheral(staffToken, "T-KB-ROLE")

	for _, target := range []*struct {
		id   uint
		role string
	}{{admin.ID, "admin"}, {colleague.ID, "staff"}} {
		status, resp, raw := app.do("POST", "/api/v1/rentals", staffToken, map[string]any{
			"user_id": target.id, "peripheral_id": peripheralID, "deposit": 50,
			"expected_return_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		})
		if status != 400 || resp.Code == 0 {
			t.Fatalf("租借登记应拒绝非会员对象: POST /api/v1/rentals user_id=%d(role=%s) 期望400业务错误, 实际 status=%d body=%s", target.id, target.role, status, raw)
		}
		if !strings.Contains(resp.Message, "会员") {
			t.Fatalf("拒绝文案应提示租借对象须为会员: POST /api/v1/rentals user_id=%d(role=%s), 实际 message=%q", target.id, target.role, resp.Message)
		}
	}

	// 拒绝后设备保持可借、不产生租借记录
	if s := app.deviceStatus(staffToken, peripheralID); s != "available" {
		t.Fatalf("非会员租借被拒后设备状态应保持可借: GET /api/v1/peripherals 期望 available, 实际 %s", s)
	}
	status, resp, raw := app.do("GET", "/api/v1/rentals?page=1&page_size=10", staffToken, nil)
	if status != 200 || resp.Code != 0 {
		t.Fatalf("租借记录列表接口异常: GET /api/v1/rentals status=%d body=%s", status, raw)
	}
	var page struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(resp.Data, &page); err != nil || page.Total != 0 {
		t.Fatalf("非会员租借被拒后不应产生租借记录: GET /api/v1/rentals 期望 total=0, 实际 body=%s", raw)
	}
}

// TestRentalLifecycleAndDuplicateGuards 正常借出 → 重复借出拒绝 → 完好归还（押金释放）→ 重复处理拒绝。
func TestRentalLifecycleAndDuplicateGuards(t *testing.T) {
	app := newTestApp(t)
	staff := app.seedUser("staff_life", "staff", 0)
	member := app.seedUser("member_life", "member", 0)
	staffToken := app.login(staff.Username)
	memberToken := app.login(member.Username)

	app.rechargeOK(staffToken, member.ID, 300)
	peripheralID := app.createPeripheral(staffToken, "T-KB-LIFE")
	rentalID := app.createRentalOK(staffToken, member.ID, peripheralID, 100)

	// 借出成功：设备已借出，押金冻结（300-100=200）
	if s := app.deviceStatus(staffToken, peripheralID); s != "rented" {
		t.Fatalf("借出后设备状态应为已借出: GET /api/v1/peripherals 期望 rented, 实际 %s", s)
	}
	if p := app.profile(memberToken); p.Balance != 200 {
		t.Fatalf("会员页余额显示异常: 押金100冻结后 GET /api/v1/auth/profile 期望 balance=200, 实际 %v", p.Balance)
	}

	// 同一设备重复借出 → 拒绝
	status, resp, raw := app.do("POST", "/api/v1/rentals", staffToken, map[string]any{
		"user_id": member.ID, "peripheral_id": peripheralID, "deposit": 100,
		"expected_return_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	if status != 409 || resp.Code == 0 {
		t.Fatalf("同一设备重复借出应被拒绝: POST /api/v1/rentals 期望409, 实际 status=%d body=%s", status, raw)
	}

	// 完好归还 → 押金释放回余额，设备恢复可借
	status, resp, raw = app.do("POST", fmt.Sprintf("/api/v1/rentals/%d/return", rentalID), staffToken, nil)
	if status != 200 || resp.Code != 0 {
		t.Fatalf("完好归还接口异常: POST /api/v1/rentals/%d/return status=%d body=%s", rentalID, status, raw)
	}
	if p := app.profile(memberToken); p.Balance != 300 {
		t.Fatalf("会员页余额显示异常: 押金释放后 GET /api/v1/auth/profile 期望 balance=300, 实际 %v", p.Balance)
	}
	if s := app.deviceStatus(staffToken, peripheralID); s != "available" {
		t.Fatalf("归还后设备状态应恢复可借: GET /api/v1/peripherals 期望 available, 实际 %s", s)
	}

	// 已归还记录重复归还 → 拒绝
	status, resp, raw = app.do("POST", fmt.Sprintf("/api/v1/rentals/%d/return", rentalID), staffToken, nil)
	if status != 409 || resp.Code == 0 {
		t.Fatalf("已归还记录不能再次处理: POST /api/v1/rentals/%d/return 期望409, 实际 status=%d body=%s", rentalID, status, raw)
	}

	// 已归还记录再登记损坏 → 拒绝
	status, resp, raw = app.do("POST", fmt.Sprintf("/api/v1/rentals/%d/damage", rentalID), staffToken, map[string]any{
		"damage_desc": "事后补报损坏", "compensation": 80,
	})
	if status != 409 || resp.Code == 0 {
		t.Fatalf("已归还记录不能再登记损坏: POST /api/v1/rentals/%d/damage 期望409, 实际 status=%d body=%s", rentalID, status, raw)
	}

	// 重复处理被拒后账户不得变动
	if p := app.profile(memberToken); p.Balance != 300 || p.Debt != 0 {
		t.Fatalf("重复处理被拒后账户不应变动: GET /api/v1/auth/profile 期望 balance=300 debt=0, 实际 balance=%v debt=%v", p.Balance, p.Debt)
	}
}

// TestDamageSettleAndNoDoubleCharge 损坏赔付（押金抵扣+欠款入账+设备转维护）→ 重复扣款拒绝。
func TestDamageSettleAndNoDoubleCharge(t *testing.T) {
	app := newTestApp(t)
	staff := app.seedUser("staff_dmg", "staff", 0)
	member := app.seedUser("member_dmg", "member", 0)
	staffToken := app.login(staff.Username)
	memberToken := app.login(member.Username)

	app.rechargeOK(staffToken, member.ID, 200)
	peripheralID := app.createPeripheral(staffToken, "T-KB-DMG")
	rentalID := app.createRentalOK(staffToken, member.ID, peripheralID, 100)

	// 损坏赔付：赔偿 150，押金 100 抵扣，欠款 50，设备转维护
	status, resp, raw := app.do("POST", fmt.Sprintf("/api/v1/rentals/%d/damage", rentalID), staffToken, map[string]any{
		"damage_desc": "键盘进水，轴体损坏", "compensation": 150,
	})
	if status != 200 || resp.Code != 0 {
		t.Fatalf("损坏赔付登记接口异常: POST /api/v1/rentals/%d/damage status=%d body=%s", rentalID, status, raw)
	}
	p := app.profile(memberToken)
	if p.Debt != 50 || p.Balance != 100 {
		t.Fatalf("会员页账户显示异常: 赔付后 GET /api/v1/auth/profile 期望 debt=50 balance=100, 实际 debt=%v balance=%v", p.Debt, p.Balance)
	}
	if s := app.deviceStatus(staffToken, peripheralID); s != "maintenance" {
		t.Fatalf("损坏设备应转入维护: GET /api/v1/peripherals 期望 maintenance, 实际 %s", s)
	}

	// 重复扣款拒绝：欠款与余额不得再次变动
	status, resp, raw = app.do("POST", fmt.Sprintf("/api/v1/rentals/%d/damage", rentalID), staffToken, map[string]any{
		"damage_desc": "重复登记", "compensation": 150,
	})
	if status != 409 || resp.Code == 0 {
		t.Fatalf("赔偿完成不能重复扣款: POST /api/v1/rentals/%d/damage 期望409, 实际 status=%d body=%s", rentalID, status, raw)
	}
	p = app.profile(memberToken)
	if p.Debt != 50 || p.Balance != 100 {
		t.Fatalf("重复扣款被拒后账户不应变动: GET /api/v1/auth/profile 期望 debt=50 balance=100, 实际 debt=%v balance=%v", p.Debt, p.Balance)
	}

	// 已赔付结案记录不能再归还
	status, resp, raw = app.do("POST", fmt.Sprintf("/api/v1/rentals/%d/return", rentalID), staffToken, nil)
	if status != 409 || resp.Code == 0 {
		t.Fatalf("已赔付结案记录不能再归还: POST /api/v1/rentals/%d/return 期望409, 实际 status=%d body=%s", rentalID, status, raw)
	}
}

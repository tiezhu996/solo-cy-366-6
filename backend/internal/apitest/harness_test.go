package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/esportsbar/backend/internal/database"
	"github.com/esportsbar/backend/internal/handler"
	"github.com/esportsbar/backend/internal/middleware"
	"github.com/esportsbar/backend/internal/model"
	"github.com/esportsbar/backend/internal/repository"
	"github.com/esportsbar/backend/internal/router"
	"github.com/esportsbar/backend/internal/service"
	"github.com/esportsbar/backend/internal/util"
)

const (
	testJWTSecret = "apitest-secret"
	testPassword  = "test123456"
)

// testApp 集成测试应用：真实 Gin 引擎 + 独立内存 SQLite 实例。
type testApp struct {
	t      *testing.T
	engine *gin.Engine
	db     *gorm.DB
}

// apiResp 统一响应体（与 pkg/response.Body 对齐）。
type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// newTestApp 装配测试应用：独立内存库 + 真实中间件与路由。
// 限流中间件依赖 Redis，与租借闭环无关，测试环境不装配。
func newTestApp(t *testing.T) *testApp {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()))
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := database.Migrate(gdb); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	// 常驻一条连接保证内存库存活；测试结束关闭连接池，内存库随之销毁，清理全部痕迹。
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("获取连接池失败: %v", err)
	}
	keepAlive, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("获取常驻连接失败: %v", err)
	}
	t.Cleanup(func() {
		_ = keepAlive.Close()
		_ = sqlDB.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	userRepo := repository.NewUserRepository(gdb)
	rechargeRepo := repository.NewRechargeRepository(gdb)
	packageRepo := repository.NewTimePackageRepository(gdb)
	userPkgRepo := repository.NewUserPackageRepository(gdb)
	orderRepo := repository.NewPackageOrderRepository(gdb)
	peripheralRepo := repository.NewPeripheralRepository(gdb)
	rentalRepo := repository.NewPeripheralRentalRepository(gdb)
	auditRepo := repository.NewAuditRepository(gdb)

	authService := service.NewAuthService(userRepo, logger, testJWTSecret, 86400)
	userService := service.NewUserService(userRepo, logger)
	rechargeService := service.NewRechargeService(userRepo, rechargeRepo, packageRepo, userPkgRepo, orderRepo, logger)
	peripheralService := service.NewPeripheralService(peripheralRepo, logger)
	rentalService := service.NewPeripheralRentalService(rentalRepo, peripheralService, userRepo, gdb, logger)
	auditService := service.NewAuditService(auditRepo, logger)

	authHandler := handler.NewAuthHandler(authService, userService, logger)
	rechargeHandler := handler.NewRechargeHandler(rechargeService, logger)
	peripheralHandler := handler.NewPeripheralHandler(peripheralService, logger)
	rentalHandler := handler.NewPeripheralRentalHandler(rentalService, logger)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.ErrorHandler(logger), middleware.Logger(logger))
	api := engine.Group("/api/v1")
	api.Use(middleware.Audit(auditService))
	router.RegisterAuth(api, authHandler, testJWTSecret)
	router.RegisterRecharge(api, rechargeHandler, testJWTSecret)
	router.RegisterPeripheral(api, peripheralHandler, testJWTSecret)
	router.RegisterPeripheralRental(api, rentalHandler, testJWTSecret)

	return &testApp{t: t, engine: engine, db: gdb}
}

// do 发起 HTTP 请求并解析统一响应。
func (a *testApp) do(method, path, token string, body any) (int, *apiResp, []byte) {
	a.t.Helper()
	var reader io.Reader
	if body != nil {
		bs, err := json.Marshal(body)
		if err != nil {
			a.t.Fatalf("构造请求体失败: %v", err)
		}
		reader = bytes.NewReader(bs)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.engine.ServeHTTP(w, req)
	raw := w.Body.Bytes()
	resp := &apiResp{}
	if err := json.Unmarshal(raw, resp); err != nil {
		a.t.Fatalf("%s %s 响应不是合法JSON: %v, body=%s", method, path, err, raw)
	}
	return w.Code, resp, raw
}

// seedUser 自建测试账号（数据隔离：每个测试实例使用独立内存库）。
func (a *testApp) seedUser(username, role string, balance float64) *model.User {
	a.t.Helper()
	hash, err := util.HashPassword(testPassword)
	if err != nil {
		a.t.Fatalf("密码哈希失败: %v", err)
	}
	u := &model.User{Username: username, Password: hash, Nickname: username, Role: role, Status: "active", Balance: balance}
	if err := a.db.Create(u).Error; err != nil {
		a.t.Fatalf("自建测试账号 %s 失败: %v", username, err)
	}
	return u
}

// login 走真实登录接口获取 token。
func (a *testApp) login(username string) string {
	a.t.Helper()
	status, resp, raw := a.do("POST", "/api/v1/auth/login", "", map[string]any{"username": username, "password": testPassword})
	if status != 200 || resp.Code != 0 {
		a.t.Fatalf("登录接口异常: POST /api/v1/auth/login status=%d body=%s", status, raw)
	}
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil || data.Token == "" {
		a.t.Fatalf("登录接口未返回token: body=%s", raw)
	}
	return data.Token
}

// profileData 会员资料（会员页账户区的数据来源）。
type profileData struct {
	ID      uint    `json:"id"`
	Balance float64 `json:"balance"`
	Debt    float64 `json:"debt"`
	Role    string  `json:"role"`
}

// profile 读取当前登录用户资料（模拟会员页刷新账户区）。
func (a *testApp) profile(token string) profileData {
	a.t.Helper()
	status, resp, raw := a.do("GET", "/api/v1/auth/profile", token, nil)
	if status != 200 || resp.Code != 0 {
		a.t.Fatalf("会员资料接口异常: GET /api/v1/auth/profile status=%d body=%s", status, raw)
	}
	var p profileData
	if err := json.Unmarshal(resp.Data, &p); err != nil {
		a.t.Fatalf("会员资料接口响应解析失败: %v, body=%s", err, raw)
	}
	return p
}

// rechargeOK 通过接口为会员充值并断言成功。
func (a *testApp) rechargeOK(staffToken string, userID uint, amount float64) {
	a.t.Helper()
	status, resp, raw := a.do("POST", "/api/v1/recharges", staffToken, map[string]any{
		"user_id": userID, "amount": amount, "payment_method": "cash",
	})
	if status != 200 || resp.Code != 0 {
		a.t.Fatalf("会员充值接口异常: POST /api/v1/recharges status=%d body=%s", status, raw)
	}
}

// createPeripheral 通过接口自建外设设备，返回设备 ID。
func (a *testApp) createPeripheral(staffToken, deviceNo string) uint {
	a.t.Helper()
	status, resp, raw := a.do("POST", "/api/v1/peripherals", staffToken, map[string]any{
		"device_no": deviceNo, "device_type": "keyboard", "name": "测试键盘",
	})
	if status != 200 || resp.Code != 0 {
		a.t.Fatalf("设备登记接口异常: POST /api/v1/peripherals status=%d body=%s", status, raw)
	}
	var data struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil || data.ID == 0 {
		a.t.Fatalf("设备登记接口未返回设备ID: body=%s", raw)
	}
	return data.ID
}

// createRentalOK 通过接口登记租借并断言成功，返回租借记录 ID。
func (a *testApp) createRentalOK(staffToken string, userID, peripheralID uint, deposit float64) uint {
	a.t.Helper()
	status, resp, raw := a.do("POST", "/api/v1/rentals", staffToken, map[string]any{
		"user_id": userID, "peripheral_id": peripheralID, "deposit": deposit,
		"expected_return_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	if status != 200 || resp.Code != 0 {
		a.t.Fatalf("租借登记接口异常: POST /api/v1/rentals status=%d body=%s", status, raw)
	}
	var data struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil || data.ID == 0 {
		a.t.Fatalf("租借登记接口未返回记录ID: body=%s", raw)
	}
	return data.ID
}

// deviceStatus 查询设备当前状态（设备列表页的数据来源）。
func (a *testApp) deviceStatus(token string, id uint) string {
	a.t.Helper()
	status, resp, raw := a.do("GET", "/api/v1/peripherals?page=1&page_size=100", token, nil)
	if status != 200 || resp.Code != 0 {
		a.t.Fatalf("设备列表接口异常: GET /api/v1/peripherals status=%d body=%s", status, raw)
	}
	var data struct {
		List []struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"list"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		a.t.Fatalf("设备列表接口响应解析失败: %v, body=%s", err, raw)
	}
	for _, d := range data.List {
		if d.ID == id {
			return d.Status
		}
	}
	a.t.Fatalf("设备列表中未找到设备 id=%d: GET /api/v1/peripherals body=%s", id, raw)
	return ""
}

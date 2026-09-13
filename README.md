# 电竞馆上机管理系统（Esports Bar）

面向电竞馆/网咖门店的一站式运营工具：机位/包厢实时状态看板、会员充值与时长包、机位预约与续费、上机时长排行榜、电竞赛事报名与战队管理。

## 快速启动（Docker Compose，推荐）

```bash
# 在项目根目录执行
docker compose up -d --build
```

启动后访问：

| 入口 | 地址 |
| --- | --- |
| 前端页面 | http://localhost:28506 |
| 后端健康检查 | http://localhost:29506/healthz |
| 后端 API | http://localhost:29506/api/v1 |

演示账号（由后端启动时自动初始化）：

| 账号 | 密码 | 角色 |
| --- | --- | --- |
| admin | admin123456 | 管理员 |
| member | member123456 | 会员 |

## 主要功能

1. **机位/包厢实时状态看板**：网格/列表展示机位实时状态（空闲/使用中/故障/预约），按区域筛选，支持 WebSocket 实时推送（`/api/v1/ws/stations`）。
2. **会员充值与时长包**：会员充值余额、购买 10 小时/30 小时/月卡，消费时优先扣除时长包余额，不足时扣余额。
3. **机位预约与续费**：会员预约指定机位与时段，到店扫码开机，上机过程可续费延长时长。
4. **上机时长排行榜**：按日/周/月统计会员累计上机时长，支持按游戏类型（LOL/CSGO/王者荣耀）筛选。
5. **赛事报名与战队管理**：门店发布电竞赛事，玩家个人/战队报名，系统自动抽签分组，记录比赛结果与战绩。
6. **外设租借与损坏赔付**：店员为到场会员登记键盘/鼠标/耳机租借（设备编号、押金、预计归还时间），押金从会员余额冻结；会员可在自己的页面查看在借记录与欠款。归还时设备完好则结束租借并释放押金回余额；损坏则登记说明、赔偿金额与处理人，押金先抵扣赔偿，不足部分计入会员欠款。同一设备不能重复借出，已归还记录不能再次处理，赔偿完成不能重复扣款。

## 技术栈

| 层 | 技术栈 |
| --- | --- |
| 前端 | Vue 3 + TypeScript |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 缓存/限流 | Redis 7 |
| 实时通信 | gorilla/websocket |
| 认证 | JWT + RBAC |
| 日志 | log/slog |
| 参数校验 | go-playground/validator/v10 |
| 接口文档 | 见下方 API 清单（README 全量列出） |

## 项目目录结构

```
.
├── backend/
│   ├── cmd/server/main.go          # 入口：装配依赖、启动服务
│   ├── internal/
│   │   ├── config/                 # 配置加载
│   │   ├── database/               # MySQL/Redis 连接与种子数据
│   │   ├── model/                  # 每个实体一个文件
│   │   ├── dto/                    # 每个实体一个 DTO 文件
│   │   ├── repository/             # 每个实体一个 repository 文件
│   │   ├── service/                # 每个实体一个 service 文件
│   │   ├── handler/                # 每个实体一个 handler 文件
│   │   ├── router/                 # 每个实体一个路由注册文件
│   │   ├── middleware/             # auth/rbac/audit/request_id/error_handler/rate_limit/logger
│   │   ├── constants/              # 枚举、错误码、日志模板、文案
│   │   └── util/                   # jwt、logger、formatters、app_error 等
│   ├── pkg/response/               # 统一响应封装
│   ├── migrations/001_init.sql     # 参考建表脚本
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/api/                    # 每个实体一个 API 文件
│   ├── src/components/             # StatusBadge / EmptyState / DataTable / ConfirmDialog
│   ├── src/pages/                  # 每个模块一个页面
│   ├── src/stores/                 # auth / station / reservation / tournament
│   ├── src/hooks/                  # useAuth / usePagination
│   ├── src/utils/                  # request / format
│   ├── src/constants/              # 与后端对应的枚举
│   ├── Dockerfile
│   └── nginx.conf
├── database/init.sql               # MySQL 初始化脚本
├── docker-compose.yml
├── .env
├── .env.example
└── README.md
```

## 环境变量说明

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | Compose 项目名，决定容器名前缀 | `esportsbar` |
| `DB_NAME` | 数据库名 | `esportsbar_db` |
| `DB_USER` | 数据库用户 | `esportsbar_user` |
| `DB_PASSWORD` | 数据库密码 | `esportsbar_pwd` |
| `JWT_SECRET` | JWT 签名密钥（生产必须修改） | `change_me_to_a_long_random_string` |
| `JWT_EXPIRE_SEC` | JWT 有效期（秒） | `86400` |
| `FRONTEND_PORT` | 前端宿主机映射端口 | `28506` |
| `BACKEND_PORT` | 后端宿主机映射端口 | `29506` |
| `DB_PORT` | MySQL 宿主机映射端口 | `44005` |
| `REDIS_PORT` | Redis 宿主机映射端口 | `46305` |
| `LOG_LEVEL` | 后端日志级别（debug/info/warn/error） | `info` |

## API 清单（前缀 /api/v1，统一响应 { "code": 0, "message": "ok", "data": ... }）

### 认证

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | /auth/register | 会员注册 | 公开 |
| POST | /auth/login | 登录（返回 JWT） | 公开 |
| GET | /auth/profile | 当前用户信息 | 登录 |

### 用户

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | /users | 用户分页列表 | admin/staff |
| GET | /users/:id | 用户详情 | admin/staff |
| POST | /users | 创建用户 | admin |
| PUT | /users/:id | 更新用户 | admin |
| DELETE | /users/:id | 删除用户 | admin |

### 机位

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | /stations | 机位分页列表（区域/状态筛选） | 登录 |
| GET | /stations/all | 全部机位（看板） | 登录 |
| GET | /stations/:id | 机位详情 | 登录 |
| POST | /stations | 创建机位 | admin/staff |
| PUT | /stations/:id | 更新机位 | admin/staff |
| PUT | /stations/:id/status | 机位状态流转 | admin/staff |
| DELETE | /stations/:id | 删除机位 | admin |

### 充值与时长包

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| POST | /recharges | 会员充值 | admin/staff |
| GET | /recharges/mine | 我的充值记录 | 登录 |
| POST | /recharges/packages | 购买时长包 | 登录 |
| GET | /recharges/packages/orders | 我的时长包订单 | 登录 |
| GET | /packages | 时长包分页列表 | 登录 |
| GET | /packages/active | 在售时长包 | 登录 |
| GET | /packages/:id | 时长包详情 | 登录 |
| POST | /packages | 创建时长包 | admin/staff |
| PUT | /packages/:id | 更新时长包 | admin/staff |
| DELETE | /packages/:id | 删除时长包 | admin |

### 预约

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | /reservations | 预约分页列表 | 登录 |
| POST | /reservations | 创建预约 | 登录 |
| POST | /reservations/:id/confirm | 确认预约 | admin/staff |
| POST | /reservations/:id/cancel | 取消预约 | 登录 |
| POST | /reservations/:id/checkin | 到店开机 | admin/staff |

### 上机记录

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | /sessions | 上机记录分页列表 | 登录 |
| GET | /sessions/rank | 上机时长排行榜（day/week/month + game_type） | 登录 |
| POST | /sessions | 开机上机 | 登录 |
| POST | /sessions/:id/renew | 续费 | 登录 |
| POST | /sessions/:id/end | 下机结算 | 登录 |

### 赛事

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | /tournaments | 赛事分页列表 | 登录 |
| GET | /tournaments/:id | 赛事详情 | 登录 |
| POST | /tournaments | 创建赛事 | admin/staff |
| PUT | /tournaments/:id | 更新赛事 | admin/staff |
| DELETE | /tournaments/:id | 删除赛事 | admin |
| POST | /tournaments/:id/register | 赛事报名（solo/team） | 登录 |
| GET | /tournaments/:id/registrations | 报名列表 | 登录 |
| POST | /tournaments/:id/draw | 自动抽签分组 | admin/staff |
| GET | /tournaments/:id/matches | 比赛场次 | 登录 |
| GET | /teams/mine | 我的战队 | 登录 |
| POST | /teams | 创建战队 | 登录 |
| POST | /matches/:id/result | 提交比赛结果 | admin/staff |

### 外设租借与损坏赔付

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | /peripherals | 外设设备分页列表（类型/状态筛选；status=available 即可借设备下拉） | 登录 |
| POST | /peripherals | 登记外设设备（设备编号唯一） | admin/staff |
| PUT | /peripherals/:id/status | 设备状态流转（available↔maintenance） | admin/staff |
| GET | /rentals | 租借记录分页列表（状态/会员筛选） | admin/staff |
| GET | /rentals/mine | 我的租借记录（强制当前会员过滤） | 登录 |
| POST | /rentals | 登记租借（冻结押金、设备置已借出） | admin/staff |
| POST | /rentals/:id/return | 完好归还（释放押金回余额、设备恢复可借） | admin/staff |
| POST | /rentals/:id/damage | 损坏归还登记（说明/赔偿金额/处理人，押金不足部分计入会员欠款） | admin/staff |

### 审计与看板

| 方法 | 路径 | 说明 | 权限 |
| --- | --- | --- | --- |
| GET | /audits | 操作审计日志分页 | admin/staff |
| GET | /dashboard/summary | 看板汇总统计 | 登录 |
| GET | /ws/stations | 机位状态 WebSocket 实时推送 | 登录 |

> 复用关系标注：`/stations/all` 与看板页面复用 `StationService.ListAll`/`StationRepository.ListAll`；`/packages/active` 与购买页复用 `TimePackageService.ListActive`；`/sessions/rank` 与 `/sessions` 复用 `SessionService` 的仓储查询能力；`/reservations` 列表与 `DashboardService` 均复用 `StationRepository`；`/rentals/mine` 与 `/rentals` 复用 `PeripheralRentalRepository.List`（会员侧强制 user_id 过滤）；外设页“租借登记”的可借设备下拉复用 `GET /peripherals?status=available`（即 `PeripheralRepository.List`）。

## curl 调用示例

```bash
# 1. 健康检查
curl -sS http://localhost:29506/healthz

# 2. 登录获取 JWT
TOKEN=$(curl -sS -X POST http://localhost:29506/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123456"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['token'])")

# 3. 带 JWT 请求头访问列表
curl -sS http://localhost:29506/api/v1/stations?page=1\&page_size=5 -H "Authorization: Bearer $TOKEN"

# 4. 创建预约
curl -sS -X POST http://localhost:29506/api/v1/reservations \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"station_id":1,"start_time":"2026-08-17T10:00:00+08:00","end_time":"2026-08-17T12:00:00+08:00"}'

# 5. 购买时长包
curl -sS -X POST http://localhost:29506/api/v1/recharges/packages \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"package_id":1,"payment_method":"balance"}'

# 6. 上机时长排行榜
curl -sS "http://localhost:29506/api/v1/sessions/rank?period=week&limit=10" -H "Authorization: Bearer $TOKEN"

# 7. 外设租借闭环（店员账号）
# 7.1 查看可借设备
curl -sS "http://localhost:29506/api/v1/peripherals?status=available&page=1&page_size=10" -H "Authorization: Bearer $TOKEN"

# 7.2 为到场会员（user_id=2）登记租借：冻结押金 100，预计 3 天后归还
curl -sS -X POST http://localhost:29506/api/v1/rentals \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"user_id":2,"peripheral_id":1,"deposit":100,"expected_return_at":"2026-09-16T22:00:00+08:00","remark":"到店自取"}'

# 7.3 完好归还：结束租借并释放押金回会员余额
curl -sS -X POST http://localhost:29506/api/v1/rentals/1/return -H "Authorization: Bearer $TOKEN"

# 7.4 或损坏归还登记：赔偿 150，押金 100 抵扣后 50 计入会员欠款
curl -sS -X POST http://localhost:29506/api/v1/rentals/1/damage \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"damage_desc":"键盘进水，轴体损坏","compensation":150}'

# 8. 会员查看自己的租借记录与欠款（会员 token）
curl -sS "http://localhost:29506/api/v1/rentals/mine?page=1&page_size=10" -H "Authorization: Bearer $MEMBER_TOKEN"
```

## 本地开发方式

后端（Go 1.22+）：

```bash
cd backend
go mod tidy
go run ./cmd/server
# 需自行准备 MySQL（默认连接 127.0.0.1:44005）与 Redis（127.0.0.1:46305）
```

后端构建与测试：

```bash
cd backend
go build ./...
go vet ./...
go test ./...
```

前端（Node 18+）：

```bash
cd frontend
npm config set registry https://registry.npmmirror.com
npm install
npm run dev   # 开发服务器 http://localhost:28506，/api 代理到 29506
npm run build
```

## Docker 部署说明

- 端口映射：前端 `28506:80`、后端 `29506:8080`、MySQL `44005:3306`、Redis `46305:6379`。
- 数据卷：`db_data`（MySQL 数据）、`redis_data`（Redis 数据）为命名卷，删除容器不丢数据。
- 健康检查：db（mysqladmin ping）、backend（/healthz），backend 等待 db/redis healthy 后再启动。
- 常用命令：
  - 查看状态：`docker compose ps`
  - 查看日志：`docker compose logs -f backend`
  - 停止并清理：`docker compose down -v --remove-orphans`（-v 会同时删除数据卷，谨慎使用）
- 常见问题：
  - 端口冲突：修改 `.env` 中对应端口后重新 `docker compose up -d`。
  - 首次启动较慢：需拉取镜像并构建前后端。
  - 修改 JWT_SECRET 后所有旧 token 失效，需重新登录。

## 枚举出现位置清单

### 机位状态（idle / using / fault / reserved）

| 端 | 文件 |
| --- | --- |
| 后端 | `backend/internal/constants/enums.go`（定义）、`backend/internal/model/station.go`（模型默认值）、`backend/internal/dto/station_dto.go`（handler 校验 oneof）、`backend/internal/service/station_service.go`（状态机 allowedStationTransition）、`backend/internal/util/formatters.go`（StatusText）、`backend/internal/constants/error_codes.go`（CodeStationBusy/Fault）、`backend/internal/constants/log_templates.go`（station_status_change 模板）、`backend/internal/repository/station_repository.go`（筛选） |
| 前端 | `frontend/src/constants/index.ts`（STATION_STATUS/TEXT/TYPE）、`frontend/src/components/StatusBadge.vue`、`frontend/src/pages/Stations.vue`（筛选与徽标）、`frontend/src/pages/Dashboard.vue`（看板状态展示） |

### 预约状态（pending / confirmed / checked_in / completed / cancelled）

| 端 | 文件 |
| --- | --- |
| 后端 | `backend/internal/constants/enums.go`（定义）、`backend/internal/model/reservation.go`、`backend/internal/dto/reservation_dto.go`（oneof 校验）、`backend/internal/service/reservation_service.go`（状态机 Confirm/Cancel/CheckIn）、`backend/internal/util/formatters.go`（StatusText）、`backend/internal/constants/error_codes.go`（CodeReservation）、`backend/internal/constants/log_templates.go`（reservation_* 模板）、`backend/internal/repository/reservation_repository.go`（CountConflict 状态集合） |
| 前端 | `frontend/src/constants/index.ts`（RESERVATION_STATUS/TEXT/TYPE）、`frontend/src/components/StatusBadge.vue`、`frontend/src/pages/Reservations.vue`（筛选与操作按钮显隐） |

### 赛事状态（draft / open / ready / finished）

| 端 | 文件 |
| --- | --- |
| 后端 | `backend/internal/constants/enums.go`、`backend/internal/model/tournament.go`、`backend/internal/dto/tournament_dto.go`（oneof）、`backend/internal/service/tournament_service.go`（Register/DrawGroups 状态机）、`backend/internal/util/formatters.go`（StatusText）、`backend/internal/constants/error_codes.go`（CodeTournament）、`backend/internal/constants/log_templates.go`（tournament_* 模板） |
| 前端 | `frontend/src/constants/index.ts`（TOURNAMENT_STATUS/TEXT/TYPE）、`frontend/src/components/StatusBadge.vue`、`frontend/src/pages/Tournaments.vue`（按钮显隐） |

### 用户角色（admin / staff / member）

| 端 | 文件 |
| --- | --- |
| 后端 | `backend/internal/constants/roles.go`、`backend/internal/model/user.go`、`backend/internal/dto/user_dto.go`（oneof）、`backend/internal/middleware/rbac.go`（角色鉴权）、`backend/internal/router/*.go`（路由 RBAC 组合）、`backend/internal/util/formatters.go`（RoleText）、`backend/internal/constants/log_templates.go`（user_* 模板） |
| 前端 | `frontend/src/constants/index.ts`（USER_ROLE/TEXT）、`frontend/src/hooks/useAuth.ts`（isAdmin/isStaffOrAdmin）、`frontend/src/pages/Stations.vue`/`Reservations.vue`/`Recharge.vue`/`Tournaments.vue`（按钮显隐） |

### 游戏类型（lol / csgo / kog / other）与支付方式（balance / cash / wechat / alipay）

| 端 | 文件 |
| --- | --- |
| 后端 | `backend/internal/constants/enums.go`、`backend/internal/dto/session_dto.go`（oneof）、`backend/internal/dto/tournament_dto.go`（oneof）、`backend/internal/model/session.go`、`backend/internal/service/session_service.go`（defaultGameType）、`backend/internal/util/formatters.go`（GameTypeText） |
| 前端 | `frontend/src/constants/index.ts`（GAME_TYPE/TEXT、PAYMENT_METHOD/TEXT）、`frontend/src/pages/Sessions.vue`、`frontend/src/pages/Recharge.vue`、`frontend/src/pages/Tournaments.vue` |

### 外设类型（keyboard / mouse / headset）

| 端 | 文件 |
| --- | --- |
| 后端 | `backend/internal/constants/enums.go`（定义）、`backend/internal/model/peripheral.go`、`backend/internal/dto/peripheral_dto.go`（oneof 校验）、`backend/internal/util/formatters.go`（PeripheralTypeText）、`backend/internal/database/database.go`（种子设备） |
| 前端 | `frontend/src/constants/index.ts`（PERIPHERAL_TYPE/TEXT）、`frontend/src/pages/Peripherals.vue`（类型选择与展示） |

### 外设设备状态（available / rented / maintenance）

| 端 | 文件 |
| --- | --- |
| 后端 | `backend/internal/constants/enums.go`（定义）、`backend/internal/model/peripheral.go`（默认值）、`backend/internal/dto/peripheral_dto.go`（oneof 校验）、`backend/internal/service/peripheral_service.go`（状态机 allowedPeripheralTransition）、`backend/internal/service/peripheral_rental_service.go`（租借/归还联动流转）、`backend/internal/util/formatters.go`（StatusText）、`backend/internal/constants/error_codes.go`（CodePeripheralBusy）、`backend/internal/constants/log_templates.go`（peripheral_* 模板）、`backend/internal/repository/peripheral_repository.go`（筛选） |
| 前端 | `frontend/src/constants/index.ts`（PERIPHERAL_STATUS/TEXT/TYPE）、`frontend/src/components/StatusBadge.vue`、`frontend/src/pages/Peripherals.vue`（徽标与维护/恢复按钮显隐） |

### 外设租借状态（renting / returned / damaged）

| 端 | 文件 |
| --- | --- |
| 后端 | `backend/internal/constants/enums.go`（定义）、`backend/internal/model/peripheral_rental.go`（默认值）、`backend/internal/dto/peripheral_rental_dto.go`（oneof 校验）、`backend/internal/service/peripheral_rental_service.go`（Return/Damage 状态机与防重复处理）、`backend/internal/util/formatters.go`（StatusText）、`backend/internal/constants/error_codes.go`（CodeRentalState）、`backend/internal/constants/log_templates.go`（rental_* 模板）、`backend/internal/repository/peripheral_rental_repository.go`（筛选与在借统计） |
| 前端 | `frontend/src/constants/index.ts`（RENTAL_STATUS/TEXT/TYPE）、`frontend/src/components/StatusBadge.vue`、`frontend/src/pages/Peripherals.vue`（筛选与归还/损坏按钮显隐） |

## 设计说明

- 分层依赖严格单向：handler → service → repository → model，构造器注入，无反向引用。
- 多步写操作均放入 service 事务（`gorm.DB.Transaction`）；并发场景使用 `SELECT ... FOR UPDATE`（`repository/common.go` 的 `clauseLocking`），如余额扣减、机位状态流转、预约冲突校验、外设重复借出校验与租借记录防重复处理。
- 外设租借资金闭环：登记租借时押金从会员余额冻结（`users.balance`），完好归还全额释放回余额；损坏归还时押金抵扣赔偿，剩余退还、不足部分计入会员欠款（`users.debt`），赔付拆分逻辑见 `service/peripheral_rental_service.go` 的 `splitDamageCompensation`。
- 横切关注点：JWT + RBAC（`middleware/auth.go`、`middleware/rbac.go`、`util/jwt.go`）、操作审计（`middleware/audit.go` + `audit_logs` 表 + 审计页面）、全局错误处理与请求追踪（`middleware/request_id.go`、`middleware/error_handler.go`、`util/app_error.go`、`constants/error_codes.go`）。
- 共享组件：`StatusBadge`、`EmptyState`、`DataTable`、`ConfirmDialog`；共享 hooks/utils：`useAuth`、`usePagination`、`request.ts`、`format.ts`。
- 严禁合并职责到单一文件：每个实体按 model / dto / repository / service / handler / router / constants 拆分，前端按 api / stores / pages / components 拆分。
- 状态机跨多处定义（屎山耦合设计）：新增机位/预约/赛事状态需同步修改后端 constants、DTO 校验、service 状态机、formatters、错误码、日志模板与前端 constants、StatusBadge、页面按钮显隐等 ≥10 处文件。

## License

MIT

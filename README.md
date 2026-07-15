# 🏨 智慧酒店管理系统 (Go)

基于 **Go + Gin + GORM** 的多酒店 SaaS 预订平台，支持 JWT 认证、房间管理、日期查询空房、预订下单事务处理、在线支付、入住退房全流程。

![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)
![Gin](https://img.shields.io/badge/Gin-1.9-00ADD8?logo=go)
![GORM](https://img.shields.io/badge/GORM-1.31-00ADD8)
![License](https://img.shields.io/badge/license-MIT-green)

## 🎯 核心特性

| 模块 | 功能 |
|------|------|
| 🔐 **认证授权** | JWT 登录 + 4级角色（超管/酒店管理员/前台/住客）+ Redis 黑名单 |
| 🏨 **多租户** | 多酒店入驻，数据隔离，酒店管理员只能操作自家资源 |
| 🚪 **房间管理** | 房型 CRUD、房间批量创建、按日期查询空房、房态日历 |
| 📋 **预订下单** | **事务 + SELECT FOR UPDATE 行锁**并发控制，防超卖 |
| 💳 **支付系统** | 工厂模式抽象层（Mock/微信/支付宝），回调幂等处理 |
| 🔑 **安全防护** | AES-256-GCM 加密身份证号、bcrypt 密码哈希、身份证脱敏展示 |
| 📊 **数据分析** | 数据看板、Excel 报表导出、日/周/月营收统计 |
| 🖨️ **入住登记** | 打印入住单 HTML、入住/退房/续住状态机 |
| 🐳 **容器部署** | Docker 多阶段构建 + docker-compose (MySQL+Redis+App) |
| 💻 **零依赖运行** | SQLite 模式，无需安装 MySQL/Redis，一条命令启动 |

## 🚀 快速开始

### 方式一：纯本地运行（零依赖，推荐体验）

```bash
git clone https://github.com/Mapl1n/hotel-booking-go.git
cd hotel-booking-go
go run ./cmd/server
```

浏览器打开 `http://localhost:8080`，使用内置 Web 界面操作。

### 方式二：Docker Compose 一键部署

```bash
docker compose up -d
# 启动 MySQL + Redis + App 三个服务
# API: http://localhost:8080/api
```

## 🔑 默认账号

| 角色 | 账号 | 密码 |
|------|------|------|
| 平台超管 | `admin` | `123456` |
| 酒店管理员 | `13900001111` | `123456` |
| 前台员工 | `13900002222` | `123456` |
| 住客 | `13800008888` | `123456` |

## 📡 API 端点

### 公开接口

```
POST /api/auth/send-code     发送短信验证码
POST /api/auth/register      住客注册
POST /api/auth/login         用户登录
GET  /api/hotels             酒店列表
GET  /api/hotels/:id         酒店详情
GET  /api/rooms              按日期查空房
GET  /api/rooms/types        房型列表
GET  /api/rooms/calendar     房态日历
```

### 需认证接口 (JWT)

```
GET  /api/auth/me                   当前用户信息
POST /api/auth/logout               退出登录（加入黑名单）
POST /api/orders                    预订下单 ★
GET  /api/orders                    订单列表
GET  /api/orders/:id                订单详情（含身份证脱敏）
PUT  /api/orders/:id/cancel         取消订单
GET  /api/orders/:id/print          打印入住单
POST /api/payment/create            创建支付
GET  /api/payment/status/:id        支付状态查询
```

### 酒店员工专属

```
PUT  /api/orders/:id/check-in       办理入住
PUT  /api/orders/:id/check-out      办理退房
PUT  /api/orders/:id/extend         续住
GET  /api/admin/dashboard           数据看板
GET  /api/admin/export              导出Excel报表
```

## 🏗️ 项目架构

```
hotel-booking-go/
├── cmd/server/           # 应用入口 + 优雅关闭
├── internal/
│   ├── config/           # Viper 配置管理
│   ├── model/            # GORM 数据模型 (7张表)
│   ├── dao/              # 数据访问层 (事务+行锁)
│   ├── service/          # 业务逻辑层 ★
│   ├── handler/          # HTTP 处理层
│   ├── middleware/        # JWT/角色/限流/CORS
│   └── router/           # 路由注册 + 内嵌Web界面
├── pkg/
│   ├── crypto/           # AES-256-GCM 加密
│   ├── payment/          # 支付抽象层 (策略模式)
│   ├── sms/              # 短信验证码 (Redis/内存降级)
│   └── response/         # 统一响应格式
├── seed/                 # 种子数据
├── test/                 # 并发测试
├── Dockerfile            # 多阶段构建 (~15MB)
└── docker-compose.yml    # MySQL + Redis + App
```

## 🔒 并发控制（核心设计）

下单是并发最敏感的操作。系统使用 **悲观行锁** 保证数据一致性：

```go
// ★ 数据库事务 + SELECT FOR UPDATE 确保原子性
err := s.db.Transaction(func(tx *gorm.DB) error {
    // 1. FOR UPDATE 锁定目标房间 → 阻塞并发事务
    room, err := s.roomDAO.FindByIDForUpdate(tx, roomID)

    // 2. 行锁保护下检查时间段冲突
    conflict, _ := s.orderDAO.CountDateConflict(tx, roomID, checkIn, checkOut)

    // 3. 创建订单
    return tx.Create(order).Error
})
```

测试结果：**10 并发抢同一房间 → 1 成功 + 9 被拒** ✅

## 🛠️ 技术栈

- **Web 框架**: Gin v1.12
- **ORM**: GORM v1.31 (MySQL/SQLite)
- **认证**: golang-jwt v5 + bcrypt
- **缓存**: Redis v9 (可选)
- **日志**: Zap
- **Excel**: excelize v2
- **数据库**: MySQL 8.0 / SQLite (纯 Go)

## 📝 License

MIT License

---

🤖 Built with [Claude Code](https://claude.ai/code)

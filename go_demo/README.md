# Go Demo Admin Platform

基于 **Go + Gin** 的全栈管理平台后端，采用标准分层架构设计，支持 PostgreSQL / MySQL 双数据库、JWT 认证、RBAC 权限控制、WebSocket 实时聊天。

---

## 目录

- [技术栈](#技术栈)
- [项目结构](#项目结构)
- [快速开始](#快速开始)
- [环境配置](#环境配置)
- [分层架构说明](#分层架构说明)
- [API 接口文档](#api-接口文档)
  - [公开接口（无需认证）](#公开接口无需认证)
  - [认证接口（需 JWT Token）](#认证接口需-jwt-token)
  - [管理员接口（需 admin:access 权限）](#管理员接口需-adminaccess-权限)
  - [聊天接口（需 messages:chat 权限）](#聊天接口需-messageschat-权限)
- [数据模型](#数据模型)
- [安全机制](#安全机制)
- [部署指南](#部署指南)

---

## 技术栈

| 类别 | 技术 | 说明 |
|------|------|------|
| 语言 | Go 1.23+ | 静态类型，高性能并发 |
| Web 框架 | Gin v1.10 | 高性能 HTTP 路由框架 |
| 数据库 | PostgreSQL / MySQL | 通过 `DATABASE_TYPE` 切换 |
| Redis | go-redis v9 | 可选，会话缓存与发布订阅 |
| 认证 | JWT (golang-jwt/v5) | Access + Refresh 双令牌 |
| 加密 | AES-256-GCM + RSA | 密码传输加密 + 密钥对生成 |
| 密码哈希 | bcrypt | 用户密码单向哈希存储 |
| 配置管理 | godotenv | .env 文件加载，支持多环境 |

---

## 项目结构

```
go_demo/
├── main.go                    # 程序入口：加载配置 → 连接DB → 启动HTTP
├── go.mod / go.sum            # Go 模块定义
│
├── config/                    # ── 配置层 ──
│   └── config.go              #   环境变量加载、模式判断、配置验证
│
├── models/                    # ── 数据模型层 ──
│   └── models.go              #   User / Role / Permission / ChatUser / ChatMessage / TokenPair / Claims
│
├── utils/                     # ── 工具层 ──
│   └── crypto.go              #   AES-256-GCM 密码加解密 (EncryptPassword / DecryptPassword)
│
├── database/                  # ── 数据访问层 ──
│   └── database.go            #   连接池管理、自动迁移、方言适配(PG/MySQL)、通用查询封装
│
├── services/                  # ── 业务逻辑层（核心） ──
│   ├── auth_service.go        #   注册 / 登录 / JWT签发验证 / 密码重置 / RSA密钥管理
│   └── rbac_service.go        #   用户CRUD / 角色CRUD / 权限检查 / 角色分配 / 密码查看
│
├── controllers/               # ── 控制器层 ──
│   ├── auth_controller.go     #   认证相关 HTTP 处理
│   ├── admin_controller.go    #   管理 API（用户/角色/权限）
│   └── chat_controller.go     #   聊天引擎：WebSocket / 消息 / 上传 / 翻译 / 在线状态
│
├── middlewares/               # ── 中间件层 ──
│   └── auth.go                #   AuthMiddleware (JWT认证) + RequirePermission (RBAC鉴权)
│
├── routes/                    # ── 路由层 ──
│   └── routes.go              #   所有路由定义 + 中间件链组装
│
├── .env.example               #   配置模板（复制后修改）
├── .env.development           #   开发环境配置
├── .env.production            #   生产环境配置
└── .gitignore                 #   Git 忽略规则
```

---

## 快速开始

### 前置条件

- Go 1.23+
- PostgreSQL 或 MySQL
- Redis（可选）

### 安装运行

```bash
# 1. 克隆或进入项目目录
cd go_demo

# 2. 安装依赖
go mod download

# 3. 配置环境变量（见下方「环境配置」）
cp .env.development .env
# 编辑 .env 填入你的数据库密码等实际值

# 4. 运行
go run .

# 或者编译后执行
go build -o server.exe .
./server.exe
```

启动成功后会输出：

```
[development] server running at http://localhost:8080
Mode          : development
Port          : 8080
DB Type       : postgres
PG Host       : localhost
...
```

访问 `http://localhost:8080/api/v1/ping` 验证服务是否正常：
```json
{"message":"pong"}
```

---

## 环境配置

通过 `.env` 文件管理所有配置，系统根据 `APP_ENV` 自动选择对应文件：

| 环境变量 | 必填 | 默认值 | 说明 |
|----------|------|--------|------|
| `APP_ENV` | 否 | `development` | 运行环境：`development` / `production` |
| `PORT` | 否 | `8080` | HTTP 监听端口 |
| `GIN_MODE` | 否 | `debug` | Gin 框架模式：`debug`(开发) / `release`(生产) |
| **数据库** ||||
| `DATABASE_TYPE` | 是 | — | 数据库类型：`postgres` 或 `mysql` |
| `DATABASE_DSN` | 否 | — | 完整 DSN 连接串（优先级最高） |
| `PG_HOST` | PG时必填 | `localhost` | PostgreSQL 主机 |
| `PG_PORT` | 否 | `5432` | PostgreSQL 端口 |
| `PG_USER` | PG时必填 | `postgres` | PostgreSQL 用户名 |
| `PG_PASSWORD` | PG时必填 | — | PostgreSQL 密码 |
| `PG_DB_NAME` | PG时必填 | — | PostgreSQL 数据库名 |
| `MYSQL_HOST` | MySQL时必填 | `localhost` | MySQL 主机 |
| `MYSQL_PORT` | 否 | `3306` | MySQL 端口 |
| `MYSQL_USER` | MySQL时必填 | `root` | MySQL 用户名 |
| `MYSQL_PASSWORD` | MySQL时必填 | — | MySQL 密码 |
| `MYSQL_DB_NAME` | MySQL时必填 | — | MySQL 数据库名 |
| `DB_MAX_OPEN_CONN` | 否 | `25` | 最大打开连接数 |
| `DB_MAX_IDLE_CONN` | 否 | `5` | 最大空闲连接数 |
| **Redis（可选）** ||||
| `REDIS_HOST` | 否 | `localhost` | Redis 主机 |
| `REDIS_PORT` | 否 | `6379` | Redis 端口 |
| `REDIS_PASSWORD` | 否 | — | Redis 密码 |
| **默认管理员** ||||
| `DEFAULT_ADMIN_USERNAME` | 否 | `admin` | 首次启动自动创建的管理员用户名 |
| `DEFAULT_ADMIN_EMAIL` | 是* | — | 管理员邮箱（唯一标识符） |
| `DEFAULT_ADMIN_PASSWORD` | 否 | `admin123` | 管理员初始密码 |
| **安全** ||||
| `JWT_SECRET` | 生产必填 | — | JWT 签名密钥（生产环境必须 >= 32 字符随机串） |

> *首次启动时会根据这些配置自动创建默认管理员账号和初始角色权限数据

---

## 分层架构说明

```
routes → controllers → services → database → models
         ↑                ↑           ↑
    middlewares        config      utils
```

| 层级 | 目录 | 职责 | 依赖方向 |
|------|------|------|----------|
| 入口层 | `main.go` | 启动编排：加载配置 → 初始化DB → 注册路由 → 启动HTTP | 调用 config/database/routes |
| 配置层 | `config/` | 环境变量加载、模式判断、生产配置校验 | 无外部依赖 |
| 模型层 | `models/` | 纯数据结构定义（User/Role/Permission/...） | 仅依赖标准库 |
| 工具层 | `utils/` | 无状态纯函数（AES加解密） | 无外部依赖 |
| 数据访问层 | `database/` | DB连接池、自动迁移、SQL方言适配、通用查询封装 | 依赖 config/utils |
| 业务逻辑层 | `services/` | 核心业务：认证/RBAC/编排业务流程 | 依赖 database/models/config/utils |
| 控制器层 | `controllers/` | HTTP协议翻译：解析请求→调用Service→返回响应 | 依赖 services/models |
| 中间件层 | `middlewares/` | JWT认证、RBAC鉴权等横切关注点 | 依赖 services |
| 路由层 | `routes/` | URL映射表、中间件链组装 | 依赖 controllers/middlewares/services |

**核心原则**：严格单向依赖，上层可调下层，下层不可反向依赖上层。

---

## API 接口文档

基础路径：`http://localhost:8080/api/v1`

> 所有需要认证的接口需在 Header 中携带：`Authorization: Bearer <access_token>`

---

### 公开接口（无需认证）

#### 健康检查

```http
GET /health
GET /api/v1/ping
GET /
```

#### 获取 RSA 公钥（用于密码加密传输）

```http
GET /api/v1/password-public-key
```

**响应：**
```json
{
  "public_key": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A...\n-----END PUBLIC KEY-----"
}
```

#### 注册

```http
POST /api/v1/register
Content-Type: application/json

{
  "username": "zhangsan",
  "email": "zhangsan@example.com",
  "phone": "13800138000",
  "encrypted_password": "<RSA加密后的base64字符串>"
}
```

**响应：**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
  "access_token_expires_in": 900,
  "refresh_token_expires_in": 604800,
  "token_type": "Bearer"
}
```

> `encrypted_password` 为使用公钥 RSA-OAEP 加密后的 base64 字符串

#### 登录

```http
POST /api/v1/login
Content-Type: application/json

{
  "email": "admin@example.com",
  "encrypted_password": "<RSA加密后的密码>"
}
```

**响应：** 同注册（返回 TokenPair）

#### 刷新 Token

```http
POST /api/v1/refresh-token
Content-Type: application/json

{
  "refresh_token": "<有效的 refresh_token>"
}
```

**响应：** 新的 TokenPair

#### 忘记密码（发送重置邮件）

```http
POST /api/v1/forgot-password
Content-Type: application/json

{
  "email": "user@example.com"
}
```

#### 重置密码

```http
POST /api/v1/reset-password
Content-Type: application/json

{
  "token": "<邮箱收到的重置Token>",
  "new_encrypted_password": "<新密码的RSA加密>"
}
```

#### WebSocket 聊天连接

```
WS /api/v1/chat/ws?token=<access_token>
```

建立 WebSocket 连接后可进行实时消息收发。消息格式：

```json
// 客户端 → 服务端
{"type": "message", "to_user_id": 2, "content": "你好"}

// 服务端 → 客户端
{"type": "message", "from_user_id": 1, "content": "你好", "created_at": "..."}
```

支持的消息类型：
- `message` — 文本消息
- `typing` — 正在输入状态
- `read` — 已读回执

---

### 认证接口（需 JWT Token）

#### 获取当前用户信息

```http
GET /api/v1/me
Authorization: Bearer <access_token>
```

**响应：**
```json
{
  "id": 1,
  "username": "admin",
  "email": "admin@example.com",
  "phone": "13800138000",
  "roles": ["admin"],
  "permissions": ["admin:access", "users:read", "users:write", ...],
  "created_at": "2026-01-01T00:00:00Z"
}
```

---

### 管理员接口（需 `admin:access` 权限）

> 以下接口除基本认证外，还需具备对应的具体权限

#### 用户管理

| 方法 | 路径 | 所需权限 | 说明 |
|------|------|----------|------|
| `GET` | `/api/v1/admin/users` | `users:read` | 分页获取用户列表 |
| `POST` | `/api/v1/admin/users` | `users:write` | 创建用户 |
| `PUT` | `/api/v1/admin/users/:id` | `users:write` | 更新用户信息 |
| `PUT` | `/api/v1/admin/users/:id/roles` | `users:write` | 设置用户角色 |
| `GET` | `/api/v1/admin/users/:id/password` | `users:password:read` | 查看用户原始密码（AES解密） |
| `PUT` | `/api/v1/admin/users/:id/password` | `users:write` | 重置用户密码 |
| `DELETE` | `/api/v1/admin/users/:id` | `users:write` | 停用用户（软删除） |

**创建用户示例：**
```http
POST /api/v1/admin/users
Content-Type: application/json

{
  "username": "lisi",
  "email": "lisi@example.com",
  "phone": "13900139000",
  "roles": ["editor"]
}
```

**更新用户示例：**
```http
PUT /api/v1/admin/users/2
Content-Type: application/json

{
  "username": "lisi_updated",
  "email": "lisi_new@example.com",
  "phone": "13911111111"
}
```

**设置用户角色示例：**
```http
PUT /api/v1/admin/users/2/roles
Content-Type: application/json

{
  "role_ids": [1, 3]
}
```

**重置密码示例：**
```http
PUT /api/v1/admin/users/2/password
Content-Type: application/json

{
  "password": "newpassword123"
}
```

#### 角色管理

| 方法 | 路径 | 所需权限 | 说明 |
|------|------|----------|------|
| `GET` | `/api/v1/admin/roles` | `roles:read` | 获取角色列表 |
| `POST` | `/api/v1/admin/roles` | `roles:write` | 创建角色 |
| `PUT` | `/api/v1/admin/roles/:id` | `roles:write` | 更新角色（名称+描述+权限） |
| `DELETE` | `/api/v1/admin/roles/:id` | `roles:write` | 删除角色（系统保留角色不可删） |

**创建角色示例：**
```http
POST /api/v1/admin/roles
Content-Type: application/json

{
  "name": "editor",
  "description": "内容编辑",
  "permission_codes": ["messages:chat", "users:read"]
}
```

#### 权限列表

```http
GET /api/v1/admin/permissions
Authorization: Bearer <access_token>
```

**响应：**
```json
[
  {"id": 1, "code": "admin:access", "description": "管理员后台访问"},
  {"id": 2, "code": "users:read", "description": "查看用户"},
  {"id": 3, "code": "users:write", "description": "编辑用户"},
  ...
]
```

---

### 聊天接口（需 `messages:chat` 权限）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/chat/users` | 获取在线用户列表 |
| `GET` | `/api/v1/chat/history/:id` | 获取与指定用户的聊天历史 |
| `POST` | `/api/v1/chat/upload` | 上传聊天文件（图片/视频/音频/文档） |
| `POST` | `/api/v1/chat/translate` | AI 翻译消息内容 |
| `WS` | `/api/v1/chat/ws` | WebSocket 实时聊天通道 |

**文件上传示例：**
```http
POST /api/v1/chat/upload
Content-Type: multipart/form-data

file=<文件>&to_user_id=2&message_type=image
```

**翻译请求示例：**
```http
POST /api/v1/chat/translate
Content-Type: application/json

{
  "text": "Hello, how are you?",
  "target_lang": "zh-CN"
}
```

---

## 数据模型

### User（用户）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 主键 |
| username | string | 用户名 |
| email | string | 邮箱（唯一） |
| phone | string | 手机号 |
| roles | []string | 角色名列表（虚拟字段，关联查询） |
| permissions | []string | 权限码列表（虚拟字段，关联查询） |
| created_at | time.Time | 创建时间 |
| deleted_at | *time.Time | 软删除时间（NULL=未删除） |

### Role（角色）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 主键 |
| name | string | 角色名（如 admin/editor/viewer） |
| description | string | 角色描述 |
| permissions | []string | 关联的权限码列表（虚拟字段） |
| created_at | time.Time | 创建时间 |

### Permission（权限）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 主键 |
| code | string | 权限码（如 users:read） |
| description | string | 权限描述 |
| created_at | time.Time | 创建时间 |

### ChatMessage（聊天消息）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 主键 |
| from_user_id | int64 | 发送者ID |
| to_user_id | int64 | 接收者ID |
| message_type | string | 消息类型：text/image/video/audio/file |
| content | string | 文本内容 / 描述 |
| media_url | string | 媒体文件URL |
| file_name | string | 原始文件名 |
| mime_type | string | MIME类型 |
| file_size | int64 | 文件大小（字节） |
| transcript | string | 音频转写文本 |
| translation | string | AI翻译结果 |
| created_at | time.Time | 发送时间 |

### TokenPair（JWT令牌对）
| 字段 | 类型 | 说明 |
|------|------|------|
| access_token | string | 访问令牌（有效期15分钟） |
| refresh_token | string | 刷新令牌（有效期7天） |
| access_token_expires_in | int64 | access_token 过期秒数 |
| refresh_token_expires_in | int64 | refresh_token 过期秒数 |
| token_type | string | 固定值 "Bearer" |

---

## 安全机制

### 1. 密码传输加密（RSA + AES）
- 前端获取服务端 RSA 公钥
- 使用 **RSA-OAEP** 加密用户密码后传输
- 服务端用私钥解密，再用 **AES-256-GCM** 加密后存入数据库
- 密码明文**永不**在网络和数据库中以明文出现

### 2. JWT 双令牌机制
- **Access Token**：短期有效（15分钟），用于 API 认证
- **Refresh Token**：长期有效（7天），仅用于刷新 Access Token
- Token 泄露影响窗口小，Refresh Token 可主动撤销

### 3. RBAC 细粒度权限控制
- 基于角色的访问控制（Role-Based Access Control）
- 内置权限体系：

| 权限码 | 说明 |
|--------|------|
| `admin:access` | 管理后台访问 |
| `users:read` | 查看用户列表 |
| `users:write` | 创建/编辑/停用用户 |
| `users:password:read` | 查看用户原始密码 |
| `roles:read` | 查看角色列表 |
| `roles:write` | 创建/编辑/删除角色 |
| `permissions:read` | 查看权限列表 |
| `messages:chat` | 聊天功能使用 |

### 4. 数据安全
- 密码使用 **bcrypt** 单向哈希存储
- 敏感数据（数据库中的密码）使用 **AES-256-GCM** 对称加密
- 支持 PostgreSQL 和 MySQL 双数据库方言
- 用户删除采用**软删除**（`deleted_at`），数据可恢复

---

## 部署指南

### 开发环境
```bash
cp .env.development .env
# 编辑 .env 填入本地数据库信息
go run .
```

### 生产环境
```bash
# 1. 准备生产配置
cp .env.production .env
# 编辑 .env，务必修改以下项:
#   - PG_PASSWORD / MYSQL_PASSWORD（真实密码）
#   - JWT_SECRET（>=32字符随机串）
#   - DEFAULT_ADMIN_PASSWORD（强密码）
#   - GIN_MODE=release
#   - APP_ENV=production

# 2. 编译
CGO_ENABLED=0 GOOS=linux go build -o server .

# 3. 运行
./server
```

### Docker 部署（推荐）
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder/app/server .
COPY --from=builder/app/.env.production .env
EXPOSE 8080
CMD ["./server"]
```

```bash
docker build -t go-demo .
docker run -d -p 8080:8080 \
  -e PG_PASSWORD=your_password \
  -e JWT_SECRET=your-random-secret-key \
  go-demo
```

---

## License

MIT

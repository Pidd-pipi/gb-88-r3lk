# MockHub · 在线 API Mock 服务

MockHub 是一个在线 API Mock 管理平台。通过配置 Mock 规则即可模拟 API 响应，支持 gofakeit 动态响应、条件响应、响应延迟、请求日志与 OpenAPI 文档导入，帮助前后端团队并行开发。

## 快速启动（Docker Compose 一键部署）

```bash
cp .env.example .env
# 编辑 .env，将 JWT_SECRET 替换为至少 32 位的随机字符串
docker compose up -d --build
```

- 前端：http://localhost:8119
- 后端 API：http://localhost:3119
- API 文档：http://localhost:3119/docs
- 存活检查：http://localhost:3119/healthz
- 就绪检查：http://localhost:3119/readyz

测试账号：`dev / dev123`（普通用户）、`lead / lead123`（管理员）

## 核心功能

- Mock 项目管理：创建项目自动生成 Base URL（`/mock/{projectId}`）
- 接口配置：路径 / 方法 / 状态码 / 响应体（JSON）/ 响应头 / 延迟
- 接口版本：每次编辑生成不可变新版本（记录操作人与时间），列表展示当前版本，历史版本可预览、可一键发布回滚
- 并发安全：编辑/发布基于 `current_version` 乐观锁（CAS），并发发布同一接口只有一个成功，失败请求不改动当前版本
- 版本可观测：Mock 响应携带 `X-Mock-API-Version` 响应头，请求日志持久化实际生效版本，页面刷新后可回读
- 平滑升级：版本功能上线前的旧接口（无版本数据）可正常编辑，首次编辑自动把旧内容回填为 v1
- 动态响应：`{{name.fullName}}`、`{{internet.email}}`、`{{repeat 5|...}}` 等 gofakeit 模板
- 条件响应：按 query/body 字段匹配返回不同响应（如 `role=admin`）
- 请求日志：记录方法、路径、Headers、Body、Query、响应状态、响应体与生效版本
- Swagger/OpenAPI 导入：上传 OpenAPI 2.0/3.0 JSON 自动生成接口
- 鉴权：JWT + RBAC（lead 管理员 / dev 普通用户）
- 可靠性：登录/注册与 Mock 接口分路由限流、结构化访问日志、优雅关闭、MySQL 连接池与启动重试

## 技术架构

```
handler → service → repository → model (GORM)
     ↓
  middleware（JWT 鉴权 / 限流 / 请求日志 / panic 恢复）
```

后端采用 Go 1.22 + Gin + GORM 分层架构，数据库为 MySQL 8.0。开发/测试环境使用 GORM AutoMigrate，生产环境建议改用 `database/init.sql` 或独立 migration 工具（如 golang-migrate）管理表结构。

## 本地开发

### 后端

```bash
cd backend
go mod tidy
go run ./cmd/server
```

构建检查：`go build ./...`；测试：`go test ./...`

### 前端（Vue 3 + TypeScript + Arco Design + Vite）

```bash
cd frontend
npm install
npm run dev
```

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 认证 | JWT（golang-jwt/v5）+ bcrypt + RBAC |
| Mock 引擎 | brianvoe/gofakeit（动态/条件响应） |
| 前端 | Vue 3 + TypeScript + Arco Design + Vite + Monaco Editor |
| 部署 | Docker Compose（Nginx 反代 + 多阶段构建） |

## 目录结构

```
├── backend/                 # Go 后端（cmd + internal 分层）
│   ├── cmd/server/main.go
│   ├── database/init.sql
│   └── internal/
│       ├── config/          # 环境变量配置与校验
│       ├── model/           # GORM 模型
│       ├── repository/      # 数据访问层
│       ├── service/         # 业务逻辑
│       ├── handler/         # HTTP 处理
│       ├── router/          # 路由注册
│       ├── middleware/      # auth/error_handler/request_logger/rate_limiter
│       ├── dto/             # 请求/响应结构体（validator 校验）
│       ├── constants/       # 错误码与消息
│       ├── util/            # jwt/response/faker 模板渲染
│       ├── logger/          # log/slog 结构化日志
│       └── docs/            # 静态 API 文档
├── frontend/                # Vue 3 前端
├── database/init.sql        # 建库脚本（表结构由 GORM AutoMigrate 完成）
├── docker-compose.yml
└── .env.example
```

## 环境变量

| 变量 | 说明 | 默认 |
| --- | --- | --- |
| COMPOSE_PROJECT_NAME | Compose 项目名 | mockhub |
| FRONTEND_PORT | 前端宿主端口 | 8119 |
| BACKEND_PORT | 后端宿主端口 | 3119 |
| DB_PORT | MySQL 宿主端口 | 33388 |
| DB_NAME / DB_USER / DB_PASSWORD / DB_ROOT_PASSWORD | 数据库配置 | mockhub / mockhub / mockhub123 / root_pwd |
| JWT_SECRET | JWT 签名密钥，生产环境必须替换为 32 位以上随机值 | 见 .env.example |
| JWT_EXPIRE_HOURS | token 有效期（小时） | 72 |
| APP_ENV | 运行环境（development/production） | development |
| SERVER_PORT | 后端内部监听端口 | 8080 |
| BASE_URL | 对外后端地址（生成 Mock URL 用） | http://localhost:3119 |
| CORS_ALLOWED_ORIGINS | 允许跨域来源，逗号分隔；生产禁止 `*` | http://localhost:8119 |
| AUTH_RATE_LIMIT | 登录/注册限流（次/分钟/IP） | 10 |
| MOCK_RATE_LIMIT | Mock 接口限流（次/分钟/IP） | 120 |
| DB_MAX_OPEN_CONNS | 数据库最大连接数 | 25 |
| DB_MAX_IDLE_CONNS | 数据库最大空闲连接数 | 5 |
| DB_CONN_MAX_LIFETIME_MIN | 连接最大存活时间（分钟） | 5 |
| DB_CONNECT_RETRIES | 启动时数据库连接重试次数 | 10 |
| DB_CONNECT_RETRY_INTERVAL_SEC | 数据库重试间隔（秒） | 3 |

## Docker 部署说明

- 端口映射：前端 `${FRONTEND_PORT}:80`、后端 `${BACKEND_PORT}:8080`、MySQL `${DB_PORT}:3306`
- 前端 Nginx 将 `/api/`、`/mock/`、`/docs/` 反向代理到后端
- 后端通过 `readyz` 作为容器健康检查，只有数据库就绪后才进入 healthy
- 数据库使用命名卷 `db_data` 持久化
- 支持中文目录名部署（`name: mockhub` 不依赖目录名）

## API 清单（核心）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /api/v1/auth/register | 注册 |
| POST | /api/v1/auth/login | 登录（返回 JWT） |
| GET | /api/v1/auth/me | 当前用户 |
| GET/POST | /api/v1/projects | 项目列表 / 创建 |
| GET/PUT/DELETE | /api/v1/projects/:id | 项目详情 / 更新 / 删除 |
| GET/POST | /api/v1/projects/:projectId/apis | 接口列表（含当前版本）/ 新建（生成 v1） |
| GET/PUT/DELETE | /api/v1/projects/:projectId/apis/:id | 接口详情 / 更新（生成新版本，可带 `baseVersion` 乐观锁）/ 删除 |
| GET | /api/v1/projects/:projectId/apis/:id/versions | 接口历史版本列表（含操作人与时间） |
| POST | /api/v1/projects/:projectId/apis/:id/versions/:version/publish | 发布指定版本（回滚），需携带 `expectedVersion` |
| POST | /api/v1/projects/:projectId/swagger/import | 导入 OpenAPI |
| GET/DELETE | /api/v1/projects/:projectId/logs | 请求日志 / 清空 |
| ANY | /mock/:projectId/*path | 公开 Mock 引擎 |
| GET | /healthz、/readyz | 健康检查 |

统一响应格式：`{ "code": 0, "message": "ok", "data": ... }`

## License

MIT

# 01 后端范围与工程基线计划

## 依据文档

- `docs/rewrite-analysis/00-project-overview.md`
- `docs/rewrite-analysis/07-config-and-dependencies.md`
- `docs/rewrite-analysis/08-deployment.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

建设 Go 后端，统一提供 HTTP API、WebSocket、FTP server lifecycle、本地文件存储、PostgreSQL 持久化和 Docker Compose 部署能力。

## 范围

- Go 后端同时承载 REST API、WebSocket 和 FTP 服务。
- PostgreSQL 是账号、配置、媒体资产、传输事件和系统元数据的唯一持久化数据库。
- 文件存储只使用服务器本地数据卷。
- Docker Compose 是正式部署方式。
- 不做 Android、Windows、Tauri、AI、FTPS、RBAC、多用户、对象存储。
- 按 Lazycat 项目基线使用 Go 后端；注册、双 token、无感刷新属于通用默认要求，本项目因已确认唯一默认系统账号而豁免。

## 建议目录

```txt
backend/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── router.go
│   │   ├── middleware/
│   │   └── handlers/
│   ├── auth/
│   ├── config/
│   ├── db/
│   │   ├── migrations/
│   │   └── queries/
│   ├── ftpserver/
│   ├── media/
│   ├── realtime/
│   ├── storage/
│   └── system/
├── api/
│   └── openapi.yaml
├── deployments/
│   ├── Dockerfile
│   └── docker-compose.yml
├── .env.example
└── go.mod
```

## 模块职责

| 模块 | 职责 | 不负责 |
|---|---|---|
| `internal/api` | router、middleware、handler、request id、错误响应 | 业务规则落地 |
| `internal/auth` | 系统账号登录、会话、密码 hash | 多用户、注册、RBAC |
| `internal/db` | PostgreSQL 连接、migration、repository/query | 本地文件读写 |
| `internal/config` | 环境变量、app settings 读取和校验 | 前端展示文案 |
| `internal/ftpserver` | FTP lifecycle、认证、PASV、上传 hook、运行统计 | 媒体识别和 EXIF |
| `internal/storage` | 本地数据卷、临时文件、路径安全、删除 | 对象存储抽象 |
| `internal/media` | hash、MIME、照片识别、EXIF、媒体入库、照片 API | FTP 协议处理 |
| `internal/realtime` | WebSocket hub、事件类型、连接生命周期 | 持久消息队列 |
| `internal/system` | 系统信息聚合、存储状态、版本 hash | 审计日志 |

## 技术选型待定项

| 项目 | 推荐 | 决策要求 |
|---|---|---|
| HTTP 框架 | Gin 或 Echo | API 面小，优先团队熟悉度和中间件生态 |
| DB 访问 | sqlc 或 pgx 手写 repository | migration 和 SQL 可审计 |
| migration | golang-migrate 或 goose | Docker 启动可自动执行或显式执行 |
| 日志 | slog 或 zap | 输出 stdout，便于 Docker 收集 |
| FTP 库 | 需验证 | 必须支持 PASV 端口、自定义认证、上传完成 hook、根目录限制 |
| 密码哈希 | Argon2id 或 bcrypt | Web 账号和 FTP 账号都不能明文保存 |

## 选型 Spike 输出

实施前需要形成一份轻量结论，可放在 PR 描述或 `backend/docs/decisions/`：

| Spike | 必须回答 |
|---|---|
| HTTP 框架 | middleware、WebSocket 升级、测试写法是否顺手 |
| FTP 库 | PASV 端口、public host、匿名/账号认证、上传完成 hook、root jail、优雅停止 |
| migration | 本地命令和容器启动时如何执行，失败是否阻止 app 启动 |
| DB 查询层 | SQL 可审计、事务边界清晰、测试可控 |
| 密码 hash | PHC/成本参数、verify 失败路径、明文清理策略 |

## 环境变量

```txt
DATABASE_URL=
STORAGE_ROOT=/data/falcondrop/uploads
TMP_ROOT=/data/falcondrop/tmp
HTTP_ADDR=:8080
FTP_HOST=0.0.0.0
FTP_PUBLIC_HOST=
FTP_PORT=2121
FTP_PASSIVE_PORTS=30000-30009
SESSION_SECRET=
COOKIE_SECURE=false
DEFAULT_SYSTEM_USERNAME=admin
DEFAULT_SYSTEM_PASSWORD=
DEFAULT_FTP_USERNAME=camera
DEFAULT_FTP_PASSWORD=
DEFAULT_FTP_ANONYMOUS_ENABLED=true
```

## 配置加载规则

- 环境变量负责启动必需配置和初始化默认值。
- `app_settings` 负责运行期可展示或可调整配置，例如 `ftp.port`、`ftp.passive_ports`、`ftp.public_host`、`storage.root`。
- `DATABASE_URL`、`SESSION_SECRET`、`STORAGE_ROOT`、`TMP_ROOT` 缺失或不可用时启动失败。
- 默认系统账号密码缺失时启动失败，避免生成不可追踪弱密码。
- 默认 FTP 密码缺失且 `DEFAULT_FTP_ANONYMOUS_ENABLED=false` 时启动失败。
- `.env.example` 只能给示例值，不能提交真实 secret。

## HTTP 基础契约

### 健康检查

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| `GET` | `/healthz` | 否 | 进程存活，不要求数据库完整可用 |
| `GET` | `/readyz` | 否 | 数据库、migration、存储目录可写都通过才返回成功 |

### 错误响应

```json
{
  "code": "STORAGE_NOT_WRITABLE",
  "message": "存储目录不可写",
  "requestId": "req_..."
}
```

- 所有业务错误必须有稳定 `code` 和中文 `message`。
- middleware 负责注入 `requestId`，日志也必须带同一个 id。
- handler 不返回服务器真实路径、SQL 错误、panic 栈给前端。

## 启动流程

1. 解析环境变量并校验必需项。
2. 初始化 logger 和 request id 生成器。
3. 连接 PostgreSQL。
4. 执行 migration 或校验 migration 已完成。
5. seed 默认系统账号、FTP 账号和基础 settings。
6. 创建 `STORAGE_ROOT` 和 `TMP_ROOT`，检查可写。
7. 初始化 repository、service、WebSocket hub 和 FTP manager。
8. 注册 HTTP router。
9. 启动 HTTP server，监听 shutdown signal。
10. 优雅关闭 HTTP、WebSocket、FTP server 和数据库连接。

## 实施步骤

1. 初始化 `go.mod` 和基础 `cmd/api/main.go`。
2. 建立配置加载、日志、数据库连接、优雅关闭。
3. 建立 API router 和统一错误响应。
4. 建立 migration 目录和启动检查。
5. 建立本地存储根目录、临时目录和可写性检查。
6. 建立 Dockerfile、docker-compose 和 `.env.example`。
7. 建立基础 health endpoint，用于容器 smoke test。
8. 编写 README 或部署片段，说明 `FTP_PUBLIC_HOST` 和 PASV 端口在 Docker/NAT 下必须正确配置。

## 验收标准

- `go test ./...` 通过。
- Docker Compose 可以启动 app 和 PostgreSQL。
- 服务启动时能连接数据库并确认存储目录可写。
- `/healthz` 和 `/readyz` 行为符合基础契约。
- 业务错误响应包含 `code`、中文 `message` 和 `requestId`。
- 日志输出到 stdout。
- 不存在 AI、FTPS、对象存储、多用户、RBAC 相关后端模块。

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

## 建议目录

```txt
backend/
├── plans/
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

## 技术选型待定项

| 项目 | 推荐 | 决策要求 |
|---|---|---|
| HTTP 框架 | Gin 或 Echo | API 面小，优先团队熟悉度和中间件生态 |
| DB 访问 | sqlc 或 pgx 手写 repository | migration 和 SQL 可审计 |
| migration | golang-migrate 或 goose | Docker 启动可自动执行或显式执行 |
| 日志 | slog 或 zap | 输出 stdout，便于 Docker 收集 |
| FTP 库 | 需验证 | 必须支持 PASV 端口、自定义认证、上传完成 hook、根目录限制 |
| 密码哈希 | Argon2id 或 bcrypt | Web 账号和 FTP 账号都不能明文保存 |

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
DEFAULT_SYSTEM_USERNAME=admin
DEFAULT_SYSTEM_PASSWORD=
DEFAULT_FTP_USERNAME=camera
DEFAULT_FTP_PASSWORD=
DEFAULT_FTP_ANONYMOUS_ENABLED=true
```

## 实施步骤

1. 初始化 `go.mod` 和基础 `cmd/api/main.go`。
2. 建立配置加载、日志、数据库连接、优雅关闭。
3. 建立 API router 和统一错误响应。
4. 建立 migration 目录和启动检查。
5. 建立本地存储根目录、临时目录和可写性检查。
6. 建立 Dockerfile、docker-compose 和 `.env.example`。
7. 建立基础 health endpoint，用于容器 smoke test。

## 验收标准

- `go test ./...` 通过。
- Docker Compose 可以启动 app 和 PostgreSQL。
- 服务启动时能连接数据库并确认存储目录可写。
- 日志输出到 stdout。
- 不存在 AI、FTPS、对象存储、多用户、RBAC 相关后端模块。

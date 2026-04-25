# FalconDrop 后端执行计划（2026-04-25）

## 当前目标

从零实现 `backend/` 的 Go 后端 MVP，并同步更新 `docs/任务/后端` 的完成状态。

## 待办清单

- [x] 初始化后端工程：`go.mod`、目录结构、配置加载、日志、HTTP 启动与优雅关闭。
- [x] 落地 PostgreSQL：migration、seed（系统账号/FTP 账号/app settings）和基础 repository。
- [x] 落地认证：登录/登出/me、session middleware、系统账号更新、FTP 账号更新。
- [x] 落地存储与媒体：本地存储校验、上传入库逻辑、资产/照片查询、照片内容与删除。
- [x] 落地实时：WebSocket hub、事件广播、系统信息聚合接口。
- [x] 落地 FTP 管理：启动/停止/状态接口，匿名与账号配置联动，上传完成回调接入 media。
- [x] 补齐部署：`.env.example`、`deployments/Dockerfile`、`deployments/docker-compose.yml`。
- [x] 补齐测试并验证：`go test ./...`、`go vet ./...`、Compose config 校验。
- [x] 更新文档：`docs/任务/后端/未完成.md` 与必要任务分片状态说明。

## Review（进行中）

- 已确认当前仓库 `backend/` 为空目录，后端需从零开始。
- 已对齐实现目标与 `docs/任务/后端/tasks/01~08` 契约范围。
- 已完成后端主干交付：配置、数据库、migration、seed、认证、会话、FTP lifecycle、媒体资产、系统信息、WebSocket、Docker Compose。
- 已通过 `go test ./...`、`go vet ./...`、`docker compose -f deployments/docker-compose.yml config`。

# 08 后端测试与验收计划

## 依据文档

- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

建立后端可重复验证标准，覆盖认证、FTP、存储、PostgreSQL、EXIF、WebSocket、Docker Compose 和删除一致性。

## 测试分层

| 层级 | 范围 | 运行要求 |
|---|---|---|
| 单元测试 | password、配置解析、路径安全、hash、EXIF、事件类型 | `go test ./...` |
| repository 测试 | migration、seed、核心 SQL、事务 | 可使用测试 PostgreSQL |
| 集成测试 | HTTP API、WebSocket、FTP 上传、媒体入库 | 可使用 Docker Compose 或 testcontainers |
| 部署 smoke test | 真实 Compose 启动、端口映射、数据卷 | 手动或脚本化执行 |

## 单元测试

- 密码 hash 和 verify。
- session 创建和失效。
- FTP 账号匿名策略。
- 端口和 PASV 配置解析。
- 路径安全和路径穿越防护。
- hash 去重。
- 同名同 hash 覆盖。
- 同名不同 hash 内部路径生成。
- MIME 和照片识别。
- EXIF 时间解析和兜底时间。
- API 错误码到 HTTP status 的映射。
- WebSocket 事件 payload 不包含敏感字段。

## 集成测试

- migration 空库执行。
- 默认系统账号和 FTP 账号初始化。
- 登录后访问受保护 API。
- 未登录访问受保护 API 返回 401。
- FTP 匿名上传。
- FTP 默认账号上传。
- FTP 密码错误拒绝。
- 上传任意格式文件后 `media_assets` 和 `transfer_events` 入库。
- 上传照片后 `photo-added` WebSocket 事件广播。
- 删除照片后文件和数据库记录都消失。
- `/api/photos/{id}/content` 只能通过登录态访问。
- 非照片文件进入 `/api/assets`，但不进入 `/api/photos`。
- 同名同 hash 上传触发覆盖语义。
- 同名不同 hash 上传保留两条资产。

## Docker Smoke Test

1. `docker compose up -d`。
2. 等待 PostgreSQL 和 app healthy。
3. 使用默认系统账号登录。
4. 启动 FTP。
5. 使用 FTP 客户端上传照片。
6. 查询 `/api/photos`。
7. 删除照片。
8. 查询数据库和数据卷确认删除。

## 主路径验收脚本

建议保留一条可手动复现的命令链：

1. `docker compose -f deployments/docker-compose.yml up -d --build`
2. 调用登录接口保存 Cookie。
3. `POST /api/ftp/start`。
4. 使用 `curl --ftp-create-dirs -T ./fixtures/photo.jpg ftp://127.0.0.1:2121/photo.jpg` 或等价 FTP 客户端上传。
5. 查询 `/api/photos`，确认照片存在且时间字段完整。
6. 建立 `/api/ws`，再次上传照片并确认收到 `photo-added`。
7. `DELETE /api/photos/{id}`。
8. 查询 `/api/photos`、数据库和数据卷确认删除完成。

## 验收命令

```bash
go test ./...
go vet ./...
docker compose -f deployments/docker-compose.yml config
```

如实现包含前端静态资源嵌入，后端镜像构建前还需要前端构建通过。

## 风险专项测试

- Docker/NAT/PASV 连接测试。
- 匿名 FTP 开启后的未认证上传测试。
- 大文件上传中断后的临时文件清理。
- WebSocket 断线后 REST 快照补偿。
- 文件删除失败时数据库一致性。
- 服务器重启后 PostgreSQL 中已有照片仍可通过 API 查询和读取 content。
- 存储目录不可写时 `/readyz` 失败且系统信息页可展示不可写状态。
- `FTP_PUBLIC_HOST` 缺失时 Docker/NAT 提示可见。
- 日志不包含密码、session、password hash。

## 验收标准

- 所有 Go 测试通过。
- Docker Compose 配置合法。
- 主路径“登录 -> 启动 FTP -> 上传照片 -> 入库 -> WebSocket -> 删除”可复现。
- 不出现数据库记录已删除但文件仍静默残留的成功响应。
- 未登录无法访问受保护 REST API、图片 content API 和 WebSocket。
- API response 不暴露服务器真实路径。
- 不存在 AI、FTPS、对象存储、多用户、RBAC 模块或入口。

## 后续 P1 测试

- 缩略图生成。
- 照片搜索筛选。
- 系统日志展示。
- 备份恢复。

# 08 后端测试与验收计划

## 依据文档

- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

建立后端可重复验证标准，覆盖认证、FTP、存储、PostgreSQL、EXIF、WebSocket、Docker Compose 和删除一致性。

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

## Docker Smoke Test

1. `docker compose up -d`。
2. 等待 PostgreSQL 和 app healthy。
3. 使用默认系统账号登录。
4. 启动 FTP。
5. 使用 FTP 客户端上传照片。
6. 查询 `/api/photos`。
7. 删除照片。
8. 查询数据库和数据卷确认删除。

## 验收命令

```bash
go test ./...
go vet ./...
docker compose -f deployments/docker-compose.yml config
```

## 风险专项测试

- Docker/NAT/PASV 连接测试。
- 匿名 FTP 开启后的未认证上传测试。
- 大文件上传中断后的临时文件清理。
- WebSocket 断线后 REST 快照补偿。
- 文件删除失败时数据库一致性。

## 验收标准

- 所有 Go 测试通过。
- Docker Compose 配置合法。
- 主路径“登录 -> 启动 FTP -> 上传照片 -> 入库 -> WebSocket -> 删除”可复现。
- 不出现数据库记录已删除但文件仍静默残留的成功响应。

## 后续 P1 测试

- 缩略图生成。
- 照片搜索筛选。
- 系统日志展示。
- 备份恢复。

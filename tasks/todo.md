# FalconDrop Web 重写任务计划

## 当前目标

根据 `docs/rewrite-analysis/` 的确认范围，将 Web 重写工作细化为前端和后端计划文档。当前阶段只制定计划，不进入功能实现。

## 待办

- [x] 梳理 `docs/rewrite-analysis/` 的范围、功能、API、数据、页面和部署约束。
- [x] 生成 `fronted/plans/` 前端详细计划。
- [x] 生成 `backend/plans/` 后端详细计划。
- [x] 补充 `docs/任务/前端/未完成.md`。
- [x] 补充 `docs/任务/后端/未完成.md`。
- [x] 校验新增文件和 git diff。

## 范围结论

- 只做 Web。
- Docker Compose 是正式部署方式。
- 前端主页面只有照片和系统信息，另有登录入口。
- 后端使用 Go 提供 REST API、WebSocket 和 FTP 服务。
- PostgreSQL 是唯一持久化数据库。
- 本地数据卷是唯一文件存储。
- 支持默认系统账号、默认 FTP 账号和匿名 FTP。
- 不做 Android、Windows、Tauri、AI、FTPS、RBAC、多用户、对象存储。

## Review

- 已生成 `fronted/plans/01-*.md` 到 `06-*.md`，覆盖前端范围、登录、照片页、系统信息页、实时通信和验收。
- 已生成 `backend/plans/01-*.md` 到 `08-*.md`，覆盖后端基线、数据库、认证、FTP、媒体、实时事件、部署和验收。
- 已补充 `docs/任务/前端/未完成.md` 和 `docs/任务/后端/未完成.md`，作为可执行任务索引。
- 已抽检 `docs/superpowers/specs/*`，其内容是 Android/RAW 未来特性设计；由于 `12-confirmed-web-scope.md` 明确只做 Web，且不做 Android 原生能力，未纳入本轮 P0 计划。

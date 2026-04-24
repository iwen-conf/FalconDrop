# FalconDrop Web 重写任务计划

## 当前目标

根据 `docs/rewrite-analysis/` 的确认范围和 Lazycat 技能中的项目基线栈，继续细化 `docs/任务/后端` 与 `docs/任务/前端`。当前阶段只完善任务文档，不进入功能实现。

## 待办

- [x] 梳理 `docs/rewrite-analysis/` 的范围、功能、API、数据、页面和部署约束。
- [x] 对齐 Lazycat 技能基线：Go 后端；React + Vite + Tailwind CSS + shadcn/ui + Zustand + TanStack Query + React Router + React Hook Form + Zod + Framer Motion 前端。
- [x] 细化 `docs/任务/后端/未完成.md` 和后端任务分片。
- [x] 细化 `docs/任务/前端/未完成.md` 和前端任务分片。
- [x] 校验文档间一致性和 git diff。

## 范围结论

- 只做 Web。
- Docker Compose 是正式部署方式。
- 前端主页面只有照片和系统信息，另有登录入口。
- 后端使用 Go 提供 REST API、WebSocket 和 FTP 服务。
- PostgreSQL 是唯一持久化数据库。
- 本地数据卷是唯一文件存储。
- 支持默认系统账号、默认 FTP 账号和匿名 FTP。
- Lazycat 创建基线默认要求注册、双 token 和无感刷新；本项目按 `12-confirmed-web-scope.md` 豁免注册和多用户，只保留唯一默认系统账号登录，认证机制以后端安全实现为准。
- 不做 Android、Windows、Tauri、AI、FTPS、RBAC、多用户、对象存储。

## Review

- 已补充 `docs/任务/后端/未完成.md`：增加技术栈、范围豁免、交付里程碑、任务依赖顺序，并按工程、数据、FTP、媒体、实时事件、部署验证重组 P0 清单。
- 已细化后端 8 个任务分片：补充模块边界、启动流程、schema 约束、事务边界、认证会话、FTP 状态机、媒体入库决策、WebSocket 契约、Compose 运维和测试验收。
- 已补充 `docs/任务/前端/未完成.md`：明确 Lazycat 前端基线栈、UI 风格、里程碑和 P0 分组。
- 已细化前端 6 个任务分片：补充 React 技术栈职责、路由与状态边界、登录会话、照片页组件拆分、系统页表单与提示、REST/WebSocket 契约和测试验收。
- 已明确 Lazycat 通用基线中的注册、双 token、无感刷新与本项目确认范围冲突，P0 按唯一默认系统账号登录处理；后续若产品范围变更需同步更新认证和前端会话文档。
- 已运行 `git diff --check`，未发现 whitespace 错误。

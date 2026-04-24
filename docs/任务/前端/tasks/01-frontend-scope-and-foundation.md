# 01 前端范围与工程基线计划

## 依据文档

- `docs/rewrite-analysis/05-page-map.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

将前端收敛为纯 Web 管理界面，只服务“相机 FTP 图传 Web 服务”。前端只提供登录、照片、系统信息三个访问面，其中登录是入口页，主功能页只有照片和系统信息。

## 范围

- 使用 React + TypeScript + Vite。
- 页面文案除专业术语外全部使用中文。
- 专业术语允许保留英文：`Docker Compose`、`PostgreSQL`、`WebSocket`、`FTP`、`EXIF`、`hash`、`API`。
- 不引入 Android、Windows、Tauri、AI 修图、FTPS、RBAC、多用户、审计日志、对象存储相关页面。

## 建议目录

```txt
fronted/
├── plans/
├── src/
│   ├── app/
│   │   ├── App.tsx
│   │   ├── routes.tsx
│   │   └── providers.tsx
│   ├── api/
│   │   ├── client.ts
│   │   ├── errors.ts
│   │   └── schemas.ts
│   ├── components/
│   │   ├── layout/
│   │   └── ui/
│   ├── features/
│   │   ├── auth/
│   │   ├── photos/
│   │   └── system-info/
│   ├── stores/
│   ├── types/
│   └── utils/
├── package.json
├── vite.config.ts
└── tsconfig.json
```

> 当前仓库目录名为 `fronted/`，计划先沿用现状；如后续统一命名为 `frontend/`，需要同步调整 Docker 构建上下文和文档引用。

## 实施步骤

1. 初始化 Vite + React + TypeScript 工程骨架。
2. 配置 ESLint、TypeScript strict、基础测试框架。
3. 建立 `api/client.ts`，统一处理 `credentials`、JSON 解析、错误响应和 `requestId`。
4. 建立 `app/providers.tsx`，集中挂载 Query Client、路由、全局错误边界和 toast。
5. 建立最小路由模型：`/login`、`/photos`、`/system`，默认 `/` 跳转 `/photos`。
6. 建立中文 UI 文案规范，避免复用原 Tauri/Android/Windows 平台文案。
7. 建立共享类型：账号、FTP 状态、系统信息、媒体资产、照片、WebSocket 事件。
8. 将 REST 快照与 WebSocket 增量分开建模，页面初始加载必须调用 REST。

## 验收标准

- `npm run typecheck` 通过。
- `npm run build` 通过。
- 未登录访问 `/photos` 或 `/system` 会跳转到 `/login`。
- 登录后可以在照片和系统信息两个主页面之间切换。
- 前端源码中不出现 Tauri IPC、Android Bridge、Windows 托盘、AI 修图入口。

## 不做项

- 不做多页面后台管理台。
- 不做角色权限菜单。
- 不做 Android 权限导览。
- 不做 Windows 本地路径选择。
- 不做 AI 修图配置页。
- 不做 FTPS 证书配置页。

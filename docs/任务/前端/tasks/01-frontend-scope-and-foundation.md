# 01 前端范围与工程基线计划

## 依据文档

- `docs/rewrite-analysis/05-page-map.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

将前端收敛为纯 Web 管理界面，只服务“相机 FTP 图传 Web 服务”。前端只提供登录、照片、系统信息三个访问面，其中登录是入口页，主功能页只有照片和系统信息。

## 范围

- 使用 React + TypeScript + Vite + Tailwind CSS + shadcn/ui + Zustand + TanStack Query + React Router + React Hook Form + Zod + Framer Motion。
- 页面文案除专业术语外全部使用中文。
- 专业术语允许保留英文：`Docker Compose`、`PostgreSQL`、`WebSocket`、`FTP`、`EXIF`、`hash`、`API`。
- 不引入 Android、Windows、Tauri、AI 修图、FTPS、RBAC、多用户、审计日志、对象存储相关页面。
- Lazycat 通用基线默认包含注册、双 token 和无感刷新；本项目按确认范围豁免注册和多用户，只做唯一默认系统账号登录与会话恢复。

## 建议目录

```txt
fronted/
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

## 技术栈职责

| 技术 | 用途 |
|---|---|
| React + TypeScript + Vite | 应用骨架、类型安全、构建 |
| Tailwind CSS | 基础布局和状态样式 |
| shadcn/ui | Button、Input、Dialog、Tabs、Dropdown、Tooltip、Form 等基础组件 |
| React Router | `/login`、`/photos`、`/system` 路由和守卫 |
| TanStack Query | REST 快照、缓存、刷新、错误状态 |
| Zustand | 会话摘要、WebSocket 连接状态、照片预览等局部 UI 状态 |
| React Hook Form + Zod | 登录、系统账号、FTP 账号表单校验 |
| Framer Motion | 预览层、确认弹窗、列表增量插入的轻量动效 |

## 视觉与交互基线

- 首屏就是可用产品界面，不做营销式 landing page。
- 页面整体应像现场运维工具：信息密度适中、状态清晰、操作可预期。
- 主题使用低饱和 shadcn CSS variables，建议 Sage 或 Slate；避免高饱和渐变和装饰性背景。
- 卡片只用于独立信息块，不做卡片嵌套卡片。
- 图标按钮优先使用 lucide-react，例如启动、停止、刷新、删除、复制连接 URL。
- 所有危险操作需要确认，删除文案必须说明会同时删除本地文件和数据库记录。
- 页面文案全部中文；技术名词按范围允许保留英文。

## 路由结构

| 路由 | 访问 | 说明 |
|---|---|---|
| `/login` | 未登录优先 | 默认系统账号登录 |
| `/photos` | 已登录 | 默认主页面，照片分组、预览、删除 |
| `/system` | 已登录 | 系统信息、FTP 状态、账号配置 |
| `/` | 自动跳转 | 已登录跳 `/photos`，未登录跳 `/login` |

## 全局状态边界

- TanStack Query 保存服务端快照，不把完整照片列表复制进 Zustand。
- Zustand 只保存会话摘要、WebSocket 状态、当前预览照片 id、局部筛选和 UI 开关。
- 表单 draft 由 React Hook Form 管理，不长期放全局 store。
- WebSocket 事件到达后优先更新 Query cache；断线重连后 invalidate 对应 query。

## 实施步骤

1. 初始化 Vite + React + TypeScript 工程骨架。
2. 配置 ESLint、TypeScript strict、基础测试框架。
3. 配置 Tailwind CSS、shadcn/ui 和主题变量。
4. 安装并配置 React Router、TanStack Query、Zustand、React Hook Form、Zod、Framer Motion、lucide-react。
5. 建立 `api/client.ts`，统一处理 `credentials`、JSON 解析、错误响应和 `requestId`。
6. 建立 `app/providers.tsx`，集中挂载 Query Client、路由、全局错误边界和 toast。
7. 建立最小路由模型：`/login`、`/photos`、`/system`，默认 `/` 跳转 `/photos`。
8. 建立中文 UI 文案规范，避免复用原 Tauri/Android/Windows 平台文案。
9. 建立共享类型：账号、FTP 状态、系统信息、媒体资产、照片、WebSocket 事件。
10. 将 REST 快照与 WebSocket 增量分开建模，页面初始加载必须调用 REST。

## 验收标准

- `npm run typecheck` 通过。
- `npm run build` 通过。
- Tailwind CSS 和 shadcn/ui 可正常构建。
- React Router、TanStack Query、Zustand、React Hook Form、Zod、Framer Motion 在工程中有明确使用位置。
- 未登录访问 `/photos` 或 `/system` 会跳转到 `/login`。
- 登录后可以在照片和系统信息两个主页面之间切换。
- 前端源码中不出现 Tauri IPC、Android Bridge、Windows 托盘、AI 修图入口。
- 页面文案除允许技术名词外为中文。

## 不做项

- 不做注册页。
- 不做多页面后台管理台。
- 不做角色权限菜单。
- 不做 Android 权限导览。
- 不做 Windows 本地路径选择。
- 不做 AI 修图配置页。
- 不做 FTPS 证书配置页。

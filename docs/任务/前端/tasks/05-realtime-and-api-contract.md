# 05 实时通信与 API 契约计划

## 依据文档

- `docs/rewrite-analysis/02-business-flows.md`
- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`

## 目标

统一前端 REST 和 WebSocket 接入方式。REST 用于页面快照，WebSocket 只负责增量事件；WebSocket 断线或事件丢失后必须通过 REST 快照补偿。

## 模块边界

- `api/client.ts` 是唯一 REST 入口，统一 Cookie、JSON、错误解析和 requestId。
- `api/ws.ts` 是唯一 WebSocket 入口，统一连接、重连、订阅、关闭。
- 业务页面不直接调用 `fetch` 或 `new WebSocket`。
- DTO 类型优先由 OpenAPI 或共享 schema 生成；未生成前手写在 `api/schemas.ts` 并用 Zod 校验关键响应。

## REST 错误模型

```ts
type ApiError = {
  code: string;
  message: string;
  requestId?: string;
};
```

前端规则：

- 用户看到 `message` 的中文文案。
- 开发调试可展示 `requestId`。
- 401 统一触发会话失效处理。
- 404 中的 `MEDIA_NOT_FOUND` 需要驱动照片列表刷新。

## fetch wrapper 规则

- 默认带 `credentials: 'include'`。
- JSON 请求自动设置 `Content-Type: application/json`。
- 204 响应返回 `undefined`。
- 非 2xx 响应统一解析为 `ApiError`。
- 网络错误包装为 `NETWORK_ERROR`，展示中文提示。
- 401 触发全局 session 失效流程，但登录接口本身的 401 只展示登录错误。
- 所有 mutation 成功后由调用方决定 invalidate 哪些 Query。

## WebSocket 事件模型

```ts
type RealtimeEvent =
  | { type: 'system-status'; payload: SystemInfo }
  | { type: 'ftp-started'; payload: FtpStatus }
  | { type: 'ftp-stopped'; payload: FtpStatus }
  | { type: 'asset-uploaded'; payload: AssetEvent }
  | { type: 'asset-overwritten'; payload: AssetEvent }
  | { type: 'photo-added'; payload: PhotoItem }
  | { type: 'photo-deleted'; payload: { id: string } };
```

如果后端使用统一事件包装，前端应先解析外层：

```ts
type ServerEvent<T extends RealtimeEvent = RealtimeEvent> = {
  eventId?: string;
  type: T['type'];
  payload: T['payload'];
  createdAt: string;
};
```

## 连接策略

1. 登录后建立 WebSocket。
2. 未登录或会话过期时关闭 WebSocket。
3. 断线后指数退避重连。
4. 重连成功后触发照片页和系统信息页重新拉取 REST 快照。
5. 不把 WebSocket 作为唯一数据来源。

## 重连状态

```ts
type RealtimeConnectionState =
  | 'idle'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'disconnected'
  | 'unauthorized';
```

- `connected` 时页面可显示实时。
- `reconnecting` 时页面显示“实时连接正在恢复”，但保留 REST 快照。
- `unauthorized` 时关闭连接并交给 auth 处理。
- 最大重连间隔需要设置上限，避免长时间静默。

## 模块划分

- `api/client.ts`：REST 基础请求。
- `api/errors.ts`：错误解析和错误码映射。
- `api/ws.ts`：WebSocket 连接、重连和事件分发。
- `stores/realtimeStore.ts`：保存连接状态，不保存完整业务数据。

## Query 更新规则

| 事件 | Query 操作 |
|---|---|
| `photo-added` | 插入 `['photos']` 缓存或 invalidate |
| `asset-overwritten` | 更新对应照片/资产或 invalidate |
| `photo-deleted` | 从 `['photos']` 移除，必要时 invalidate |
| `system-status` | 更新或 invalidate `['system-info']` |
| `ftp-started` / `ftp-stopped` | 更新 `['ftp-status']` 并 invalidate `['system-info']` |

如果当前缓存中没有足够上下文，优先 invalidate，不写复杂脆弱的手动合并。

## Zod 校验

- 登录表单、账号表单必须用 Zod。
- REST response 可先用 TypeScript 类型约束；对 WebSocket 事件建议用 Zod 做运行时校验，避免未知事件破坏页面。
- 未识别事件类型只记录 warning，不弹错误给用户。

## 实施步骤

1. 定义 REST DTO 和 WebSocket 事件类型。
2. 建立 fetch wrapper，统一处理 Cookie、JSON、错误响应。
3. 建立 WebSocket client，支持订阅、取消订阅、重连和关闭。
4. 在照片页处理 `photo-added`、`asset-overwritten`、`photo-deleted`。
5. 在系统信息页处理 `system-status`、`ftp-started`、`ftp-stopped`。
6. 重连后触发 REST 快照刷新。
7. 为错误解析、401、WebSocket 重连、事件分发写测试。
8. 禁止页面直接 `fetch` 和直接 `new WebSocket`，可通过代码审查或 lint 约定执行。
9. 为未知事件、坏 payload、401 close、logout close 写测试。

## 验收标准

- REST 错误以中文提示展示。
- WebSocket 断开时 UI 有连接状态提示。
- WebSocket 重连后页面数据能通过 REST 快照纠正。
- 上传、覆盖、删除和 FTP 状态变化能实时更新。
- 退出登录后 WebSocket 关闭，Query cache 清理。
- 页面代码不直接拼接服务器文件路径。

## 不做项

- 不使用 SSE。
- 不做离线编辑队列。
- 不用 WebSocket 承载登录认证。
- 不做 WebSocket 事件持久回放。

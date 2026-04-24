# 05 实时通信与 API 契约计划

## 依据文档

- `docs/rewrite-analysis/02-business-flows.md`
- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`

## 目标

统一前端 REST 和 WebSocket 接入方式。REST 用于页面快照，WebSocket 只负责增量事件；WebSocket 断线或事件丢失后必须通过 REST 快照补偿。

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

## 连接策略

1. 登录后建立 WebSocket。
2. 未登录或会话过期时关闭 WebSocket。
3. 断线后指数退避重连。
4. 重连成功后触发照片页和系统信息页重新拉取 REST 快照。
5. 不把 WebSocket 作为唯一数据来源。

## 模块划分

- `api/client.ts`：REST 基础请求。
- `api/errors.ts`：错误解析和错误码映射。
- `api/ws.ts`：WebSocket 连接、重连和事件分发。
- `stores/realtimeStore.ts`：保存连接状态，不保存完整业务数据。

## 实施步骤

1. 定义 REST DTO 和 WebSocket 事件类型。
2. 建立 fetch wrapper，统一处理 Cookie、JSON、错误响应。
3. 建立 WebSocket client，支持订阅、取消订阅、重连和关闭。
4. 在照片页处理 `photo-added`、`asset-overwritten`、`photo-deleted`。
5. 在系统信息页处理 `system-status`、`ftp-started`、`ftp-stopped`。
6. 重连后触发 REST 快照刷新。
7. 为错误解析、401、WebSocket 重连、事件分发写测试。

## 验收标准

- REST 错误以中文提示展示。
- WebSocket 断开时 UI 有连接状态提示。
- WebSocket 重连后页面数据能通过 REST 快照纠正。
- 上传、覆盖、删除和 FTP 状态变化能实时更新。

## 不做项

- 不使用 SSE。
- 不做离线编辑队列。
- 不用 WebSocket 承载登录认证。

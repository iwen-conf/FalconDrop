# 06 系统信息与实时事件计划

## 依据文档

- `docs/rewrite-analysis/02-business-flows.md`
- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/05-page-map.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

提供系统信息聚合 API 和 WebSocket 事件通道，使前端能展示版本、系统 hash、系统时间、账号、FTP 状态、存储状态，并实时响应上传、覆盖、删除和 FTP 状态变化。

## API

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/api/system/info` | 系统信息聚合快照 |
| `GET` | `/api/ws` | WebSocket 事件通道 |

## 系统信息响应

```go
type SystemInfo struct {
    Version       string
    BuildHash     string
    SystemTime    time.Time
    SystemAccount AccountSummary
    FtpAccount    FtpAccountSummary
    FtpStatus     FtpStatus
    Storage       StorageStatus
}
```

## 存储状态

```go
type StorageStatus struct {
    Root       string
    Writable   bool
    FreeBytes  *int64
    LastError  string
    CheckedAt  time.Time
}
```

## WebSocket 事件

| 事件 | 触发 |
|---|---|
| `system-status` | 系统或存储状态变化 |
| `ftp-started` | FTP 启动成功 |
| `ftp-stopped` | FTP 停止成功 |
| `asset-uploaded` | 任意资产上传入库 |
| `asset-overwritten` | 同名同 hash 覆盖 |
| `photo-added` | 照片资产入库 |
| `photo-deleted` | 照片删除成功 |

## 事件原则

- WebSocket 只做增量通知。
- 前端重连后应重新拉取 REST 快照。
- 每个事件带 `createdAt`。
- 可选带 `eventId`，方便后续补偿。
- 不保证客户端永远在线，不把 WebSocket 当持久队列。

## 实施步骤

1. 实现 `system.Service` 聚合版本、build hash、系统时间、账号、FTP 状态、存储状态。
2. 在构建阶段注入 `version` 和 `buildHash`。
3. 实现 `realtime.Hub`，管理连接、订阅和广播。
4. 将 FTP 启停事件接入 hub。
5. 将 media 上传、覆盖、删除事件接入 hub。
6. 实现 WebSocket 鉴权，只允许已登录会话连接。
7. 实现 ping/pong 和连接清理。
8. 编写系统信息、WebSocket 鉴权、事件广播测试。

## 验收标准

- `/api/system/info` 返回版本、hash、系统时间、系统账号、FTP 账号、FTP 状态和存储状态。
- 未登录不能连接 `/api/ws`。
- 上传照片后广播 `photo-added`。
- 删除照片后广播 `photo-deleted`。
- 启停 FTP 后广播对应事件。

## 不做项

- 不做 SSE。
- 不做持久消息队列。
- 不做审计日志流。

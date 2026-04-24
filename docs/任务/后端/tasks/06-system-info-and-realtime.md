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

## 模块边界

- `system.Service` 只做聚合查询和状态探测，不直接修改 FTP、账号或媒体数据。
- `realtime.Hub` 只做在线连接和事件广播，不承担持久化队列。
- REST 快照是权威数据；WebSocket 事件只用于增量刷新和提示。

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

## 响应字段要求

| 字段 | 来源 | 要求 |
|---|---|---|
| `version` | 构建注入或配置 | 没有注入时返回明确默认值，如 `dev` |
| `buildHash` | 构建注入 | 没有注入时返回 `unknown`，不能报错 |
| `systemTime` | 服务端当前时间 | 使用 ISO 8601 |
| `systemAccount` | `system_accounts` | 只返回 username、updatedAt，不返回 hash |
| `ftpAccount` | `ftp_account` | 返回 username、anonymousEnabled、updatedAt，不返回 hash |
| `ftpStatus` | `ftpserver.Manager` | 包含 running、host、publicHost、port、passivePorts、url 和统计 |
| `storage` | storage checker | 包含 root 展示值、writable、freeBytes、lastError、checkedAt |

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

## 存储检查规则

- 检查 `STORAGE_ROOT` 和 `TMP_ROOT` 是否存在、是否可写。
- 尽量返回剩余空间；获取失败不影响主流程，但 `lastError` 应说明原因。
- 不向前端返回宿主机敏感绝对路径时，可返回容器内路径或配置展示值。
- 存储检查失败时 `/api/system/info` 仍返回 200，但 `storage.writable=false`；`/readyz` 可按部署策略返回失败。

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

## 统一事件包装

```ts
type ServerEvent<T> = {
  eventId?: string;
  type: string;
  payload: T;
  createdAt: string;
};
```

- `eventId` P0 可选；如果生成，建议使用 UUID。
- `createdAt` 必填。
- payload 只包含前端需要更新的摘要，不包含服务器真实路径。

## 事件原则

- WebSocket 只做增量通知。
- 前端重连后应重新拉取 REST 快照。
- 每个事件带 `createdAt`。
- 可选带 `eventId`，方便后续补偿。
- 不保证客户端永远在线，不把 WebSocket 当持久队列。
- 广播失败不能回滚已经提交的数据库事务。
- 单个慢客户端不能阻塞全局广播。

## 鉴权与连接规则

- `/api/ws` 必须复用 HTTP 登录态。
- 未登录、session 过期或 Cookie 无效时拒绝升级或立即关闭连接。
- 连接建立后定期 ping/pong，超时清理。
- logout 后可以不强制踢掉所有既有连接，但下一次 ping 或业务事件前应能检测会话失效；若实现不了，logout 时关闭当前浏览器连接。
- WebSocket 不承载登录请求。

## 事件来源

| 来源 | 事件 |
|---|---|
| FTP manager start/stop | `ftp-started`、`ftp-stopped` |
| FTP manager 统计变化 | 可选 `system-status` 或 `ftp-status`，P0 可通过系统页轮询/刷新补偿 |
| media service 上传新资产 | `asset-uploaded` |
| media service 同名同 hash 覆盖 | `asset-overwritten` |
| media service 新照片 | `photo-added` |
| media service 删除照片 | `photo-deleted` |
| storage checker 状态变化 | `system-status` |

## 实施步骤

1. 实现 `system.Service` 聚合版本、build hash、系统时间、账号、FTP 状态、存储状态。
2. 在构建阶段注入 `version` 和 `buildHash`。
3. 实现 `realtime.Hub`，管理连接、订阅和广播。
4. 将 FTP 启停事件接入 hub。
5. 将 media 上传、覆盖、删除事件接入 hub。
6. 实现 WebSocket 鉴权，只允许已登录会话连接。
7. 实现 ping/pong 和连接清理。
8. 编写系统信息、WebSocket 鉴权、事件广播测试。
9. 编写慢客户端、断线清理和重连后 REST 补偿的测试或验证脚本。
10. 在 OpenAPI 或共享类型中固定事件 payload。

## 验收标准

- `/api/system/info` 返回版本、hash、系统时间、系统账号、FTP 账号、FTP 状态和存储状态。
- 未登录不能连接 `/api/ws`。
- 上传照片后广播 `photo-added`。
- 删除照片后广播 `photo-deleted`。
- 启停 FTP 后广播对应事件。
- WebSocket 断线不会影响 FTP 上传入库。
- 事件 payload 不包含服务器真实路径和密码 hash。

## 不做项

- 不做 SSE。
- 不做持久消息队列。
- 不做审计日志流。
- 不做客户端错过事件后的服务端回放；前端通过 REST 快照补偿。

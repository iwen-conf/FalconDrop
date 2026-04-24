# 04 FTP 服务与本地存储计划

## 依据文档

- `docs/rewrite-analysis/01-feature-map.md`
- `docs/rewrite-analysis/02-business-flows.md`
- `docs/rewrite-analysis/07-config-and-dependencies.md`
- `docs/rewrite-analysis/08-deployment.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

实现 Go 管理的 FTP 服务，支持启动、停止、状态查询、默认 FTP 账号、匿名访问、PASV 端口范围、本地数据卷存储和上传完成 hook。

## 模块边界

- `ftpserver` 负责 FTP 协议、连接状态、认证、PASV、root jail、上传完成通知。
- `storage` 负责本地数据卷路径、临时文件、正式路径提交、删除和路径安全。
- `media` 负责 hash、MIME、EXIF、入库和 WebSocket 业务事件。
- FTP 层不直接写 `media_assets`，只把完成后的文件交给 media service。

## API

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/api/ftp/status` | 获取 FTP 状态和连接信息 |
| `POST` | `/api/ftp/start` | 启动 FTP 服务 |
| `POST` | `/api/ftp/stop` | 停止 FTP 服务 |

## FTP 运行状态

```go
type FtpStatus struct {
    IsRunning        bool
    Host             string
    PublicHost       string
    Port             int
    PassivePorts     string
    URL              string
    AnonymousEnabled bool
    Username         string
    ConnectedClients int
    FilesReceived    int
    BytesReceived    int64
    LastFile         string
    UpdatedAt        time.Time
}
```

## 状态机

| 当前状态 | 操作 | 下一个状态 | 规则 |
|---|---|---|---|
| `stopped` | start | `starting` | 校验端口、PASV、存储、账号配置 |
| `starting` | 启动成功 | `running` | 广播 `ftp-started` |
| `starting` | 启动失败 | `stopped` | 返回稳定错误码 |
| `running` | start | `running` | 幂等返回当前状态 |
| `running` | stop | `stopping` | 停止监听并等待连接收敛 |
| `stopping` | 停止完成 | `stopped` | 广播 `ftp-stopped` |
| `stopped` | stop | `stopped` | 幂等成功 |

停止后迟到的连接数或上传统计不能把状态恢复成 `running`。

## FTP 库验证清单

必须在实施前用 spike 验证：

1. 支持 PASV 被动端口范围配置。
2. 支持自定义认证，包含匿名和用户名密码。
3. 支持限制用户根目录，防止路径穿越。
4. 支持上传完成 hook 或可包装 storage backend。
5. 支持获取 remote address。
6. 支持 Docker/NAT 环境下设置 public host。
7. 支持优雅停止。
8. 支持或可包装临时文件上传，避免半文件进入正式库。
9. 支持大文件流式写入，不把完整文件读入内存。

## 启动前校验

- `FTP_PORT` 在 `1..65535` 范围内。
- PASV 端口范围合法，起始端口小于等于结束端口。
- PASV 端口范围不与 HTTP 端口和 FTP 控制端口冲突。
- `STORAGE_ROOT` 和 `TMP_ROOT` 存在且可写。
- `ftp_account` 存在；匿名关闭时账号和密码 hash 完整。
- `FTP_PUBLIC_HOST` 为空时仍可启动，但系统信息页要能提示 Docker/NAT 场景可能需要配置。

## 存储规则

- `STORAGE_ROOT` 是上传文件根目录。
- FTP 用户不能逃逸出 `STORAGE_ROOT`。
- 上传时先写临时文件，完成后再进入正式路径。
- 保留原文件名用于展示。
- 用内容 hash 作为去重依据。
- 同名且 hash 一致：允许覆盖，保持同一内容记录。
- 同名但 hash 不一致：不覆盖，生成不冲突内部路径。
- 允许所有格式上传，非照片也要落盘和入库。
- 路径规范化后必须仍在 `STORAGE_ROOT` 下。
- 禁止 null byte、绝对路径、`..` 逃逸和平台特殊路径。
- 文件名用于展示时保留原始值；内部路径可以转义或加 hash 后缀。

## 内部路径建议

```txt
uploads/
├── 2026/
│   └── 04/
│       └── 25/
│           ├── IMG_0001.jpg
│           └── IMG_0001__{hash8}.jpg
```

最终格式可在实现时确认，但必须满足同名不同 hash 并存。

## 上传完成事件

FTP 层传给 media service 的数据建议包含：

```go
type UploadedFileEvent struct {
    OriginalFilename string
    RelativePath     string
    TempPath         string
    Size             int64
    RemoteAddr       string
    CompletedAt      time.Time
}
```

处理规则：

1. FTP storage backend 写入 `TMP_ROOT`。
2. PUT 成功后发布 `UploadedFileEvent` 给 media service。
3. media service 计算 hash 和目标 storage path。
4. media service 原子移动临时文件到正式路径。
5. media service 写数据库和传输事件。
6. media service 发布 WebSocket 事件。
7. 如果 media 处理失败，写 `transfer_events.failed`，并清理临时文件。

## 统计规则

- `ConnectedClients` 由连接建立/断开事件维护。
- `FilesReceived` 只在 PUT 完成且进入处理流程后递增。
- `BytesReceived` 累加 PUT 完成字节数。
- `LastFile` 保存最近一次完成上传的原文件名。
- app 重启后运行态统计可归零，历史统计以 `transfer_events` 为准。

## 实施步骤

1. 完成 Go FTP 库 spike，记录结论。
2. 实现 `ftpserver.Manager`，管理启动、停止、状态快照和幂等逻辑。
3. 实现 FTP auth provider，读取唯一 FTP 账号和匿名策略。
4. 实现本地 storage backend 和路径安全检查。
5. 实现 PASV 端口和 public host 配置。
6. 实现上传完成 hook，交给 media service 处理 hash、入库和事件。
7. 实现连接数、文件数、字节数、最后文件统计。
8. 停止后忽略迟到统计，避免状态被恢复为运行中。
9. 编写 FTP 匿名上传、账号上传、密码错误、路径穿越、PASV 配置测试。
10. 编写中断上传、临时文件清理和大文件不爆内存测试。
11. 编写 Docker Compose 下 PASV 连接验证脚本或文档。

## 错误码

- `FTP_ALREADY_RUNNING`
- `FTP_NOT_RUNNING`
- `FTP_PORT_UNAVAILABLE`
- `FTP_PASSIVE_PORT_NOT_CONFIGURED`
- `FTP_AUTH_FAILED`
- `STORAGE_NOT_WRITABLE`
- `STORAGE_PATH_INVALID`
- `FTP_START_FAILED`
- `FTP_STOP_FAILED`
- `FTP_PUBLIC_HOST_REQUIRED`
- `FTP_PASSIVE_PORT_CONFLICT`

## 验收标准

- 匿名模式可连接并上传。
- 非匿名模式账号密码正确时可上传，错误时拒绝。
- FTP 控制端口和 PASV 端口可通过 Docker Compose 映射。
- 上传任意格式文件后本地数据卷出现文件。
- 上传失败或中断不会留下被当作正式资产的半文件。
- 路径穿越上传被拒绝。
- FTP 停止后状态稳定为 stopped。
- `/api/ftp/status` 能展示 running、连接数、接收文件数、字节数、最后文件和匿名状态。

## 不做项

- 不做 FTPS。
- 不做 IP 限制。
- 不做多 FTP 账号。
- 不做 NAS/S3/MinIO。

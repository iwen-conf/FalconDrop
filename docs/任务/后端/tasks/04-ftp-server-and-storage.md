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

## FTP 库验证清单

必须在实施前用 spike 验证：

1. 支持 PASV 被动端口范围配置。
2. 支持自定义认证，包含匿名和用户名密码。
3. 支持限制用户根目录，防止路径穿越。
4. 支持上传完成 hook 或可包装 storage backend。
5. 支持获取 remote address。
6. 支持 Docker/NAT 环境下设置 public host。
7. 支持优雅停止。

## 存储规则

- `STORAGE_ROOT` 是上传文件根目录。
- FTP 用户不能逃逸出 `STORAGE_ROOT`。
- 上传时先写临时文件，完成后再进入正式路径。
- 保留原文件名用于展示。
- 用内容 hash 作为去重依据。
- 同名且 hash 一致：允许覆盖，保持同一内容记录。
- 同名但 hash 不一致：不覆盖，生成不冲突内部路径。
- 允许所有格式上传，非照片也要落盘和入库。

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

## 错误码

- `FTP_ALREADY_RUNNING`
- `FTP_NOT_RUNNING`
- `FTP_PORT_UNAVAILABLE`
- `FTP_PASSIVE_PORT_NOT_CONFIGURED`
- `FTP_AUTH_FAILED`
- `STORAGE_NOT_WRITABLE`
- `STORAGE_PATH_INVALID`

## 验收标准

- 匿名模式可连接并上传。
- 非匿名模式账号密码正确时可上传，错误时拒绝。
- FTP 控制端口和 PASV 端口可通过 Docker Compose 映射。
- 上传任意格式文件后本地数据卷出现文件。
- 路径穿越上传被拒绝。
- FTP 停止后状态稳定为 stopped。

## 不做项

- 不做 FTPS。
- 不做 IP 限制。
- 不做多 FTP 账号。
- 不做 NAS/S3/MinIO。

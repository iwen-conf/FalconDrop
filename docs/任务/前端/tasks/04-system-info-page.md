# 04 系统信息页面计划

## 依据文档

- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/05-page-map.md`
- `docs/rewrite-analysis/06-permission-model.md`
- `docs/rewrite-analysis/08-deployment.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

系统信息页展示 Web 服务的运行状态、系统版本、系统 hash、系统时间、默认系统账号、默认 FTP 账号、匿名状态、FTP 连接信息、PASV 端口和本地存储状态。

## API 契约

| 方法 | 路径 | 用途 |
|---|---|---|
| `GET` | `/api/system/info` | 获取系统信息聚合快照 |
| `GET` | `/api/ftp/status` | 获取 FTP 运行状态 |
| `POST` | `/api/ftp/start` | 启动 FTP 服务 |
| `POST` | `/api/ftp/stop` | 停止 FTP 服务 |
| `PUT` | `/api/system/account` | 更新默认系统账号 |
| `PUT` | `/api/ftp/account` | 更新默认 FTP 账号和匿名策略 |
| `GET` | `/api/ws` | 接收系统和 FTP 状态事件 |

## 页面结构

- 系统概览：版本、系统 hash、系统时间。
- Web 账号：当前默认系统账号、更新时间、修改密码入口。
- FTP 账号：用户名、匿名访问状态、修改密码入口。
- FTP 运行态：运行中/已停止、host、port、PASV 端口范围、连接 URL、当前连接数、接收文件数、接收字节数、最后接收文件。
- 存储状态：数据卷路径、可写状态、最近错误。
- 部署提示：Docker Compose、PASV 端口映射、防火墙提示。

## 风险提示规则

- 匿名 FTP 开启时必须显示醒目的中文提示：匿名访问已开启，任何能访问 FTP 端口的设备都可上传文件。
- 存储不可写时必须提示 FTP 上传会失败。
- PASV 端口未配置或状态异常时必须提示 Docker/NAT/防火墙可能导致相机连接失败。
- FTPS 不在范围内，不展示证书或 TLS 配置。

## 表单规则

### 默认系统账号

1. 支持修改用户名和密码。
2. 密码修改需要二次输入确认。
3. 保存成功后重新获取 `/api/auth/me`。

### 默认 FTP 账号

1. 支持修改用户名和密码。
2. 支持开启或关闭匿名访问。
3. 非匿名模式下用户名和密码必填。
4. 服务器运行中是否允许修改由后端决定；前端要按后端错误给出中文提示。

## 实施步骤

1. 建立 `features/system-info/systemInfoApi.ts`。
2. 建立系统信息聚合查询和 FTP 状态查询。
3. 实现系统概览卡片。
4. 实现 FTP 状态和连接信息卡片。
5. 实现匿名 FTP 风险提示。
6. 实现存储状态和 Docker/PASV 部署提示。
7. 实现系统账号和 FTP 账号编辑表单。
8. 接入 WebSocket 的 `system-status`、`ftp-started`、`ftp-stopped` 事件。

## 验收标准

- 页面能展示系统版本、系统 hash、系统时间、系统账号和 FTP 账号。
- FTP 启停后页面状态实时更新。
- 匿名 FTP 开启时有明确中文风险提示。
- 存储不可写或 PASV 配置异常时页面能展示中文诊断提示。
- 修改账号信息后页面快照刷新。

## 不做项

- 不做用户列表。
- 不做角色配置。
- 不做审计日志页面。
- 不做 FTPS 证书管理。
- 不做对象存储配置。

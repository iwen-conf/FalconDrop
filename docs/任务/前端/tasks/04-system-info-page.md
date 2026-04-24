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

## 页面定位

- `/system` 是运行与配置页，不是完整后台管理台。
- 页面优先帮助用户确认服务是否可用、相机应该连哪个 FTP 地址、匿名 FTP 是否开启、部署是否存在 PASV/存储风险。
- 所有配置只围绕唯一系统账号、唯一 FTP 账号和 FTP 启停，不扩展成多用户或复杂配置中心。

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

## 组件拆分

| 组件 | 职责 |
|---|---|
| `SystemInfoPage` | 页面 query、事件订阅、布局 |
| `SystemOverviewPanel` | version、build hash、系统时间 |
| `FtpStatusPanel` | FTP 状态、连接 URL、启动/停止、统计 |
| `FtpAccountForm` | FTP 用户名、密码、匿名策略 |
| `SystemAccountForm` | 系统账号用户名、密码更新 |
| `StorageStatusPanel` | 数据卷路径、可写性、剩余空间、错误 |
| `DeploymentHintsPanel` | Docker Compose、PASV、防火墙提示 |

使用 shadcn/ui 的 Card、Alert、Button、Switch、Input、Form、Dialog、Tooltip；复制连接 URL 用 lucide-react 图标按钮。

## 风险提示规则

- 匿名 FTP 开启时必须显示醒目的中文提示：匿名访问已开启，任何能访问 FTP 端口的设备都可上传文件。
- 存储不可写时必须提示 FTP 上传会失败。
- PASV 端口未配置或状态异常时必须提示 Docker/NAT/防火墙可能导致相机连接失败。
- FTPS 不在范围内，不展示证书或 TLS 配置。
- FTP 停止时不要显示为错误，而是明确“已停止”并提供启动操作。
- `FTP_PUBLIC_HOST` 为空时，如果后端返回提示，需要展示 Docker/NAT 下可能无法被相机访问。

## 表单规则

### 默认系统账号

1. 支持修改用户名和密码。
2. 密码修改需要二次输入确认。
3. 保存成功后重新获取 `/api/auth/me`。
4. 使用 React Hook Form + Zod 校验必填、两次密码一致。
5. 保存失败时保留表单内容并展示中文错误。

### 默认 FTP 账号

1. 支持修改用户名和密码。
2. 支持开启或关闭匿名访问。
3. 非匿名模式下用户名和密码必填。
4. 服务器运行中是否允许修改由后端决定；前端要按后端错误给出中文提示。
5. 匿名开启时仍允许保存默认账号，方便后续关闭匿名。
6. 匿名关闭时，保存前前端先做必填校验，后端仍做最终校验。

## FTP 启停交互

- 点击启动前显示当前端口和 PASV 范围。
- 启动中按钮进入 loading 并禁用重复提交。
- 启动成功后刷新 `/api/system/info` 和 `/api/ftp/status`。
- 停止操作幂等；停止成功后刷新状态。
- `FTP_PORT_UNAVAILABLE`、`FTP_PASSIVE_PORT_NOT_CONFIGURED`、`STORAGE_NOT_WRITABLE` 需要展示具体中文提示。

## TanStack Query 策略

- Query key：`['system-info']`、`['ftp-status']`。
- 页面进入先拉取 `/api/system/info`。
- FTP 启停和账号保存成功后 invalidate 两个 query。
- WebSocket 收到 `system-status`、`ftp-started`、`ftp-stopped` 后更新缓存或 invalidate。
- WebSocket 重连成功后重新拉取系统快照。

## 实施步骤

1. 建立 `features/system-info/systemInfoApi.ts`。
2. 建立系统信息聚合查询和 FTP 状态查询。
3. 实现系统概览卡片。
4. 实现 FTP 状态和连接信息卡片。
5. 实现匿名 FTP 风险提示。
6. 实现存储状态和 Docker/PASV 部署提示。
7. 实现系统账号和 FTP 账号编辑表单。
8. 接入 WebSocket 的 `system-status`、`ftp-started`、`ftp-stopped` 事件。
9. 为账号表单编写 Zod schema 和表单测试。
10. 为匿名 FTP、存储不可写、PASV 异常提示编写组件测试。

## 验收标准

- 页面能展示系统版本、系统 hash、系统时间、系统账号和 FTP 账号。
- FTP 启停后页面状态实时更新。
- 匿名 FTP 开启时有明确中文风险提示。
- 存储不可写或 PASV 配置异常时页面能展示中文诊断提示。
- 修改账号信息后页面快照刷新。
- 关闭匿名时，用户名和密码为空会在前端阻止提交并展示中文提示。
- 系统页不出现 FTPS、AI、对象存储、多用户配置入口。

## 不做项

- 不做用户列表。
- 不做角色配置。
- 不做审计日志页面。
- 不做 FTPS 证书管理。
- 不做对象存储配置。

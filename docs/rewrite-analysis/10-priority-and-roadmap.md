# Web 复刻优先级与路线图

## 优先级定义

- P0：Web 版必须实现，否则无法完成相机 FTP 上传到 Web 平台并在浏览器查看照片的主流程。
- P1：增强可用性、可靠性和运维体验，但不改变已确认产品边界。
- Out：已确认不做的原项目能力或扩展能力。

## P0：必须实现

| 优先级 | 功能 | 原项目位置 | Web 复刻说明 |
|---|---|---|---|
| P0 | Docker Compose 正式部署 | 原项目无 | app + postgres + 本地数据卷 + FTP 端口映射 |
| P0 | 默认系统账号登录 | 原项目无 | 只保留一个 Web 默认账号，不做角色 |
| P0 | Go FTP 服务启动/停止 | `commands/server.rs`、`ftp/server_factory.rs` | Go 后端管理 FTP server lifecycle |
| P0 | 默认 FTP 账号 | `AuthConfig`、`FtpAuthConfig`、`crypto.rs` | 只保留一个 FTP 默认账号 |
| P0 | 匿名 FTP | `AuthConfig` | 支持匿名，不限制 IP |
| P0 | 端口和 PASV 配置 | `network.rs`、`usePortCheck.ts` | Docker 部署必须支持控制端口和 passive port range |
| P0 | 本地数据卷存储 | `AppConfig.savePath` | 只做服务器本地存储 |
| P0 | 所有格式上传 | FTP PUT | 所有格式都允许落盘和入库 |
| P0 | hash 去重和同名规则 | 原项目无 | 同名同 hash 覆盖；同名不同 hash 不覆盖 |
| P0 | PostgreSQL schema/migrations | 原项目无 | system_accounts、ftp_account、app_settings、media_assets、transfer_events |
| P0 | 上传事件入库 | `ftp/listeners.rs`、`ftp/stats.rs` | 写 `media_assets`、`transfer_events` |
| P0 | 照片识别和 EXIF 时间 | `commands/exif.rs`、`file_index/service.rs` | 照片页按 EXIF 时间分类 |
| P0 | 照片页面 | `GalleryCard.tsx`、`PreviewWindow.tsx` | 展示上传的所有照片，支持预览和删除 |
| P0 | 系统信息页面 | 原项目无 | 展示系统版本、系统 hash、系统时间、系统账号、FTP 账号等 |
| P0 | WebSocket 实时刷新 | `server-events.ts` 可参考事件语义 | 上传、覆盖、删除和状态变化实时推送 |

## P1：增强功能

| 优先级 | 功能 | Web 复刻说明 |
|---|---|---|
| P1 | 部署检查页/状态块 | 检查 PASV 端口映射、数据卷可写、PostgreSQL 连接 |
| P1 | 更强 EXIF 兼容 | 扩展更多相机格式的拍摄时间解析 |
| P1 | 缩略图生成 | 提升照片页加载性能，但不改变本地存储边界 |
| P1 | 照片搜索筛选 | 按时间、文件名、hash、上传时间筛选 |
| P1 | 系统日志展示 | 查看关键运行日志和上传失败原因 |
| P1 | 备份恢复文档 | 覆盖 PostgreSQL 和本地数据卷 |

## Out：已确认不做

| 功能 | 原项目位置 | 不做原因 |
|---|---|---|
| RBAC、多用户、用户管理 | 原项目无 | 已确认只需要一个默认系统账号 |
| 审计日志 | 原项目无 | 已确认不做角色和审计系统 |
| AI 修图 | `src-tauri/src/ai_edit/` | 已确认不做 |
| FTPS | `crypto/tls.rs` | 已确认不需要 |
| 对象存储 S3/MinIO | 原项目无 | 已确认只做本地存储 |
| 外部分享链接 | 原项目 Android 分享 | 不在首期目标内 |
| Android MediaStore 写入 | `ftp/android_mediastore/`、`MediaStoreBridge.kt` | Web 版使用服务器本地存储 |
| Android 权限导览 | `PermissionBridge.kt` | 浏览器端无 Android 权限 |
| Android 前台服务保活 | `FtpForegroundService.kt` | Web 版由 Docker 管理 |
| Android 图库 Bridge/Activity | `GalleryBridgeV2.kt`、`ImageViewerActivity.kt` | React Web 照片页替代 |
| Windows 托盘 | `platform/windows.rs` | Web 版无桌面托盘 |
| Windows 开机自启动 | `platform/windows.rs` | Web 版用 Docker restart policy |
| Windows 自动预览窗口 | `auto_open/service.rs` | Web 照片预览替代 |
| Tauri IPC/窗口 API | `src-tauri/src/commands/` | Web REST/WebSocket 替代 |

## 第一阶段路线图：Web MVP

目标：相机可通过 FTP 上传到 Go 服务端，用户可在浏览器照片页实时看到照片，并在系统信息页看到系统与账号信息。

1. 搭建 Go API、React Web、PostgreSQL、Docker Compose 和本地数据卷。
2. 建立默认系统账号登录和会话鉴权。
3. 建立数据库迁移：system_accounts、ftp_account、app_settings、media_assets、transfer_events。
4. 实现 Go FTP server 启停、默认 FTP 账号、匿名访问和 PASV 配置。
5. 实现上传落盘、hash 计算、同名同 hash 覆盖、同名不同 hash 不覆盖。
6. 上传完成后写入媒体表和传输事件表。
7. 实现照片识别和 EXIF 时间提取。
8. 实现照片页：按 EXIF 时间分类、预览、删除。
9. 实现系统信息页：版本、hash、系统时间、系统账号、FTP 账号、FTP 状态。
10. 实现 WebSocket 实时刷新。

验收标准：

- Docker Compose 可作为正式部署方式启动。
- 默认 Web 账号能登录。
- 系统信息页能展示系统版本、系统 hash、系统时间、系统账号和 FTP 账号。
- FTP 默认账号和匿名模式均可连接。
- FTP 客户端上传任意格式文件后，本地存储和 PostgreSQL 均有记录。
- FTP 客户端上传照片后，照片页通过 WebSocket 实时出现照片。
- 照片按 EXIF 时间分类展示。
- 同名同 hash 上传按覆盖处理；同名不同 hash 不覆盖已有文件。
- 删除照片会直接删除本地文件和数据库记录。

## 第二阶段路线图：生产加固

1. 增加 Docker 部署检查和数据卷可写检查。
2. 强化 PASV 端口映射提示和故障诊断。
3. 补齐更多相机照片格式的 EXIF 时间解析。
4. 增加缩略图和照片搜索筛选。
5. 增加系统日志展示。
6. 补齐单元测试、集成测试和 E2E。

验收标准：

- Docker 部署问题能在系统信息页或日志中定位。
- FTP 在 Docker/NAT/防火墙环境下有清晰配置。
- 照片库能按时间和文件名筛选。
- 备份 PostgreSQL 和本地数据卷后可恢复服务。

## 复刻顺序建议

1. 先做 Docker + PostgreSQL + 默认账号登录。
2. 再做 FTP 上传落盘、hash 去重和媒体入库。
3. 再做照片页、系统信息页和 WebSocket。
4. 最后做部署检查、缩略图、搜索和日志。

# Web 复刻风险点与已确认问题

## 复刻风险点

| 风险点 | 影响 | 代码位置 | 解决建议 |
|---|---|---|---|
| 产品形态从本地应用变成 Web 服务 | 原项目很多平台能力不能直接复用 | `src-tauri/src/lib.rs`、Kotlin Bridge、`platform/windows.rs` | Web 版只保留 FTP 接收、照片页、系统信息页 |
| FTP 服务暴露在 Docker 主机网络上 | 匿名开放且不限制 IP，存在未授权上传风险 | `ftp/server.rs`、`FtpAuthConfig` | 系统信息页清晰展示匿名状态，部署文档强调网络边界 |
| PASV/NAT/防火墙配置复杂 | 相机连接失败，尤其 Docker 部署 | `network.rs`、`server_factory.rs` | Docker Compose 明确映射控制端口和 PASV 端口范围 |
| 原项目无 Web 登录 | Web 端需要最小登录保护 | 原项目无 | 新增唯一默认系统账号 |
| 浏览器不能访问服务器本地文件路径 | 原 Tauri `convertFileSrc` 方案不可用 | `PreviewWindow.tsx` | 通过 `/api/photos/{id}/content` 受控输出文件 |
| 原项目文件索引是内存模型 | Web 刷新后需要持久化 | `file_index/service.rs` | 媒体资产入 PostgreSQL |
| 所有格式都允许上传 | 非照片文件不能直接进入照片页 | `file_index/service.rs` | `media_assets` 记录所有资产，照片页只查 `is_photo=true` |
| EXIF 时间作为分类标准 | 部分照片缺少 EXIF 或解析失败 | `commands/exif.rs` | 保留兜底时间字段，但照片分组优先 EXIF 时间 |
| 同名文件处理复杂 | 同名同 hash 覆盖，同名不同 hash 不能覆盖 | 原项目无 | 存储路径使用 hash 或冲突后缀，数据库记录 content hash |
| 直接删除 | 删除失败可能造成数据库和本地文件不一致 | `GalleryBridge.kt` 可参考删除语义 | 删除流程需要事务边界和失败补偿策略 |
| WebSocket 事件丢失 | 照片页状态不准 | `server-events.ts`、`ftp/events.rs` | 页面加载必须先查 REST 快照，WebSocket 只做增量 |
| Docker 数据卷权限错误 | FTP 上传失败 | 原项目本机目录权限 | 启动时检查数据目录可写 |
| 原项目 TS bindings 来自 ts-rs | Go 版类型契约需重建 | `src-tauri/bindings/` | 使用 OpenAPI 或手写共享 DTO |

## 明确舍弃的原项目能力

| 能力 | 原项目位置 | Web 版处理 |
|---|---|---|
| Android MediaStore | `ftp/android_mediastore/`、`MediaStoreBridge.kt` | 不复刻 |
| Android 权限/前台服务 | `PermissionBridge.kt`、`FtpForegroundService.kt` | 不复刻 |
| Android 图库 Bridge | `GalleryBridgeV2.kt` | React Web 照片页替代 |
| Android 内置查看器 | `ImageViewerActivity.kt` | Web 预览替代 |
| Windows 托盘/自启 | `platform/windows.rs` | Docker restart policy 替代 |
| Windows 自动预览窗口 | `auto_open/service.rs` | Web 预览替代 |
| Tauri IPC | `commands/` | REST + WebSocket 替代 |
| 本地配置 JSON 作为唯一持久化 | `config_service.rs` | PostgreSQL settings 替代 |
| AI 修图 | `src-tauri/src/ai_edit/` | 不复刻 |
| FTPS | `crypto/tls.rs` | 不复刻 |
| RBAC/多用户 | 原项目无 | 不新增 |
| 对象存储 | 原项目无 | 不新增 |

## 当前项目可借鉴的部分

| 项目 | 借鉴方式 |
|---|---|
| FTP 生命周期 | 复用启停、幂等、事件 pipeline 的设计思想 |
| 端口检测 | 复用端口可用性检查和自动选端口思路 |
| 认证模型 | 保留 FTP 匿名/认证两种模式，但收敛为唯一默认账号 |
| 统计事件 | 保留上传、覆盖、删除、运行态事件语义 |
| EXIF | 保留字段设计：ISO、光圈、快门、焦距、拍摄时间 |
| 文件索引 | 参考按 EXIF 时间排序的思路，持久化到 PostgreSQL |

## 已确认问题

| 问题 | 结论 |
|---|---|
| Web 版部署在哪里 | Docker Compose，作为正式部署方式 |
| FTP 是否限制 IP | 不限制 IP |
| FTP 是否支持匿名 | 支持匿名 |
| FTP 是否必须支持 FTPS | 不需要 |
| 文件存储使用什么 | 服务器本地存储 |
| 数据库使用什么 | PostgreSQL |
| 实时通道使用什么 | WebSocket |
| 是否需要多用户和角色 | 不需要，只要一个默认系统账号 |
| FTP 账号数量 | 一个默认 FTP 账号 |
| AI 修图是否进入范围 | 不做 |
| 首期媒体格式范围 | 允许所有格式上传 |
| 页面数量 | 照片、系统信息两个主页面 |
| 删除语义 | 直接删除 |
| 排序/分类时间 | EXIF 时间 |

## 仍需实现时细化的问题

1. 无 EXIF 时间的照片在分组中显示到哪个兜底分组。
2. 同名不同 hash 的内部文件名冲突后缀格式。
3. 默认系统账号和默认 FTP 账号的初始化密码来源。
4. Docker Compose 中 PASV 端口范围的默认值。
5. WebSocket 断线重连后的事件补偿策略。

## 最大风险排序

1. FTP 部署网络风险：PASV、NAT、防火墙、Docker 端口映射会直接决定相机能否连接。
2. 匿名 FTP 安全风险：已确认支持且不限制 IP，需要依赖部署网络边界和清晰状态提示。
3. 存储一致性风险：上传文件、数据库记录、hash 覆盖和直接删除之间需要严格处理。

## Web 复刻前置检查清单

- 用真实相机或 FTP 客户端验证 Go FTP 库能力。
- 确认 Docker Compose 的 FTP 控制端口和 PASV 端口映射。
- 确认 PostgreSQL 和本地数据卷路径。
- 确认默认系统账号和默认 FTP 账号初始化方式。
- 建立 E2E 验收脚本：登录、启动 FTP、上传照片、照片入库、WebSocket 刷新、删除照片。

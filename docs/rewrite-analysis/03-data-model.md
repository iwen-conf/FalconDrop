# 数据模型分析

## 总体结论

当前项目没有数据库、ORM Model、迁移文件或服务端持久化表。核心数据分为四类：

1. 配置 JSON：`AppConfig` 及其子结构，持久化到本机配置文件。
2. 运行态内存数据：FTP server handle、统计快照、文件索引、AI 队列。
3. 平台数据：Windows 文件系统、Android MediaStore 记录。
4. 前后端/桥接 DTO：Tauri IPC 类型、Android JS Bridge JSON、事件 payload。

由于复刻目标现在是纯 Web，且已确认使用 PostgreSQL，建议新增一组收敛后的 Web 数据模型。下文先记录原项目真实 DTO，再给出 Web 版应新增的实体。

## Web 版新增实体建议

| 实体 | 用途 | 关键字段 |
|---|---|---|
| `SystemAccount` | 唯一 Web 默认账号 | id, username, passwordHash, updatedAt |
| `FtpAccount` | 唯一 FTP 默认账号 | id, username, passwordHash, anonymousEnabled, updatedAt |
| `AppSetting` | 系统配置 | key, valueJson, updatedAt |
| `MediaAsset` | 已上传文件资产 | id, originalFilename, storagePath, contentHash, size, mimeType, isPhoto, exifTakenAt, uploadedAt |
| `TransferEvent` | 上传、覆盖、删除事件 | id, assetId, eventType, originalFilename, contentHash, remoteAddr, createdAt |

不建 `Role`、`UserRole`、`AiEditJob`、`AuditLog` 作为首期核心表。Web 版不做 RBAC、多用户、AI 修图和审计日志。

## 实体：AppConfig

证据：`src-tauri/src/config.rs::AppConfig`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `savePath` | Path/String | 是 | 保存路径 | Windows 可选，Android 固定为 DCIM/CameraFTP |
| `port` | u16 | 是 | FTP 端口 | 高级连接启用时生效 |
| `autoSelectPort` | bool | 是 | 端口占用时自动选择 | 防止启动失败 |
| `advancedConnection` | `AdvancedConnectionConfig` | 是 | 高级连接设置 | 控制端口和 FTP 认证 |
| `previewConfig` | `PreviewWindowConfig?` | Windows | Windows 预览配置 | 仅 Windows 有效 |
| `androidImageViewer` | `AndroidImageViewerConfig?` | Android | Android 图片查看配置 | 仅 Android 有效 |
| `aiEdit` | `AiEditConfig` | 是 | AI 修图配置 | SeedEdit 自动/手动修图 |

### 关联关系

- `AppConfig` embeds `AdvancedConnectionConfig`。
- `AppConfig` optionally has one `PreviewWindowConfig`。
- `AppConfig` optionally has one `AndroidImageViewerConfig`。
- `AppConfig` has one `AiEditConfig`。

### 重要约束

- `port != 0`。
- `savePath` 不能为空。
- Android 运行时会强制归一化 `savePath` 为 `/storage/emulated/0/DCIM/CameraFTP`。
- Windows 默认端口 `21`，Android 默认端口 `2121`。

### 复刻建议

- Go 中建模为 Web 版配置 DTO，不要求保持原 Tauri `AppConfig` 完全兼容。
- 配置进入 PostgreSQL `app_settings`，不再保留 JSON 文件作为真源。
- 首期不包含 AI 配置和 API Key。

## 实体：AdvancedConnectionConfig

证据：`src-tauri/src/config.rs::AdvancedConnectionConfig`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `enabled` | bool | 是 | 是否启用高级连接 | false 时忽略自定义端口和认证 |
| `auth` | `AuthConfig` | 是 | FTP 认证配置 | 匿名或用户名密码 |

### 关联关系

- `AdvancedConnectionConfig` belongs to `AppConfig`。
- `AdvancedConnectionConfig` has one `AuthConfig`。

### 复刻建议

- 启动 FTP 时把 UI 配置转换成运行时 `FtpAuthConfig`，不要直接信任半配置状态。

## 实体：AuthConfig / FtpAuthConfig

证据：`src-tauri/src/config.rs::AuthConfig`、`src-tauri/src/ftp/types.rs::FtpAuthConfig`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `anonymous` | bool | 是 | 是否匿名访问 | true 时任何用户名/密码都可连接 |
| `username` | string | 否 | 用户名 | 非匿名模式需要 |
| `passwordHash` | string | 否 | Argon2id PHC hash | 非匿名模式需要 |

### 关联关系

- `AuthConfig` belongs to `AdvancedConnectionConfig`。
- `AuthConfig` converts to runtime `FtpAuthConfig`。

### 重要约束

- `anonymous=true` 或 `username` 为空或 `passwordHash` 为空时，运行时降级为匿名访问。
- 明文密码只通过 `save_auth_config` 传给后端一次，后端 hash 后持久化。

### 复刻建议

- Go 使用 `golang.org/x/crypto/argon2` 或成熟 password hashing 包保存 PHC 格式。
- API 层只接受明文 `password`，落盘只保存 hash。
- 复刻时增加后端校验：非匿名必须 username/password 非空，否则明确返回校验错误，而不是悄悄降级。

## 实体：PreviewWindowConfig

证据：`src-tauri/src/config.rs::PreviewWindowConfig`、`src/components/PreviewConfigCard.tsx`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `enabled` | bool | 是 | 是否自动预览 | 上传后是否自动打开图片 |
| `method` | enum | 是 | `built-in-preview`、`system-default`、`windows-photos`、`custom` | 图片打开方式 |
| `customPath` | string? | 否 | 自定义程序路径 | `method=custom` 时生效 |
| `autoBringToFront` | bool | 是 | 新图是否自动置顶 | 内置预览窗口行为 |

### 关联关系

- `PreviewWindowConfig` belongs to `AppConfig`，仅 Windows 使用。

### 复刻建议

- Web 版不复刻 `PreviewWindowConfig` 的窗口置顶、打开本地程序、打开文件夹等能力。
- 对应功能改为浏览器内图片预览页/弹窗，图片通过受控 HTTP 接口读取。

## 实体：AndroidImageViewerConfig

证据：`src-tauri/src/config.rs::AndroidImageViewerConfig`、`src/components/ConfigCard.tsx`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `openMethod` | enum | 是 | `built-in-viewer` 或 `external-app` | Android 打开图片方式 |
| `autoOpenLatestWhenVisible` | bool | 是 | 新图到达且应用可见时自动打开最新图 | Android 预览偏好 |

### 复刻建议

- 如果保留 Android 目标，仍需 Kotlin/Java Activity 或原生模块实现 viewer。
- React 只保存偏好和发起调用，不应承担 content URI 权限细节。

## 实体：AiEditConfig / ProviderConfig / SeedEditConfig

证据：`src-tauri/src/ai_edit/config.rs`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `autoEdit` | bool | 是 | 上传后是否自动修图 | 自动队列开关 |
| `prompt` | string | 否 | 自动修图提示词 | 自动修图需要非空 |
| `manualPrompt` | string | 否 | 手动修图上次提示词 | 预填弹窗 |
| `manualModel` | string | 否 | 手动修图上次模型 | 空则使用 provider model |
| `provider` | `ProviderConfig` | 是 | 当前仅 `seed-edit` | 外部 AI 服务配置 |
| `apiKey` | string | 否 | 火山引擎 API Key | 当前明文保存 |
| `model` | string | 是 | 模型 ID | 默认 `doubao-seedream-5-0-260128` |

### 关联关系

- `AiEditConfig` belongs to `AppConfig`。
- `ProviderConfig` currently has one variant `SeedEditConfig`。

### 重要约束

- 自动修图触发要求 `autoEdit=true` 且 prompt 非空。
- SeedEdit API base URL 为 `https://ark.cn-beijing.volces.com/api/v3`。

### 复刻建议

- Web 版已确认不做 AI 修图；该实体只作为原项目事实记录，不进入首期数据库、API 或前端页面。

## 实体：ServerInfo

证据：`src-tauri/src/ftp/types.rs::ServerInfo`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `isRunning` | bool | 是 | 是否运行 | 前端状态 |
| `ip` | string | 是 | 推荐 IPv4 | 相机连接地址 |
| `port` | u16 | 是 | 监听端口 | 相机连接端口 |
| `url` | string | 是 | `ftp://ip:port` | 展示用 |
| `username` | string | 是 | 用户名 | 匿名时为 `anonymous` |
| `passwordInfo` | string | 是 | 密码提示 | 匿名时 `(任意密码)`，认证时 `(配置密码)` |

### 复刻建议

- 该 DTO 是启动成功响应和状态同步的参考对象。Web 版可保留 `isRunning`、`host/ip`、`port`、`url`、`username`、`anonymousEnabled` 等字段，但不需要兼容原 Tauri 的 `passwordInfo` 文案字段。

## 实体：ServerStateSnapshot / ServerRuntimeView

证据：`src-tauri/src/ftp/types.rs`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `isRunning` | bool | 是 | 运行态 | UI 是否显示运行 |
| `connectedClients` | number | 是 | 当前连接客户端数 | 相机连接状态 |
| `filesReceived` | number | 是 | 已接收文件数 | 统计卡片 |
| `bytesReceived` | number | 是 | 已接收字节数 | 统计卡片 |
| `lastFile` | string? | 否 | 最后接收文件 | 最新照片兜底 |

### 复刻建议

- Go 中用事件流维护运行快照，并提供 `/api/ftp/status` 查询。
- 注意停止后的迟到统计不能恢复运行态。

## 实体：FileInfo / FileIndex

证据：`src-tauri/src/file_index/types.rs`、`src-tauri/src/file_index/service.rs`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `path` | Path/String | 是 | 文件绝对路径或 content URI 变体 | 打开/预览目标 |
| `filename` | string | 是 | 文件名 | UI 展示 |
| `exifTime` | SystemTime? | 否 | EXIF 拍摄时间 | Rust 内部，TS skip |
| `modifiedTime` | SystemTime | 是 | 修改时间 | Rust 内部，TS skip |
| `sortTime` | number | 是 | 毫秒时间戳 | 前端排序/显示 |

### 关联关系

- `FileIndex` has many `FileInfo`。
- `FileIndex` has current index。

### 重要约束

- 支持索引扩展名仅 `jpg/jpeg/heif/hif/heic`。
- 排序新到旧。
- current index 可能为空。

### 复刻建议

- Windows 用文件系统扫描 + watcher。
- Android 不要用文件路径递归扫描作为主方案，应以 MediaStore 查询和 content URI 为准。
- 如果新增 RAW 支持，要统一扩展名、MIME、EXIF、图库、预览能力。

## 实体：ExifInfo

证据：`src-tauri/src/commands/exif.rs::ExifInfo`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `iso` | number? | 否 | ISO | 拍摄参数 |
| `aperture` | string? | 否 | `f/2.8` | 拍摄参数 |
| `shutterSpeed` | string? | 否 | `1/125s` | 拍摄参数 |
| `focalLength` | string? | 否 | `24mm` | 拍摄参数 |
| `datetime` | string? | 否 | `YYYY-MM-DD HH:mm:ss` | 拍摄时间 |

### 复刻建议

- Go 可用 EXIF 库解析基础字段，但 HEIF/RAW 支持要单独评估。
- 若使用 Android content URI，需要原生层提供临时可读文件或输入流。

## 实体：MediaPageRequest / MediaPageResponse / MediaItemDto

证据：`src/types/gallery-v2.ts`、`MediaPageProvider.kt`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `cursor` | string? | 否 | 分页游标 | keyset pagination |
| `pageSize` | number | 是 | 页大小 | 前端默认 120 |
| `sort` | `'dateDesc'` | 是 | 排序 | 当前只支持 dateDesc |
| `mediaId` | string | 是 | MediaStore ID | 图库主键 |
| `uri` | string | 是 | content URI | 打开/删除/分享 |
| `dateModifiedMs` | number | 是 | 修改时间毫秒 | 排序游标 |
| `width` / `height` | number? | 否 | 图片尺寸 | UI |
| `mimeType` | string? | 否 | MIME | 判断类型 |
| `displayName` | string? | 否 | 文件名 | UI |
| `nextCursor` | string? | 否 | 下一页游标 | 分页 |
| `revisionToken` | string | 是 | 版本标识 | 当前为 count |
| `totalCount` | number | 是 | 总数 | 图库标题 |

### 重要约束

- selection 固定为 `RELATIVE_PATH LIKE '%DCIM/CameraFTP/%'`。
- sort 固定 `DATE_MODIFIED DESC, _ID DESC`。
- cursor 是 base64 JSON：`{ dateModifiedMs, mediaId }`。

### 复刻建议

- Android 继续以 MediaStore `_ID` 作为稳定键。
- revision token 建议改为更强版本，例如最大 `DATE_MODIFIED` + count + latest id。

## 实体：ThumbRequest / ThumbResult

证据：`src/types/gallery-v2.ts`、`GalleryBridgeV2.kt`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `requestId` | string | 是 | 请求 ID | 回调匹配 |
| `mediaId` | string | 是 | MediaStore ID | 缓存键 |
| `uri` | string | 是 | content URI | 解码输入 |
| `dateModifiedMs` | number | 是 | 修改时间 | 缓存失效键 |
| `sizeBucket` | `s/m` | 是 | 缩略图尺寸 | 缓存分桶 |
| `priority` | `visible/nearby` | 是 | 优先级 | 可见优先 |
| `viewId` | string | 是 | 视图 ID | listener 生命周期 |
| `status` | enum | 是 | `ready/failed/cancelled` | 处理结果 |
| `localPath` | string? | 否 | 本地缓存文件 | 前端显示 |
| `errorCode` | enum? | 否 | 错误码 | 失败原因 |

### 复刻建议

- Go 不适合直接处理 Android content URI 缩略图；Android 原生层继续负责解码和缓存。

## 实体：DeleteImagesResult

证据：`src/types/global.ts`、`GalleryBridge.kt`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `deleted` | string[] | 是 | 已删除 URI | UI 移除 |
| `notFound` | string[] | 是 | 已不存在 URI | UI 也移除 |
| `failed` | string[] | 是 | 删除失败 URI | UI 保留并提示 |

### 复刻建议

- 保留三分类，避免用户重复看到已不存在的媒体项。

## 实体：StorageInfo / PermissionStatus / ServerStartCheckResult

证据：`src-tauri/src/platform/types.rs`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `displayName` | string | 是 | 存储显示名 | UI |
| `path` | string | 是 | 路径 | 诊断/展示 |
| `exists` | bool | 是 | 是否存在 | 启动前检查 |
| `writable` | bool | 是 | 是否可写 | 启动前检查 |
| `hasAllFilesAccess` | bool | 是 | 是否有全部文件访问 | 历史字段，Android 实际用媒体权限 |
| `needsUserAction` | bool | 是 | 是否需要用户操作 | 权限提示 |
| `canStart` | bool | 是 | 能否启动 | 启动门禁 |
| `reason` | string? | 否 | 失败原因 | UI 提示 |
| `storageInfo` | StorageInfo? | 否 | 存储详情 | UI/诊断 |

### 复刻建议

- Android 命名建议改成 media access，避免 `allFilesAccess` 与实际 `READ_MEDIA_IMAGES` 不一致。

## 实体：AiEditProgressEvent

证据：`src-tauri/src/ai_edit/progress.rs`。

### 字段说明

| 事件类型 | 字段 | 说明 |
|---|---|---|
| `queued` | `queueDepth` | 当前队列深度 |
| `queuedDropped` | `fileName`, `queueDepth` | 自动队列满导致丢弃 |
| `progress` | `current`, `total`, `fileName`, `failedCount` | 处理中 |
| `completed` | `current`, `total`, `fileName`, `failedCount`, `outputPath?` | 单文件成功 |
| `failed` | `current`, `total`, `fileName`, `error`, `failedCount` | 单文件失败 |
| `done` | `total`, `failedCount`, `failedFiles`, `outputFiles`, `cancelled` | 批处理结束 |

### 复刻建议

- Web 版已确认不做 AI 修图；该事件只作为原项目事实记录，不进入首期 WebSocket 事件集合。

## 实体：Android MediaStore QueryResult

证据：`src-tauri/src/ftp/android_mediastore/types.rs`、`MediaStoreBridge.kt`。

### 字段说明

| 字段名 | 类型 | 是否必填 | 说明 | 业务含义 |
|---|---|---|---|---|
| `contentUri` / `uri` | string | 是 | MediaStore URI | 文件引用 |
| `displayName` | string | 是 | 文件名 | FTP list/gallery |
| `size` | number | 是 | 文件大小 | FTP metadata |
| `dateModified` | number | 是 | 修改时间毫秒 | FTP metadata/gallery |
| `mimeType` | string | 是 | MIME | 分类 |
| `relativePath` | string | 是 | MediaStore 相对路径 | 虚拟目录 |

### 重要约束

- 图片进入 `MediaStore.Images`。
- 视频进入 `MediaStore.Video`。
- 非媒体进入 `MediaStore.Files`，路径重映射到 `Download/CameraFTP/`。

### 复刻建议

- Android 端设计 `StorageBackend` 接口时保持文件系统和 MediaStore 两套实现。
- 注意 MediaStore pending/finalize/abort 三阶段写入。

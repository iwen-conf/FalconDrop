# 功能点清单

## Web 复刻范围说明

本文件先按原项目真实代码列出功能点。由于复刻目标已明确为“只做 Web”，以下原生能力只作为源码事实和业务参考，不进入 Web MVP：

- Android 权限、MediaStore、图库 Bridge、前台服务、内置图片查看器。
- Windows 托盘、开机自启动、Tauri 二级预览窗口、Shell 打开文件夹。
- Tauri IPC 本身。

根据最新确认范围，Web 版实际保留范围收敛为：FTP 接收、默认 FTP 账号、匿名访问、上传统计、所有格式入库、照片识别、EXIF 时间分类、本地存储、PostgreSQL、WebSocket、照片页和系统信息页。FTPS、AI 修图、RBAC、多用户、对象存储和原生平台能力不进入 Web 版。

## 模块：FTP 服务生命周期

### 功能点 1：启动 FTP/FTPS 服务器

- 功能描述：用户点击启动后，本地启动 FTP/FTPS 服务，选择可用端口，选择推荐 IP，返回连接信息。
- 入口页面/接口：`src/components/ServerCard.tsx`；IPC `invoke('start_server')`；后端 `src-tauri/src/commands/server.rs::start_server`。
- 涉及角色：摄影师/现场操作员、相机 FTP 客户端。
- 输入数据：配置中的 `port`、`autoSelectPort`、`advancedConnection`、保存路径和平台权限。
- 输出数据：`ServerInfo { isRunning, ip, port, url, username, passwordInfo }`。
- 业务规则：
  - 如果服务器已运行，`start_server` 幂等返回当前 `ServerInfo`。
  - 高级连接关闭时使用平台默认端口：Windows `21`，Android `2121`，见 `src-tauri/src/constants.rs`。
  - 如果请求端口不可用且 `auto_select_port=true`，从 `MIN_PORT=1025` 开始寻找可用端口，见 `src-tauri/src/ftp/server_factory.rs`。
  - IP 推荐优先级为 WiFi、以太网、其他，并过滤 loopback、169.254、虚拟网卡，见 `src-tauri/src/network.rs`。
  - Android 启动前前端会检查存储、通知、电池优化权限，见 `src/stores/serverStore.ts`。
- 异常情况：无可用端口、无可用网卡、存储不可写、权限缺失、FTP 绑定失败。
- 复刻优先级：P0。Web 版只复刻 FTP，不复刻 FTPS。

### 功能点 2：停止 FTP/FTPS 服务器

- 功能描述：停止当前本地 FTP 服务，清理状态和运行句柄。
- 入口页面/接口：`ServerCard`；IPC `invoke('stop_server')`；`commands/server.rs::stop_server`。
- 涉及角色：摄影师/现场操作员。
- 输入数据：当前 `FtpServerState`。
- 输出数据：无，前端状态重置。
- 业务规则：
  - 未运行时停止操作幂等成功。
  - 如果 stop 返回错误但运行态已停止，清理 stale handle 并视为成功。
- 异常情况：停止失败且运行态仍为运行中。
- 复刻优先级：P0。Web 版只复刻 FTP，不复刻 FTPS。

### 功能点 3：运行状态同步和传输统计

- 功能描述：前端实时显示服务器运行态、连接数、接收文件数、字节数、最后文件。
- 入口页面/接口：`StatsCard`、`InfoCard`；IPC `get_server_runtime_state`；事件 `server-started`、`server-stopped`、`stats-update`。
- 涉及角色：摄影师/现场操作员。
- 输入数据：FTP Presence/Data 事件。
- 输出数据：`ServerStateSnapshot`。
- 业务规则：
  - `FtpPresenceListener` 统计在线连接数。
  - `FtpDataListener` 在 PUT 时记录上传次数、字节数和文件名。
  - 停止后的延迟 stats 更新会被忽略，见 `src-tauri/src/ftp/types.rs` 测试。
- 异常情况：事件监听初始化失败时前端会尝试从 `get_server_runtime_state` 同步。
- 复刻优先级：P0。Web 版需要通过 WebSocket 推送运行状态。

## 模块：连接配置与认证

### 功能点 1：高级连接设置开关

- 功能描述：用户可启用高级连接设置，配置自定义端口和 FTP 认证。
- 入口页面/接口：`ConfigCard`、`AdvancedConnectionConfigPanel`；IPC `save_config`。
- 涉及角色：摄影师/现场操作员。
- 输入数据：`advancedConnection.enabled`、`port`。
- 输出数据：持久化后的 `AppConfig`。
- 业务规则：
  - 服务器运行中禁用相关配置输入，避免运行态配置漂移。
  - 高级连接关闭时后端忽略用户配置端口和认证，使用平台默认端口和匿名模式。
- 异常情况：端口为空、非法、超范围或被占用。
- 复刻优先级：P0。

### Web 版修正：账号范围

- 只保留一个默认 FTP 账号。
- 支持匿名访问，不限制 IP。
- 不做多 FTP 账号管理页面。

### 功能点 2：端口校验与占用检测

- 功能描述：端口输入失焦后校验格式和范围，并通过后端尝试绑定检测占用。
- 入口页面/接口：`AdvancedConnectionConfig.tsx`；hook `usePortCheck.ts`；IPC `check_port_available`。
- 涉及角色：摄影师/现场操作员。
- 输入数据：端口号。
- 输出数据：`{ available: boolean }`。
- 业务规则：
  - Android 最小端口为 `1024`，Windows 最小端口为 `1`。
  - 通用合法范围为 `1..65535`。
  - 后端检测绑定 `0.0.0.0:port`。
- 异常情况：检测失败视为不可用。
- 复刻优先级：P0。Web 版只做服务器本地数据卷，不做 NAS/S3/MinIO 抽象。

### 功能点 3：FTP 匿名/用户名密码认证

- 功能描述：可允许匿名访问，也可配置用户名和密码；密码由后端 Argon2id 哈希保存。
- 入口页面/接口：`AdvancedConnectionConfig.tsx`；IPC `save_auth_config`；后端 `commands/config.rs::save_auth_config`。
- 涉及角色：摄影师/现场操作员、相机 FTP 客户端。
- 输入数据：`anonymous`、`username`、明文 `password`。
- 输出数据：`AuthConfig { anonymous, username, passwordHash }`。
- 业务规则：
  - 匿名模式或空密码保存为空 hash。
  - 非匿名模式下，如果用户名为空或 hash 为空，`FtpAuthConfig::from` 会降级为匿名访问。
  - 密码使用 Argon2id PHC 格式，见 `src-tauri/src/crypto.rs` 和 `commands/config.rs`。
  - UI 文案提示“用户名或密码未配置，将使用匿名访问模式”。
- 异常情况：密码保存失败、配置重载失败。
- 复刻优先级：P0。Web 版允许所有格式上传，所有资产入库；照片页只展示 `is_photo=true` 的资产，不触发 AI 自动修图。

## 模块：存储路径与权限

### 功能点 1：保存路径管理

- 功能描述：Windows 可选择保存目录；Android 固定为 `/storage/emulated/0/DCIM/CameraFTP`。
- 入口页面/接口：`PathSelector`、`ConfigCard`；IPC `select_save_directory`、`get_storage_info`、`ensure_storage_ready`。
- 涉及角色：Windows 桌面用户、Android 设备用户。
- 输入数据：保存目录选择、平台默认路径。
- 输出数据：`StorageInfo`、保存后的 `AppConfig.savePath`。
- 业务规则：
  - Android `AppConfig::normalized_for_current_platform` 强制 save path 为固定 DCIM 路径。
  - Windows 默认使用系统图片目录，失败则 `./pictures`。
  - 保存路径变更后触发 `FileIndexService.update_save_path` 重新扫描。
- 异常情况：路径不存在、不可写、权限不足。
- 复刻优先级：P0。Web 版照片页按 EXIF 时间分类，缺失 EXIF 时由后端提供兜底时间字段。

### 功能点 2：Android 权限导览

- 功能描述：检查并引导授权媒体读取、通知和电池优化白名单。
- 入口页面/接口：`PermissionDialog`、`PermissionList`、`permissionStore.ts`；JS Bridge `PermissionAndroid`。
- 涉及角色：Android 设备用户。
- 输入数据：Android 运行时权限状态。
- 输出数据：`PermissionCheckResult { storage, notification, batteryOptimization }`。
- 业务规则：
  - 存储权限要求 `READ_MEDIA_IMAGES` 完整授权；Android 14 部分照片访问不满足，会打开应用权限设置。
  - 服务器启动前需要存储、通知、电池优化都通过，见 `src/stores/serverStore.ts`。
  - 前台服务使用 `FOREGROUND_SERVICE_CONNECTED_DEVICE`，并持有 WakeLock/WiFiLock。
- 异常情况：用户拒绝授权、ROM 不支持电池优化白名单、通知权限拒绝。
- 复刻优先级：Out。Web 版不复刻 Android 权限；对应能力改为服务器存储目录和部署健康检查。

## 模块：上传后处理和文件索引

### 功能点 1：FTP 上传后统计、索引和事件

- 功能描述：相机上传文件后记录统计，图片文件进入索引并触发前端/平台后处理。
- 入口页面/接口：FTP PUT；后端 `src-tauri/src/ftp/listeners.rs::FtpDataListener`。
- 涉及角色：相机 FTP 客户端、摄影师/现场操作员。
- 输入数据：上传相对路径、字节数。
- 输出数据：统计快照、`file-uploaded`、`file-index-changed`、`media-library-refresh-requested` 等事件。
- 业务规则：
  - 上传后等待文件就绪，最大 `FILE_READY_TIMEOUT_SECS=5`。
  - 当前文件索引支持扩展名：`jpg`、`jpeg`、`heif`、`hif`、`heic`。
  - 图片上传后触发 AI 自动修图检查。
  - Windows 图片上传后可自动打开预览。
- 异常情况：文件 5 秒内未就绪、索引失败、非图片跳过预览。
- 复刻优先级：P0。

### 功能点 2：目录扫描和最新照片

- 功能描述：启动后扫描保存目录，按 EXIF 拍摄时间或修改时间排序，提供最新照片和导航索引。
- 入口页面/接口：`LatestPhotoCard`；IPC `get_file_list`、`get_latest_image`、`navigate_to_file`。
- 涉及角色：摄影师/现场操作员。
- 输入数据：保存目录文件。
- 输出数据：`FileInfo { path, filename, sortTime }`。
- 业务规则：
  - 排序新到旧。
  - 优先使用 EXIF DateTimeOriginal，否则使用文件修改时间。
  - Windows 使用 `notify` watcher；Android 不使用文件系统 watcher，主要依赖 MediaStore 事件。
- 异常情况：目录读取失败、文件被删除导致导航失败时重新找相邻可用文件。
- 复刻优先级：P0。

## 模块：Windows 自动预览与桌面能力

### 功能点 1：内置预览窗口

- 功能描述：收到新图片后打开或更新 Windows 预览窗口，支持缩放、拖拽、全屏、导航、EXIF 和 AI 修图。
- 入口页面/接口：`PreviewConfigCard`、`PreviewWindow`；IPC `open_preview_window`；事件 `preview-image`。
- 涉及角色：Windows 桌面用户。
- 输入数据：图片路径、`PreviewWindowConfig`。
- 输出数据：Tauri label 为 `preview` 的二级窗口。
- 业务规则：
  - 内置窗口默认尺寸 `1024x768`。
  - 可配置收到新图时是否置顶，置顶持续 `2` 秒。
  - 窗口创建后延迟 `300ms` 发送图片事件。
- 异常情况：图片不存在、窗口事件发送失败、自定义打开失败。
- 复刻优先级：Out。Web 版不复刻 Tauri/Windows 窗口；改为 React 图片预览页或弹窗。

### 功能点 2：系统默认/Windows Photos/自定义程序打开

- 功能描述：用户可选择图片打开方式。
- 入口页面/接口：`PreviewConfigCard`；后端 `auto_open/windows.rs`。
- 涉及角色：Windows 桌面用户。
- 输入数据：`ImageOpenMethod`、自定义 exe 路径。
- 输出数据：外部程序打开图片。
- 业务规则：
  - `built-in-preview` 为推荐项。
  - 自定义程序只在 Windows 选择 `.exe`。
- 异常情况：程序路径为空、ShellExecute/Photos 激活失败。
- 复刻优先级：Out。Web 版不打开服务器本地外部程序。

### 功能点 3：托盘与开机自启动

- 功能描述：Windows 支持系统托盘启动/停止/显示/退出，支持注册表开机自启。
- 入口页面/接口：`ConfigCard` 自启动开关；托盘事件 `tray-start-server`、`tray-stop-server`。
- 涉及角色：Windows 桌面用户。
- 输入数据：自启动开关。
- 输出数据：HKCU Run 项和托盘状态。
- 业务规则：
  - 自启动使用 `--autostart` 参数，启动后隐藏窗口并延迟 `500ms` 启动服务器。
  - 关闭主窗口时弹出“退出程序/最小化到托盘”。
- 异常情况：注册表写入失败、托盘图标更新失败。
- 复刻优先级：Out。Web 版用 systemd/Docker restart policy 管理服务进程。

## 模块：Android 图库和图片查看

### 功能点 1：图库分页浏览

- 功能描述：Android 通过 MediaStore 查询 `DCIM/CameraFTP` 图片，虚拟列表展示缩略图。
- 入口页面/接口：`GalleryCard`；JS Bridge `GalleryAndroidV2.listMediaPage`；Kotlin `MediaPageProvider.kt`。
- 涉及角色：Android 设备用户。
- 输入数据：`MediaPageRequest { cursor, pageSize, sort }`。
- 输出数据：`MediaPageResponse { items, nextCursor, revisionToken, totalCount }`。
- 业务规则：
  - 默认页大小 `120`。
  - 只查询 `MediaStore.Images` 且 `RELATIVE_PATH LIKE '%DCIM/CameraFTP/%'`。
  - 排序为 `DATE_MODIFIED DESC, _ID DESC`。
  - cursor 为 base64 JSON，包含 `dateModifiedMs` 和 `mediaId`。
- 异常情况：无权限、MediaStore 查询失败、cursor 解码失败。
- 复刻优先级：P0/P1。Web 版需要媒体库分页，但数据源改为 PostgreSQL `media_assets`，不是 Android MediaStore。

### 功能点 2：缩略图队列与缓存

- 功能描述：图库按可见区域和附近区域异步生成缩略图，使用内存/磁盘缓存。
- 入口页面/接口：`useThumbnailScheduler`、`GalleryAndroidV2.enqueueThumbnails`、`ThumbnailPipelineManager.kt`。
- 涉及角色：Android 设备用户。
- 输入数据：`ThumbRequest` 列表。
- 输出数据：`ThumbResult` 通过 `window.__galleryThumbDispatch` 回调。
- 业务规则：
  - 支持取消请求、按视图注册 listener、按 mediaId 失效缓存。
  - Kotlin Pipeline 有队列上限和优先级分配，见 `ThumbnailPipelineManager.kt`。
- 异常情况：OOM、解码失败、权限拒绝、任务取消。
- 复刻优先级：P1。Web 版由 Go worker 生成缩略图，不复刻 Android Bridge。

### 功能点 3：图库选择、删除、分享、AI 修图

- 功能描述：长按进入选择模式，批量删除、分享或发起 AI 修图。
- 入口页面/接口：`useGallerySelection.ts`；JS Bridge `GalleryAndroid.deleteImages`、`shareImages`。
- 涉及角色：Android 设备用户。
- 输入数据：选中的 mediaId 和 content URI。
- 输出数据：删除结果、分享 Intent、AI 修图任务。
- 业务规则：
  - 长按阈值 `400ms`，移动超过 `15px` 取消长按。
  - 删除返回 `deleted`、`notFound`、`failed`，删除/不存在都从 UI 移除。
  - Android 删除遇到 `SecurityException` 时使用 `MediaStore.createDeleteRequest` 请求系统确认。
  - 分享支持单张 `ACTION_SEND` 和多张 `ACTION_SEND_MULTIPLE`。
- 异常情况：删除需要系统确认、部分删除失败、无 URI 映射。
- 复刻优先级：P1。Web 版只保留删除；不做 AI 修图和分享链接。

### 功能点 4：Android 内置图片查看器

- 功能描述：可用内置 Activity 打开图片，支持滑动导航、EXIF 回调、AI 修图弹窗和进度同步。
- 入口页面/接口：`image-open.ts`；JS Bridge `ImageViewerAndroid.openOrNavigateTo`；`ImageViewerActivity.kt`。
- 涉及角色：Android 设备用户。
- 输入数据：当前 content URI、全部 URI 列表。
- 输出数据：图片查看 Activity。
- 业务规则：
  - 配置 `openMethod=built-in-viewer` 时优先内置查看器。
  - 内置失败时 fallback 到外部应用 chooser，再 fallback 到 Tauri preview。
  - EXIF 由前端调用 `get_image_exif` 后通过 `onExifResult` 回传原生查看器。
- 异常情况：URI 无法解析、Activity 打开失败、EXIF 解析失败。
- 复刻优先级：Out。Web 版使用浏览器图片预览替代。

## 模块：AI 修图

### 功能点 1：AI 修图配置

- 功能描述：配置火山引擎 API Key、模型、自动修图开关和提示词。
- 入口页面/接口：`AiEditConfigCard`、`AiEditConfigPanel`；配置模型 `src-tauri/src/ai_edit/config.rs`。
- 涉及角色：摄影师/现场操作员。
- 输入数据：API Key、模型 ID、提示词、自动开关。
- 输出数据：`AppConfig.aiEdit`。
- 业务规则：
  - 当前 provider 只有 `seed-edit`。
  - 默认模型为 `doubao-seedream-5-0-260128`。
  - 自动修图开启但提示词为空时不会触发有效自动任务。
- 异常情况：API Key 缺失、保存失败。
- 复刻优先级：Out。Web 版已确认不做 AI 修图。

### 功能点 2：手动/自动 AI 修图队列

- 功能描述：上传后可自动入队，预览或图库可手动入队；向 SeedEdit API 提交图片并保存结果。
- 入口页面/接口：IPC `trigger_ai_edit`、`enqueue_ai_edit`、`cancel_ai_edit`；后端 `AiEditService`。
- 涉及角色：摄影师/现场操作员、AI 服务调用方。
- 输入数据：图片路径、prompt、model。
- 输出数据：`save_path/AIEdit/{stem}_AIEdit_{timestamp}.jpg`。
- 业务规则：
  - 手动队列容量 `4`，自动队列容量 `32`；手动优先。
  - 自动触发条件：`auto_edit=true` 且 prompt 非空。
  - 结果文件名带 UTC 毫秒时间戳，冲突时追加 `_1.._99`。
  - 进度通过 `ai-edit-progress` 事件广播。
- 异常情况：队列满丢弃自动任务、API 超时、下载失败、图片预处理失败、用户取消。
- 复刻优先级：Out。Web 版已确认不做 AI 修图。

## 模块：关于与外部链接

### 功能点 1：关于和捐赠

- 功能描述：显示项目关于信息、捐赠二维码和外部链接。
- 入口页面/接口：`AboutCard`、`WeChatDonateDialog`、`openExternalLink`。
- 涉及角色：所有应用用户。
- 输入数据：URL、静态图片资产。
- 输出数据：外部浏览器或对话框。
- 业务规则：Android 外部链接优先走 `PermissionAndroid.openExternalLink`，否则走 Tauri `open_external_link`。
- 异常情况：浏览器不可用、链接打开失败。
- 复刻优先级：P2。

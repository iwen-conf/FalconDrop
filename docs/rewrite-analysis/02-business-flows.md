# 关键业务流程

## Web 版范围说明

本文件主体记录原 Tauri/Android/Windows 项目的真实业务流程，用于理解来源系统。Web 重写时以 `12-confirmed-web-scope.md` 为准，不复刻 Android 权限、Windows 托盘、Tauri 预览窗口、AI 修图、FTPS、RBAC、多用户和对象存储。

Web 版目标流程收敛为：

1. Docker Compose 启动 Go API、FTP server、React Web、PostgreSQL 和本地数据卷。
2. 用户使用唯一默认系统账号登录。
3. 系统信息页展示系统版本、系统 hash、系统时间、系统账号、FTP 账号、匿名状态、FTP 状态和本地存储状态。
4. 相机通过默认 FTP 账号或匿名模式上传任意格式文件。
5. 后端保存文件到本地数据卷，计算 hash，并按“同名同 hash 覆盖、同名不同 hash 不覆盖”规则入库。
6. 后端识别照片并提取 EXIF 时间；照片页按 EXIF 时间分类展示。
7. WebSocket 推送上传、覆盖、删除和运行状态变化。
8. 用户在照片页删除照片时，直接删除数据库记录和本地文件。

## 业务流程：应用启动初始化

### 参与角色

- 摄影师/现场操作员
- Windows 桌面用户或 Android 设备用户

### 流程步骤

1. Tauri 入口 `src-tauri/src/lib.rs::run` 初始化日志、平台服务、托管状态和插件。
2. Android 平台调用 `config::init_android_paths` 初始化配置路径和默认存储目录。
3. Rust 创建并管理 `FtpServerState`、`ConfigService`、`FileIndexService`、`AutoOpenService`、`AiEditService`。
4. 前端 `src/bootstrap/useAppBootstrap.ts` 加载平台、权限和配置。
5. 前端注册服务器运行事件监听，并调用 `get_server_runtime_state` 同步初始状态。
6. 如果当前 Tauri 窗口 label 为 `preview`，`src/App.tsx` 只渲染 `PreviewWindow`；否则渲染主界面。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| 未初始化 | 应用启动 | UI 配置加载中 | 系统 | `useAppBootstrap` |
| 配置未加载 | `load_config` 成功 | 配置就绪 | 系统 | `ConfigService` 读取 JSON |
| 服务器状态未知 | `get_server_runtime_state` | 运行中或停止 | 系统 | 防止前端刷新后状态丢失 |

### 关键业务规则

- 配置读取失败时后端返回默认配置，见 `commands/config.rs::load_config_from_service_or_default`。
- Android 配置路径在 Tauri app data dir 下，README 标注为 `/data/data/com.gjk.cameraftpcompanion/files/config.json`。
- Windows 日志写入 `dirs::data_dir()/cameraftp/logs/app.log`；Android 日志写入 `/storage/emulated/0/DCIM/CameraFTP/logs/app.log`。

### 异常分支

- 配置 JSON 无效会回退默认值，存在用户配置被“静默重置”的风险。
- 事件监听初始化失败时仅记录 warning，前端仍显示但可能不能实时更新。

## 业务流程：启动服务器

### 参与角色

- 摄影师/现场操作员
- Android 设备用户
- 相机 FTP 客户端

### 流程步骤

1. 用户点击 `ServerCard` 的“启动服务器”。
2. 前端调用 `permissionStore.checkPrerequisites` 和 Android JS Bridge 权限检查。
3. 如果权限缺失，显示 `PermissionDialog` 或打开系统授权页面。
4. 权限满足后调用 `serverStore.startServer`。
5. 前端 IPC 调用 `start_server`。
6. 后端读取 `AppConfig`，调用平台 `ensure_storage_ready`。
7. 后端确定端口：高级连接启用时使用配置端口，否则使用平台默认端口；端口不可用时按配置自动查找备用端口。
8. 后端选择推荐 IPv4 地址，创建 `ServerConfig` 和 FTP server actor。
9. 后端启动事件处理器，连接 file index 的 event bus。
10. 前端收到 `ServerInfo` 或 `server-started` 事件，更新连接信息和统计卡片。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| Stopped | 点击启动 | Starting | 用户 | UI 显示启动中 |
| Starting | 权限缺失 | WaitingPermission | 系统 | Android 需要用户授权 |
| Starting | 绑定端口成功 | Running | Rust 后端 | `start_server_with_event_pipeline` |
| Starting | 端口/网卡/存储失败 | Stopped | Rust 后端 | 返回错误给前端 |
| Running | 再次调用启动 | Running | 用户/托盘 | 幂等返回当前 `ServerInfo` |

### 关键业务规则

- Windows 默认端口是 `21`，Android 默认端口是 `2121`。
- 自动选端口从 `1025` 起扫。
- FTP 服务监听 `0.0.0.0:port`，展示推荐局域网 IPv4。
- 匿名认证是默认模式；非匿名必须同时具备用户名和密码 hash。
- PASV 端口范围使用 libunftp 默认范围，当前没有独立 UI 配置。

### 异常分支

- Android 后端 `check_server_start_prerequisites` 当前主要返回平台层检查，真正的通知/电池权限门禁在前端 JS Bridge，复刻时不要只依赖后端。
- Windows 使用 21 端口可能需要管理员权限或被防火墙阻止。
- FTPS 使用自签名证书，部分相机或客户端可能需要额外信任设置。

## 业务流程：FTP 上传文件

### 参与角色

- 相机 FTP 客户端
- 摄影师/现场操作员
- AI 服务调用方

### 流程步骤

1. 相机 FTP 客户端连接应用展示的 `ftp://ip:port`。
2. FTP 认证器验证匿名或用户名密码。
3. 客户端执行 PUT 上传文件。
4. `FtpDataListener` 记录上传统计并发出 `FileUploaded` 领域事件。
5. 如果扩展名是当前索引支持的图片，后端等待文件写入就绪。
6. 文件就绪后加入 `FileIndexService` 索引。
7. 后端触发 AI 自动修图检查。
8. Windows 平台根据预览配置自动打开或更新预览窗口。
9. Android MediaStore 写入完成后通过 Kotlin 触发 `gallery-items-added`，图库增量插入新图片。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| 已连接 | PUT 开始 | 接收中 | 相机 | libunftp 处理数据流 |
| 接收中 | PUT 完成 | 已接收 | 相机 | stats 增加 |
| 已接收 | 文件就绪 | 已索引 | 系统 | 图片扩展名受限 |
| 已索引 | 自动修图开启 | AI 队列中 | 系统 | prompt 不能为空 |
| 已索引 | Windows 自动预览开启 | 预览已更新 | 系统 | 仅 Windows |

### 关键业务规则

- 文件就绪等待最大 5 秒。
- `FileIndexService::is_supported_image` 只识别 `jpg/jpeg/heif/hif/heic`。
- Android MediaStore 后端会把非媒体文件映射到 `Download/CameraFTP/`。
- Android MediaStore 上传不支持 resume、mkdir、rmdir、rename，见 `src-tauri/src/ftp/android_mediastore/backend.rs`。
- Android MediaStore 后端防御路径穿越和 null byte。

### 异常分支

- README 提到 RAW 支持，但当前文件索引不索引 RAW；EXIF 和 MediaStore MIME 映射包含 RAW，存在实现不一致。
- Android 图库只查 `MediaStore.Images`，视频、RAW、非媒体不会出现在图库页。
- 文件就绪超时后不会进入索引和自动预览。

## 业务流程：停止服务器

### 参与角色

- 摄影师/现场操作员
- Windows 托盘用户

### 流程步骤

1. 用户在主界面或托盘点击停止。
2. 前端调用 `stop_server`。
3. 后端找到当前 `FtpServerHandle` 并调用 stop。
4. 服务停止后清空 `FtpServerState`。
5. 运行态发出 `server-stopped`，前端重置连接信息和统计。
6. Android 通过 JNI 同步停止前台服务。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| Running | 点击停止 | Stopping | 用户 |
| Stopping | 停止成功 | Stopped | Rust 后端 |
| Stopped | 再次停止 | Stopped | 用户/托盘 | 幂等成功 |

### 关键业务规则

- stop 是幂等命令。
- 延迟到达的 stats 不能重新把前端置为运行中。

### 异常分支

- 如果 stop 错误但 runtime snapshot 已停止，后端清理 stale handle 并返回成功。

## 业务流程：配置保存与认证保存

### 参与角色

- 摄影师/现场操作员

### 流程步骤

1. 用户在配置页修改保存路径、端口、预览、Android 查看器或 AI 设置。
2. `configStore.updateDraft` 更新前端 draft。
3. 100ms debounce 后调用 `save_config`。
4. 后端持久化 JSON 配置。
5. 如果保存路径变化，触发文件索引重新扫描。
6. 用户修改密码时，前端调用 `save_auth_config`，后端哈希明文密码后保存，并重新 `load_config` 同步。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| ConfigLoaded | 修改普通配置 | DraftDirty | 用户 |
| DraftDirty | debounce flush | ConfigSaved | 前端 |
| ConfigSaved | 保存路径变化 | Reindexing | Rust 后端 |
| AuthEditing | 密码失焦 | AuthHashing | 用户 |
| AuthHashing | 保存成功 | ConfigReloaded | Rust 后端 |

### 关键业务规则

- `save_config` 会保留后端拥有的 `preview_config` 字段，避免全量保存覆盖 Windows 预览状态，见 `merge_backend_owned_fields`。
- 认证单独保存是为了后端哈希密码，前端不直接写 hash。
- 前端有写队列，保证配置写入串行化。

### 异常分支

- `AppConfig::validate` 当前只校验端口非 0 和路径非空；非匿名用户名/密码校验主要在 UI 和 `FtpAuthConfig::from` 降级逻辑中。
- AI API Key 明文保存在配置 JSON。

## 业务流程：Android 图库浏览、删除与分享

### 参与角色

- Android 设备用户

### 流程步骤

1. 用户切换到底部导航的“图库”。
2. `GalleryCard` 检查 `GalleryAndroidV2` 是否可用。
3. `useGalleryPager` 调用 `listMediaPage({ cursor:null, pageSize:120, sort:'dateDesc' })` 加载第一页。
4. `VirtualGalleryGrid` 根据可见范围通知 `useThumbnailScheduler`。
5. 前端向 `GalleryAndroidV2.enqueueThumbnails` 批量请求缩略图。
6. Kotlin 生成缩略图并通过 `window.__galleryThumbDispatch` 回调结果。
7. 用户长按 400ms 进入选择模式。
8. 删除时前端把 mediaId 映射为 content URI，调用 `GalleryAndroid.deleteImages`。
9. 分享时调用 `GalleryAndroid.shareImages` 打开系统分享。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| Idle | 打开图库 | LoadingPage | 用户 |
| LoadingPage | MediaStore 返回 | Browsing | Kotlin Bridge |
| Browsing | 长按图片 | Selecting | 用户 |
| Selecting | 删除 | Deleting | 用户 |
| Deleting | 删除成功/不存在 | Browsing | Kotlin Bridge | UI 移除 |
| Selecting | 取消/切换 Tab | Browsing | 用户 |

### 关键业务规则

- 仅 Android 显示图库；非 Android `GalleryCard` 返回 null。
- 删除结果中 `notFound` 也会触发 UI 移除动画。
- 如果 Android 删除需要确认，Kotlin 使用 `MediaStore.createDeleteRequest`。
- 图库增量事件包括 `gallery-items-added` 和 `gallery-items-deleted`。

### 异常分支

- 没有 `READ_MEDIA_IMAGES` 权限时，图库为空或加载失败，前端会引导权限。
- `revisionToken` 当前只是 `count:<count>`，同数量替换文件可能无法被识别。

## 业务流程：图片预览与 EXIF

### 参与角色

- Windows 桌面用户
- Android 设备用户

### 流程步骤

1. 用户点击“最新照片”或图库缩略图。
2. Android 若配置为内置查看器，前端调用 `ImageViewerAndroid.openOrNavigateTo`。
3. 内置查看器打开成功后，前端调用 `get_image_exif` 并通过 `onExifResult` 回传 EXIF。
4. Android 内置查看器失败时，尝试外部应用 chooser。
5. 如果没有 Android Bridge 或外部打开失败，前端 fallback 到 Tauri `open_preview_window`。
6. Windows 内置预览窗口通过 `PreviewWindow` 展示图片，支持导航、缩放、全屏、打开文件夹和 AI 修图。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| NoPreview | 打开图片 | PreviewOpen | 用户 |
| PreviewOpen | 下一张/上一张 | PreviewNavigating | 用户 |
| PreviewNavigating | 导航成功 | PreviewOpen | 后端 |
| PreviewOpen | 图片不存在 | PreviewError | 系统 |
| PreviewOpen | ESC | FullscreenOff 或关闭弹窗 | 用户 |

### 关键业务规则

- 预览导航依赖 `FileIndexService` 的当前索引。
- 导航目标文件不存在时，前端重新拉取列表并从目标位置向前后查找可打开文件。
- EXIF 字段包括 ISO、光圈、快门、焦距、拍摄时间。

### 异常分支

- Android content URI 需要 `ImageViewerAndroid.resolveFilePath` 转为真实路径后才能给 Rust EXIF 解析。
- `convertFileSrc` 依赖 Tauri asset protocol；当前 `tauri.conf.json` asset scope 允许 `**`，安全范围较宽。

## 业务流程：AI 修图

### 参与角色

- 摄影师/现场操作员
- AI 服务调用方

### 流程步骤

1. 用户配置火山引擎 API Key、模型和提示词。
2. 手动修图：用户从预览窗口或 Android 图库选择图片，输入 prompt 后调用 `enqueue_ai_edit`。
3. 自动修图：FTP 图片上传后 `AiEditService.on_file_uploaded` 根据配置决定是否入自动队列。
4. 后端预处理图片：Windows/非 Android 用 Rust image/heic，Android 用 `ImageProcessorBridge`。
5. SeedEdit provider 调用 `https://ark.cn-beijing.volces.com/api/v3/images/generations`。
6. 后端下载或解析结果并保存到 `save_path/AIEdit`。
7. 进度通过 `ai-edit-progress` 事件传给前端和 Android 图片查看器。
8. 用户可取消任务，后端取消 token 并发出 done cancelled。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| Idle | 入队 | Queued | 用户/系统 |
| Queued | Worker 取任务 | Processing | 后端 |
| Processing | 单文件成功 | Completed | AI 服务 |
| Processing | 单文件失败 | Failed | AI 服务/后端 |
| Processing | 用户取消 | Done(cancelled) | 用户 |
| Queued | 自动队列满 | QueuedDropped | 后端 | 仅自动任务 |
| Completed/Failed | 批处理结束 | Done | 后端 |

### 关键业务规则

- 手动队列容量 4，自动队列容量 32；手动任务优先。
- SeedEdit 请求体包含 `model`、`prompt`、`image`、`size:"4K"`、`response_format:"url"`、`watermark:false`。
- API 超时为 180 秒。
- 输出文件名规则为 `{stem}_AIEdit_{yyyyMMdd_HHmmss.SSS}.jpg`，冲突时最多尝试 `_1.._99`。

### 异常分支

- API Key 缺失或无效。
- 自动任务队列满会丢弃并发出 `queuedDropped`。
- Android 新生成文件需要 `scanNewFile` 或 MediaStore 刷新后才会出现在图库。

## 业务流程：Windows 开机自启动

### 参与角色

- Windows 桌面用户

### 流程步骤

1. 用户在配置页打开“开机自启动”。
2. 前端调用 `set_autostart_command`。
3. Windows 平台层写入 HKCU `Software\Microsoft\Windows\CurrentVersion\Run`。
4. 系统下次启动时带 `--autostart` 参数运行应用。
5. `lib.rs::run` 检测 autostart mode，隐藏窗口。
6. 平台层延迟 `AUTOSTART_DELAY_MS=500` 自动启动服务器。

### 状态流转

| 当前状态 | 操作 | 下一个状态 | 触发角色 | 备注 |
|---|---|---|---|---|
| AutostartOff | 开关打开 | AutostartOn | 用户 |
| AutostartOn | 系统登录 | AppHidden | Windows |
| AppHidden | 延迟启动服务器 | Running | 应用 |

### 关键业务规则

- 自启动只在 Windows 显示。
- 主窗口关闭不是立即退出，会触发“退出程序/最小化到托盘”选择。

### 异常分支

- 注册表权限或路径异常导致自启动写入失败。
- 自启动模式下端口/权限/网络失败时服务不会成功运行，需日志排查。

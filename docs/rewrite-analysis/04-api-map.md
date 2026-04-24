# API 接口清单

## 总体说明

当前项目没有 REST API、GraphQL、SSE 或传统 WebSocket 服务。前后端交互主要由三类接口组成：

- Tauri IPC commands：React `invoke()` 调 Rust 命令。
- Tauri events：Rust `emit()` 推送事件到前端。
- Android JS Bridge：Kotlin 通过 WebView `addJavascriptInterface` 注入对象给前端调用，或通过 `evaluateJavascript` 调前端全局函数。

Web 复刻时，不保留 Tauri IPC 和 Android JS Bridge。应把可迁移的业务能力映射为 Go REST API，把实时事件映射为 WebSocket；Android/Windows 原生命令不进入 Web API。

## Tauri IPC Commands

证据：`src-tauri/src/lib.rs` 的 `tauri::generate_handler!`。

| 方法 | 路径/命令 | 功能 | 请求参数 | 响应数据 | 权限要求 | 备注 |
|---|---|---|---|---|---|---|
| IPC | `start_server` | 启动 FTP 服务 | 无 | `ServerInfo` | Android 需权限；本地应用权限 | 幂等，运行中返回当前信息 |
| IPC | `stop_server` | 停止 FTP 服务 | 无 | `void` | 本地应用权限 | 幂等 |
| IPC | `get_server_runtime_state` | 获取运行状态 | 无 | `ServerRuntimeView` | 无 | 前端启动时同步 |
| IPC | `load_config` | 加载配置 | 无 | `AppConfig` | 本地配置读权限 | 失败返回默认配置 |
| IPC | `save_config` | 保存配置 | `config: AppConfig` | `void` | 本地配置写权限 | 保存路径变化触发重扫 |
| IPC | `save_auth_config` | 保存认证配置 | `anonymous, username, password` | `void` | 本地配置写权限 | 后端 Argon2id hash 明文密码 |
| IPC | `select_save_directory` | 选择保存目录 | 无 | `string?` | Windows 文件对话框 | Android 平台返回平台结果 |
| IPC | `check_port_available` | 检测端口可用性 | `port: u16` | `boolean` | 绑定端口权限 | 绑定 `0.0.0.0:port` |
| IPC | `get_platform` | 获取平台名 | 无 | `string` | 无 | `windows` / `android` 等 |
| IPC | `set_autostart_command` | 设置 Windows 开机自启 | `enable: boolean` | `void` | Windows 注册表写权限 | HKCU Run |
| IPC | `get_autostart_status` | 查询自启状态 | 无 | `boolean` | Windows 注册表读权限 | 配置页初始化 |
| IPC | `quit_application` | 退出应用 | 无 | `void` | 本地应用权限 | `app.exit(0)` |
| IPC | `hide_main_window` | 隐藏主窗口 | 无 | `void` | 桌面窗口权限 | 托盘最小化 |
| IPC | `show_main_window` | 显示主窗口 | 无 | `void` | 桌面窗口权限 | 托盘显示 |
| IPC | `get_storage_info` | 获取存储信息 | 无 | `StorageInfo` | 平台存储权限 | 配置页/权限页 |
| IPC | `check_permission_status` | 检查平台权限 | 无 | `PermissionStatus` | 平台权限 | Android 主要权限由 JS Bridge 检查 |
| IPC | `request_all_files_permission` | 请求存储权限 | 无 | `void` | Android/平台能力 | Android emit 设置事件，实际前端用 Bridge |
| IPC | `ensure_storage_ready` | 确保存储可用 | 无 | `string` | 文件写权限 | 返回可用路径 |
| IPC | `check_server_start_prerequisites` | 检查启动条件 | 无 | `ServerStartCheckResult` | 平台权限 | 启动前调用 |
| IPC | `update_preview_config` | 更新预览配置 | `patch` | `PreviewWindowConfig` | Windows | 局部更新并广播 `preview-config-changed` |
| IPC | `open_preview_window` | 打开/更新预览 | `filePath: string` | `void` | 文件读取/窗口权限 | Windows 主要实现 |
| IPC | `select_executable_file` | 选择自定义程序 | 无 | `string?` | Windows 文件对话框 | 仅 `.exe` |
| IPC | `open_folder_select_file` | 打开文件夹并选中文件 | `filePath: string` | `void` | Windows shell 权限 | Android 空实现 |
| IPC | `open_external_link` | 打开外部链接 | `url: string` | `void` | Shell/浏览器 | Android 主要走 JS Bridge |
| IPC | `get_file_list` | 获取文件索引 | 无 | `FileInfo[]` | 文件读取权限 | 返回 Arc<Vec> 序列化 |
| IPC | `get_current_file_index` | 获取当前索引 | 无 | `number?` | 无 | 预览导航 |
| IPC | `navigate_to_file` | 导航到指定索引 | `index: number` | `FileInfo` | 文件存在性 | 文件不存在时返回错误并清理 |
| IPC | `get_latest_image` | 获取最新图片 | 无 | `FileInfo?` | 文件索引 | Android 有 GalleryV2 时前端不用此命令 |
| IPC | `get_image_exif` | 解析 EXIF | `filePath: string` | `ExifInfo?` | 文件读取权限 | 支持 JPG/PNG/HEIF/RAW parser |
| IPC | `trigger_ai_edit` | 同步触发单张 AI 修图 | `filePath, prompt?, model?` | `string` 输出路径 | API Key/文件读写 | 等待结果 |
| IPC | `enqueue_ai_edit` | 批量入队 AI 修图 | `filePaths, prompt?, model?` | `void` | API Key/文件读写 | 非阻塞 |
| IPC | `cancel_ai_edit` | 取消 AI 修图 | 无 | `void` | 无 | drain/cancel token |

## Android JS Bridge

证据：`src/types/global.ts`、`MainActivity.kt` 注册 `PermissionAndroid`、`GalleryAndroid`、`GalleryAndroidV2`、`ImageViewerAndroid`。

### PermissionAndroid

| 方法 | 功能 | 请求参数 | 响应数据 | 权限要求 | 备注 |
|---|---|---|---|---|---|
| `checkAllPermissions` | 检查存储、通知、电池优化 | 无 | JSON string `{storage, notification, batteryOptimization}` | Android 权限 API | 前端启动服务器前调用 |
| `requestStoragePermission` | 请求媒体读取权限 | 无 | void | Android runtime permission | 部分照片访问会打开应用设置 |
| `requestNotificationPermission` | 请求通知权限 | 无 | void | `POST_NOTIFICATIONS` | Android 13+ |
| `requestBatteryOptimization` | 请求电池优化白名单 | 无 | void | 系统设置 | 打开系统授权页 |
| `openExternalLink` | 打开外部链接 | `url` | void | 浏览器 Intent | 关于/文档链接 |
| `openImageWithChooser` | 用外部应用打开图片 | `path` content URI 或路径 | JSON string `{success,message}` | URI 授权 | 支持 chooser |

### GalleryAndroid

| 方法 | 功能 | 请求参数 | 响应数据 | 权限要求 | 备注 |
|---|---|---|---|---|---|
| `deleteImages` | 删除 MediaStore 图片 | JSON string URI array | JSON string `DeleteImagesResult` | 媒体删除权限 | `SecurityException` 时系统确认 |
| `shareImages` | 分享图片 | JSON string URI array | boolean | URI read grant | 单张/多张 Intent |
| `registerBackPressCallback` | 选择模式拦截返回键 | 无 | boolean | 无 | 前端设置 `window.__galleryOnBackPressed` |
| `unregisterBackPressCallback` | 取消返回键拦截 | 无 | boolean | 无 | 退出选择模式 |

### GalleryAndroidV2

| 方法 | 功能 | 请求参数 | 响应数据 | 权限要求 | 备注 |
|---|---|---|---|---|---|
| `listMediaPage` | 分页查询 MediaStore 图片 | JSON `MediaPageRequest` | JSON `MediaPageResponse` | `READ_MEDIA_IMAGES` | 只查 DCIM/CameraFTP |
| `enqueueThumbnails` | 缩略图入队 | JSON `ThumbRequest[]` | void/空 JSON | 图片读取权限 | 结果异步回调 |
| `cancelThumbnailRequests` | 取消缩略图请求 | JSON requestId array | void | 无 | 滚动时取消 |
| `registerThumbnailListener` | 注册缩略图 listener | `viewId, listenerId` | void | 无 | 结果通过全局函数回调 |
| `unregisterThumbnailListener` | 注销 listener | `listenerId` | void | 无 | 生命周期清理 |
| `invalidateMediaIds` | 失效缩略图缓存 | JSON mediaId array | void | 无 | 删除后调用 |

### ImageViewerAndroid

| 方法 | 功能 | 请求参数 | 响应数据 | 权限要求 | 备注 |
|---|---|---|---|---|---|
| `openOrNavigateTo` | 打开或复用内置查看器 | `uri, allUrisJson` | boolean | Android Activity | 内置 viewer |
| `isAppVisible` | 查看器/应用是否可见 | 无 | boolean | 无 | 自动打开最新照片 |
| `onExifResult` | 接收前端 EXIF 结果 | `exifJson?` | void | 无 | Kotlin viewer 回调 |
| `resolveFilePath` | URI 转真实路径 | `uri` | string? | ContentResolver | 给 Rust EXIF/AI 用 |
| `onAiEditComplete` | AI 修图完成通知 | `success,message,cancelled` | void | 无 | 原生 UI |
| `updateAiEditProgress` | AI 修图进度 | `current,total,failedCount` | void | 无 | 原生 UI |
| `scanNewFile` | 扫描新文件 | `filePath` | void | MediaScanner | AI 结果入图库 |

## 前端全局函数供 Android 调用

证据：`src/App.tsx`、`ImageViewerActivity.kt`。

| 函数 | 调用方 | 功能 | 请求参数 | 响应 |
|---|---|---|---|---|
| `window.__tauriGetAiEditPrompt` | `ImageViewerActivity` | 获取当前 AI 修图 prompt/model/API Key 状态 | 无 | JSON string |
| `window.__tauriTriggerAiEditWithPrompt` | `ImageViewerActivity` | 从原生查看器触发 AI 修图并可保存配置 | `filePath,prompt,model?,saveAsAutoEdit?,apiKey?` | Promise<void> |
| `window.__tauriGetAiEditProgress` | `ImageViewerActivity` | 获取当前 AI 进度 | 无 | progress object |
| `window.__tauriCancelAiEdit` | `ImageViewerActivity` | 取消 AI 修图 | 无 | Promise<void> |
| `window.__galleryThumbDispatch` | `GalleryBridgeV2` | 缩略图结果派发 | `listenerId,resultJson` | void |
| `window.__galleryOnBackPressed` | `MainActivity` | Android 返回键处理 | 无 | void |

## 事件清单

| 事件名 | 方向 | Payload | 功能 | 证据 |
|---|---|---|---|---|
| `server-started` | Rust -> 前端 | `{ ip, port }` 或运行态同步 | 服务器已启动 | `ftp/events.rs`、`server-events.ts` |
| `server-stopped` | Rust -> 前端 | void | 服务器已停止 | `server-events.ts` |
| `stats-update` | Rust -> 前端 | `ServerStateSnapshot` | 传输统计更新 | `server-events.ts` |
| `file-uploaded` | Rust -> 前端 | `{ path, size }` | 文件上传瞬时事件 | `ftp/events.rs` |
| `file-index-changed` | Rust -> 前端 | `{ count, latestFilename }` | 文件索引变化 | `usePreviewNavigation.ts` |
| `media-library-refresh-requested` | Rust -> 前端 | void | Android 图库刷新兜底 | `server-events.ts` |
| `tray-start-server` | Windows -> 前端 | void | 托盘启动服务器 | `platform/windows.rs` |
| `tray-stop-server` | Windows -> 前端 | void | 托盘停止服务器 | `platform/windows.rs` |
| `window-close-requested` | Rust -> 前端 | void | Windows 关闭主窗口选择退出/托盘 | `lib.rs`、`useQuitFlow.ts` |
| `preview-image` | Rust -> 预览窗口 | `{ filePath, bringToFront }` | 更新预览图片 | `auto_open/service.rs` |
| `preview-config-changed` | Rust -> 前端 | `{ config }` | 同步预览配置 | `auto_open/service.rs` |
| `ai-edit-progress` | Rust -> 前端 | `AiEditProgressEvent` | AI 修图进度 | `ai_edit/progress.rs` |
| `gallery-items-added` | Android -> 前端 DOM | `{ items, timestamp }` | 图库增量新增 | `GalleryCard.tsx`、`MediaStoreBridge.kt` |
| `gallery-items-deleted` | Android -> 前端 DOM | `{ mediaIds, timestamp }` | 图库增量删除 | `GalleryCard.tsx` |

## Go + React API 映射建议

| 原 IPC/Bridge | Go + React 建议接口 | 说明 |
|---|---|---|
| `start_server` | `POST /api/ftp/start` | 返回 FTP 运行状态和连接信息 |
| `stop_server` | `POST /api/ftp/stop` | 幂等 |
| `get_server_runtime_state` | `GET /api/ftp/status` | 初始同步 |
| `load_config` / `save_config` | `GET /api/system/info` / `PUT /api/ftp/account` | Web 版配置进入 PostgreSQL |
| `save_auth_config` | `PUT /api/ftp/account` | 更新唯一 FTP 默认账号，后端 hash |
| `check_port_available` | `GET /api/network/ports/{port}/available` | 或 query 参数 |
| `get_file_list` / `navigate_to_file` | `GET /api/photos` / `GET /api/photos/{id}` | 照片列表和详情 |
| `get_image_exif` | `GET /api/photos/{id}` | EXIF 字段随照片详情返回 |
| Tauri events | `GET /api/ws` | WebSocket 统一事件通道 |
| Android Bridge | 不映射 | Web 版不复刻 Android 原生能力 |

## Web 版新增 API

| 方法 | 路径 | 功能 | 说明 |
|---|---|---|---|
| `POST` | `/api/auth/login` | Web 登录 | 唯一默认系统账号 |
| `POST` | `/api/auth/logout` | 退出登录 | 清理会话 |
| `GET` | `/api/auth/me` | 当前账号 | 路由守卫 |
| `GET` | `/api/photos` | 照片分页/分组 | 按 EXIF 时间分类展示 |
| `GET` | `/api/photos/{id}` | 照片详情 | 包含原文件名、hash、EXIF 时间 |
| `GET` | `/api/photos/{id}/content` | 原图访问 | 替代 `convertFileSrc`/content URI |
| `DELETE` | `/api/photos/{id}` | 删除照片 | 直接删除数据库记录和本地文件 |
| `GET` | `/api/assets` | 全部上传资产 | 可包含非照片格式 |
| `GET` | `/api/system/info` | 系统信息 | 版本、hash、时间、系统账号、FTP 账号 |
| `GET` | `/api/ftp/status` | FTP 状态 | 连接信息和运行态 |
| `POST` | `/api/ftp/start` | 启动 FTP | 返回连接信息 |
| `POST` | `/api/ftp/stop` | 停止 FTP | 幂等 |
| `PUT` | `/api/ftp/account` | 更新 FTP 账号 | 唯一默认账号和匿名策略 |
| `PUT` | `/api/system/account` | 更新系统账号 | 唯一默认账号 |
| `GET` | `/api/ws` | WebSocket | 上传、覆盖、删除和状态事件 |

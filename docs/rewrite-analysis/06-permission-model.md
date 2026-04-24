# 权限模型分析

## 认证方式

当前项目没有应用登录、用户账号、角色表、Session、Cookie、JWT 或菜单权限。权限模型由三部分组成：

1. FTP 连接认证：匿名或用户名密码，用于相机 FTP 客户端连接。
2. 本机/系统权限：Windows 文件系统、注册表、托盘、Shell；Android 媒体读取、通知、前台服务、电池优化、WakeLock/WiFiLock。
3. Tauri capability：控制 WebView 可调用哪些窗口/API 能力。

## 角色模型

| 角色 | 说明 | 可以访问的功能 | 关键权限 | 备注 |
|---|---|---|---|---|
| 应用本机用户 | 打开应用的人 | 全部 UI 功能 | 操作系统本机权限 | 无账号隔离 |
| 相机 FTP 客户端 | 通过 FTP 连接的设备 | 上传/下载/删除/目录操作，受平台 StorageBackend 支持限制 | FTP 匿名或用户名密码 | 非 UI 用户 |
| Windows 平台服务 | 应用内部平台层 | 托盘、自启、预览、打开文件夹 | 注册表、Shell、文件系统 | 代码 `platform/windows.rs` |
| Android 平台服务 | 应用内部平台层 | MediaStore、前台服务、权限、图片查看 | Android runtime permissions | Kotlin Bridge |
| AI 服务 | 外部 SeedEdit API | 图片生成/编辑 | API Key | 不代表应用用户 |

## 权限粒度

以下表格描述的是原项目当前实现；Web 版目标权限模型见后文“Go + React Web 重写建议”。

| 粒度 | 当前实现 | 说明 |
|---|---|---|
| 页面级 | 无 RBAC；按平台条件显示 | Gallery 仅 Android；PreviewConfig 仅 Windows |
| 接口级 | Tauri 本地 IPC；无用户鉴权 | 桌面/移动本地应用假设 WebView 可信 |
| 按钮级 | 基于运行态/平台/权限禁用 | 服务器运行中禁用端口/路径等配置 |
| 数据级 | 无多租户；本机路径/MediaStore 范围 | Android 图库只查 `DCIM/CameraFTP` |
| FTP 连接级 | 匿名或用户名密码 | Argon2id 校验 |

## FTP 认证模型

证据：`src-tauri/src/config.rs::AuthConfig`、`src-tauri/src/ftp/types.rs::FtpAuthConfig`、`src-tauri/src/crypto.rs`。

### 当前实现

- 默认匿名访问：`AuthConfig::default()` 中 `anonymous=true`。
- 高级连接关闭时，运行时总是使用匿名认证。
- 高级连接开启且非匿名时，必须有 `username` 和 `passwordHash` 才会转换为 `Authenticated`。
- 密码保存通过 `save_auth_config`，后端使用 Argon2id 哈希。
- 展示层不会显示认证密码明文；匿名模式展示“任意用户名/密码”。

### 风险

- 非匿名配置不完整时静默降级匿名，可能违背用户预期。
- FTP 匿名默认开启，适合局域网便利性，但安全性较弱。
- FTPS 自签名证书可能导致客户端信任问题。

## Android 权限模型

证据：`AndroidManifest.xml`、`PermissionBridge.kt`、`FtpForegroundService.kt`。

### Manifest 权限

- 网络：`INTERNET`、`ACCESS_NETWORK_STATE`。
- 存储/媒体：`READ_EXTERNAL_STORAGE`、`WRITE_EXTERNAL_STORAGE`（maxSdk 32）、`READ_MEDIA_IMAGES`、`READ_MEDIA_VISUAL_USER_SELECTED`。
- 前台服务：`FOREGROUND_SERVICE`、`FOREGROUND_SERVICE_CONNECTED_DEVICE`。
- 网络状态/WiFi：`CHANGE_WIFI_STATE`、`CHANGE_NETWORK_STATE`。
- 通知：`POST_NOTIFICATIONS`。
- 保活：`WAKE_LOCK`、`REQUEST_IGNORE_BATTERY_OPTIMIZATIONS`。

### 当前实现方式

- `PermissionBridge.checkAllPermissions` 返回 `{ storage, notification, batteryOptimization }`。
- 存储权限要求 `READ_MEDIA_IMAGES` 完整授权。
- 如果只有 Android 14 partial photo access，`requestStoragePermission` 打开应用权限设置。
- 启动 FTP 前前端 `serverStore.startServer` 要求三项都通过，否则弹出权限对话框。
- 运行时 `FtpForegroundService` 使用前台通知、WakeLock 和 WiFiLock。

### 数据权限

- 图库查询限制在 `MediaStore.Images` 且路径匹配 `DCIM/CameraFTP`。
- 删除/分享以 content URI 为单位，通过系统权限和 Intent grant 控制。

### 风险

- Rust 后端 `check_server_start_prerequisites` 对 Android 通知/电池权限没有完整强制，真实门禁在前端 JS Bridge。
- Android 13/14 媒体权限复杂，复刻时不能用旧的“所有文件访问权限”思路替代。

## Windows 权限模型

证据：`platform/windows.rs`、`auto_open/windows.rs`、`commands/config.rs`。

### 当前实现方式

- 保存路径通过文件夹选择器获得，应用直接读写。
- 开机自启写 HKCU Run：`Software\Microsoft\Windows\CurrentVersion\Run`。
- 托盘操作通过 Tauri tray icon。
- 打开外部程序和文件夹通过 Windows Shell API。
- 端口 21 可能需要管理员权限或防火墙放行，但代码主要通过端口绑定结果反馈。

### 风险

- 默认端口 21 对普通用户不友好，可能启动失败或被安全软件拦截。
- 自定义程序路径执行存在安全边界，必须只由本机用户选择，不应被远程输入控制。

## Tauri Capability / WebView 权限

证据：`src-tauri/tauri.conf.json`、`src-tauri/capabilities/default.json`。

### 当前实现方式

- `tauri.conf.json` 中 `csp: null`。
- `assetProtocol` 启用且 `allow: ["**"]`。
- Tauri capabilities 允许窗口 fullscreen、always on top 等能力。

### 风险

- CSP 为空和 asset protocol 全路径 allow 扩大 WebView 攻击面。
- 本地应用假设前端可信；如果未来改为远程 Web UI，需要重新设计鉴权和路径访问控制。

## Go + React Web 重写建议

### 后端建议

- Web 版必须新增登录认证，建议使用 HttpOnly Cookie session 或短期 JWT + refresh token。
- Web 版只需要一个默认系统账号，不做 RBAC、多用户和角色区分。
- FTP 只需要一个默认账号，并支持匿名访问；匿名访问不限制 IP。
- 非匿名配置不完整时拒绝保存或拒绝启动。
- Web 用户密码和 FTP 密码都使用 Argon2id/bcrypt 等安全哈希。
- 文件访问必须通过媒体 ID 和权限校验，不允许浏览器传任意服务器路径。
- 删除照片时直接删除数据库记录和本地文件。

### 前端建议

- 使用路由守卫控制页面访问。
- 登录后拥有全部系统操作权限。
- 对危险配置给出明确状态：匿名 FTP 已开启、PASV 端口未开放、本地存储不可写。

### 数据权限建议

- MVP 是全局照片库，唯一系统账号访问同一批数据。
- 媒体访问接口按 media id 授权，不暴露服务器真实路径。

### 权限测试建议

- 未登录访问 API。
- 登录后访问照片和系统信息接口。
- FTP 匿名开关关闭后匿名连接。
- FTP 密码错误、账号禁用。
- 直接猜测媒体文件 URL。

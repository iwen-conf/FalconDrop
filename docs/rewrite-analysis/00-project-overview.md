# 项目概览

## 分析边界

- 分析对象：当前仓库真实代码，根目录为 `/Users/iluwen/Documents/Code/Lazycat/Projects/CameraFTP`。
- 主要证据：`README.md`、`package.json`、`src-tauri/Cargo.toml`、`src-tauri/src/lib.rs`、`src/App.tsx`、`src-tauri/gen/android/app/src/main/AndroidManifest.xml`、Android Kotlin Bridge、前端组件和 Store。
- 本项目不是传统 Web 前后端分离系统，也没有独立 HTTP 后端和数据库。它是一个本地跨平台应用：React 负责 UI，Rust/Tauri 负责本地 IPC、FTP 服务、文件索引和平台能力，Android 通过 Kotlin Bridge 补齐 MediaStore、权限、前台服务和图片查看器。

## 根目录结构

| 路径 | 作用 | 证据 |
|---|---|---|
| `README.md` | 产品说明、特性、配置路径、源码结构、构建说明 | `README.md` |
| `package.json` / `bun.lock` | 前端依赖和脚本，React 18、Vite、Tailwind、Zustand、Vitest | `package.json` |
| `src/` | React/TypeScript 前端，单页多 Tab UI | `src/App.tsx`、`src/components/`、`src/stores/` |
| `src-tauri/` | Tauri v2 Rust 后端、Android 生成工程、应用配置 | `src-tauri/src/lib.rs`、`src-tauri/tauri.conf.json` |
| `src-tauri/src/commands/` | Tauri IPC 命令层 | `src-tauri/src/commands/mod.rs` |
| `src-tauri/src/ftp/` | FTP/FTPS 服务、事件、统计、Android MediaStore 存储后端 | `src-tauri/src/ftp/server_factory.rs`、`src-tauri/src/ftp/listeners.rs` |
| `src-tauri/src/file_index/` | 图片索引、扫描、Windows 文件监听 | `src-tauri/src/file_index/service.rs` |
| `src-tauri/src/auto_open/` | Windows 自动预览、系统默认/照片应用/自定义程序打开 | `src-tauri/src/auto_open/service.rs` |
| `src-tauri/src/ai_edit/` | 火山引擎 SeedEdit AI 修图、队列、图片预处理、进度事件 | `src-tauri/src/ai_edit/` |
| `src-tauri/gen/android/` | Android 原生工程，Kotlin Bridge、MediaStore、前台服务、图片查看 Activity | `src-tauri/gen/android/app/src/main/java/com/gjk/cameraftpcompanion/` |
| `scripts/` / `build.sh` | 统一构建脚本、Windows/Android/frontend 构建 | `build.sh`、`scripts/build-windows.sh`、`scripts/build-android.sh` |
| `docs/superpowers/` | 未来特性设计文档，不属于已实现主功能 | `docs/superpowers/specs/` |
| `poc/` | 实验性代码，非主流程 | `poc/` |

## 依赖文件与运行资产

| 类型 | 文件 | 结论 |
|---|---|---|
| 前端依赖 | `package.json`、`bun.lock` | 使用 Vite + React + TypeScript + TailwindCSS + Zustand，测试用 Vitest |
| Rust 依赖 | `src-tauri/Cargo.toml`、`src-tauri/Cargo.lock` | Tauri v2、libunftp、tokio、nom-exif、argon2、rcgen、reqwest、heic、jni、Windows API |
| Tauri 配置 | `src-tauri/tauri.conf.json` | 应用名 `图传伴侣`，identifier `com.gjk.cameraftpcompanion`，版本 `1.6.1`，启用 `protocol-asset` |
| Android Manifest | `src-tauri/gen/android/app/src/main/AndroidManifest.xml` | 声明网络、媒体读取、前台服务、通知、WakeLock、电池优化等权限 |
| Docker/CI | 未发现 | 无 Dockerfile、docker-compose、CI 配置 |
| 环境变量示例 | 未发现 | 无 `.env.example` |
| 数据库迁移 | 未发现 | 无 DB、ORM、migration |
| Go 后端 | 未发现 | 当前项目没有 `go.mod` |

## 技术栈判断

### 前端

- React 18 + TypeScript，入口 `src/main.tsx`，主应用 `src/App.tsx`。
- Vite 构建，TailwindCSS 实用类样式，组件在 `src/components/`。
- Zustand 管理本地 UI 和 IPC 状态：`src/stores/configStore.ts`、`src/stores/serverStore.ts`、`src/stores/permissionStore.ts`。
- Tauri IPC 调用通过 `@tauri-apps/api/core` 的 `invoke()`；事件监听通过 `@tauri-apps/api/event`。
- Android 原生能力以 `window.PermissionAndroid`、`window.GalleryAndroid`、`window.GalleryAndroidV2`、`window.ImageViewerAndroid` 注入，类型定义在 `src/types/global.ts`。

### 本地后端与平台层

- Rust 2021 + Tauri v2，入口 `src-tauri/src/lib.rs`。
- FTP/FTPS 服务基于 `libunftp`，配置和状态由 Rust 管理，前端通过 IPC 控制。
- 文件索引为内存索引，无数据库；Windows 使用文件系统路径，Android 使用 MediaStore。
- Android 平台能力由 Kotlin 实现：权限、图库分页、缩略图队列、MediaStore 文件写入、前台服务、图片查看器。
- AI 修图调用火山引擎 Ark SeedEdit API，代码位于 `src-tauri/src/ai_edit/providers/seededit.rs`。

## 单体/分层方式

这是“本地单体应用 + 平台适配层”，不是浏览器访问的前后端分离系统。

- UI 层：`src/App.tsx`、`src/components/`、`src/hooks/`。
- 状态/交互层：`src/stores/`、`src/services/`。
- IPC 命令层：`src-tauri/src/commands/`。
- 领域服务层：`src-tauri/src/ftp/`、`src-tauri/src/file_index/`、`src-tauri/src/ai_edit/`、`src-tauri/src/auto_open/`。
- 平台层：`src-tauri/src/platform/`、`src-tauri/gen/android/app/src/main/java/`。
- 配置层：`src-tauri/src/config.rs`、`src-tauri/src/config_service.rs`。

## 项目定位

CameraFTP（图传伴侣）是一个给相机照片传输使用的本地 FTP/FTPS companion 应用。它在 Windows 或 Android 设备上启动 FTP 服务，让相机通过同一局域网把照片上传到应用指定目录，并提供连接信息、传输统计、最新照片预览、图库浏览、Android 前台保活和 AI 修图。

## 面向用户

| 角色 | 说明 | 可以访问的功能 | 关键权限 | 备注 |
|---|---|---|---|---|
| 摄影师/现场操作员 | 主使用者，在拍摄现场接收相机照片 | 启停 FTP、查看连接信息、预览最新照片、图库、AI 修图、配置 | 本机应用使用权、Android 媒体/通知/电池权限 | 代码无登录用户系统，此角色是业务使用者 |
| Windows 桌面用户 | 在 PC 上运行接收服务 | 自定义保存目录、开机自启、托盘、自动预览、打开文件夹 | Windows 文件系统、注册表自启动、可能需要防火墙/端口权限 | 证据：`src-tauri/src/platform/windows.rs` |
| Android 设备用户 | 在手机/平板上运行接收服务 | 固定 DCIM 存储、图库、删除/分享、内置查看器、前台服务保活 | `READ_MEDIA_IMAGES`、通知、电池优化白名单、WakeLock/WiFiLock | 证据：`PermissionBridge.kt`、`FtpForegroundService.kt` |
| 相机 FTP 客户端 | 第三方设备，通过 FTP/FTPS 上传文件 | 连接 FTP/FTPS、上传/删除/目录操作，取决于平台后端支持 | FTP 匿名或用户名密码认证 | 不是应用登录用户 |
| AI 服务调用方 | 应用内部调用火山引擎 SeedEdit | 图片预处理、上传、下载 AI 结果 | 火山引擎 API Key | 证据：`src-tauri/src/ai_edit/` |

## 解决的业务问题

- 相机和电脑/手机之间快速传图，不依赖数据线或厂商软件。
- 接收端提供清晰的 FTP 连接信息，降低相机配置难度。
- 在传输过程中自动统计、索引和预览最新照片。
- Android 端通过前台服务、通知、WakeLock/WiFiLock 增强长时间接收稳定性。
- 支持接收后自动或手动调用生成式 AI 修图。

## 核心业务价值

- 把本机设备变成面向相机的 FTP/FTPS 接收端。
- 让现场用户能立即看到“是否连接、收了多少、最新照片是什么”。
- 在 Android 上让照片进入系统 MediaStore，能被系统图库和分享流程识别。
- 通过本地配置和平台能力减少外部部署成本。

## 最重要的业务模块

1. FTP/FTPS 服务启停、端口、IP、认证和传输统计：`src-tauri/src/ftp/`、`src/components/ServerCard.tsx`、`src/components/InfoCard.tsx`。
2. 跨平台存储和权限：`src-tauri/src/platform/`、`PermissionBridge.kt`、`src/stores/permissionStore.ts`。
3. 文件索引、最新照片和预览：`src-tauri/src/file_index/`、`src/components/LatestPhotoCard.tsx`、`src/components/PreviewWindow.tsx`。
4. Android 图库和 MediaStore：`GalleryBridgeV2.kt`、`MediaPageProvider.kt`、`MediaStoreBridge.kt`。
5. AI 修图队列与外部模型调用：`src-tauri/src/ai_edit/`、`src/components/AiEditConfigCard.tsx`。

## Web 版 MVP 复刻范围

已确认的 Web 版 MVP 以 `12-confirmed-web-scope.md` 为准，应包含：

- Docker Compose 正式部署。
- Go 服务端 FTP 服务启动/停止，浏览器端展示 host、port、默认 FTP 账号、匿名状态和连接说明。
- Web 登录认证，只提供一个默认系统账号，不做多用户和角色区分。
- FTP 只提供一个默认账号，并支持匿名访问；匿名不限制 IP。
- 端口配置、PASV 端口范围、端口占用检测和 Docker 端口映射提示。
- 服务端本地数据卷存储，不做 NAS/S3/MinIO 抽象。
- 允许所有格式上传；照片页面只展示可识别为照片的上传资产。
- 上传后写入 PostgreSQL，记录媒体资产、hash、传输事件和系统配置。
- React Web 只提供两个主页面：照片、系统信息。
- 实时更新使用 WebSocket。

首期不做：

- RBAC、多用户、用户管理、审计日志。
- AI 修图手动/自动队列。
- FTPS 证书配置。
- 对象存储、分享链接、多租户。
- Android/Windows/Tauri 原生能力。

## 版本与文档差异

- `package.json`、`src-tauri/Cargo.toml`、`src-tauri/tauri.conf.json` 都显示版本 `1.6.1`。
- `README.md` 中存在旧版本徽章或说明与代码版本不一致的风险，需要复刻时以 manifest 为准。

## Web 复刻结论

只做 Web 时，不再复刻 Android/Windows 原生能力。推荐把原项目迁移为“Go FTP/媒体服务端 + React Web”：Go 负责 FTP、默认账号认证、匿名策略、媒体入库、WebSocket 实时事件和本地文件存储；React 负责登录、照片页和系统信息页；PostgreSQL 负责持久化。

# 配置与外部依赖

## 配置文件

| 类型 | 名称 | 用途 | 配置位置 | 是否复刻必须 | 备注 |
|---|---|---|---|---|---|
| 应用配置 | `config.json` | 保存路径、端口、认证、预览、Android 查看器、AI 修图配置 | Windows `%APPDATA%\cameraftp\config.json`；Android `/data/data/com.gjk.cameraftpcompanion/files/config.json` | 否 | 原项目证据；Web 版配置进入 PostgreSQL |
| Tauri 配置 | `tauri.conf.json` | 应用 identifier、窗口、构建、asset protocol | `src-tauri/tauri.conf.json` | Web 版不需要 | 仅作为原项目证据 |
| Android Manifest | `AndroidManifest.xml` | Android 权限、Activity、Service、Provider | `src-tauri/gen/android/app/src/main/AndroidManifest.xml` | Web 版不需要 | 仅作为原项目证据 |
| ProGuard | `proguard-rules.pro` | 防止 JNI 字符串引用类被 R8 删除 | `src-tauri/gen/android/app/proguard-rules.pro` | Web 版不需要 | 仅作为原项目证据 |
| TS 绑定 | `src-tauri/bindings/` | Rust `ts-rs` 导出的 TS 类型 | `src-tauri/bindings/` | 建议 | 当前部分绑定文件可能缺失或未生成 |

## 环境变量与构建变量

| 类型 | 名称 | 用途 | 配置位置 | 是否复刻必须 | 备注 |
|---|---|---|---|---|---|
| 开发 | `TAURI_DEV_HOST` | Tauri/Vite dev host | Tauri/Vite 环境 | 否 | 当前 `tauri.conf.json` `devUrl: null` |
| Android | `JAVA_HOME` | JDK 路径 | shell 环境 | Android 构建必须 | 脚本自动探测 JDK 17/21 |
| Android | `ANDROID_HOME` / `ANDROID_SDK_ROOT` | Android SDK 路径 | shell 环境 | Android 构建必须 | `scripts/build-android.sh` |
| Android | `NDK_HOME` | NDK 路径 | shell 环境 | Android 构建必须 | 自动从 SDK 检测 |
| Android | `GRADLE_OPTS` | Gradle 并行构建 | `scripts/build-android.sh` | 否 | 设置为 `-Dorg.gradle.parallel=true` |
| 签名 | `KEYSTORE_ALIAS` | Android keystore alias | shell 环境 | Release 建议 | 默认 `cameraftp` |
| 签名 | `KEYSTORE_PASSWORD` | Android keystore 密码 | shell 环境 | Release 建议 | 默认 `cameraftp123`，有风险 |
| 签名 | `KEY_PASSWORD` | Android key 密码 | shell 环境 | Release 建议 | 默认等于 store password |
| 签名 | `KEYSTORE_DNAME` | Android key DN | shell 环境 | 否 | 默认开发 DN |
| 输出 | `DEPLOY_PATH` | APK 复制目标 | shell 环境 | 否 | 默认 `/mnt/ext1/shared-files/nginx` |

## 外部服务与系统依赖

| 类型 | 名称 | 用途 | 配置位置 | 是否复刻必须 | 备注 |
|---|---|---|---|---|---|
| FTP 服务 | `libunftp` | 本地 FTP/FTPS server | `src-tauri/Cargo.toml` | 是，或 Go 等价库 | Web 版只需要 FTP |
| FTPS 证书 | `rcgen` 自签名证书 | 生成 FTP TLS 证书 | `src-tauri/src/crypto/tls.rs` | 否 | 已确认 Web 版不做 FTPS |
| 密码哈希 | `argon2` + `zeroize` | FTP 密码 hash 和明文清理 | `src-tauri/Cargo.toml` | 是 | 非匿名认证必须 |
| EXIF | `nom-exif` | 图片拍摄参数和排序时间 | `src-tauri/src/commands/exif.rs`、`file_index/service.rs` | 是 | Web 版照片页按 EXIF 时间分类 |
| AI 服务 | 火山引擎 Ark SeedEdit | AI 修图 | `src-tauri/src/ai_edit/providers/seededit.rs` | 否 | 已确认 Web 版不做 AI 修图 |
| HTTP Client | `reqwest` | 调用 AI API 和下载结果 | `src-tauri/Cargo.toml` | 否 | Web 版不需要 AI 调用 |
| Android MediaStore | Android 系统 `MediaStore` | 保存/查询/删除/分享图片 | Kotlin Bridge | Web 版不需要 | 改为服务器媒体库 |
| Android 前台服务 | `FtpForegroundService` | 保活、通知、WakeLock/WiFiLock | Kotlin | Web 版不需要 | 改为服务进程/容器管理 |
| Windows Shell | ShellExecute、Photos Activation、Explorer select | 打开图片/文件夹 | `auto_open/windows.rs` | Web 版不需要 | 浏览器预览/下载替代 |
| Windows Registry | HKCU Run | 开机自启动 | `platform/windows.rs` | Web 版不需要 | systemd/Docker restart policy 替代 |
| 文件监听 | `notify` | Windows 保存目录监听 | `src-tauri/Cargo.toml` | Web 版可选 | Web 版通常由上传事件入库 |
| 网络接口 | `local-ip-address` + Windows IP Helper | 推荐局域网 IP | `network.rs` | P0 | 过滤虚拟网卡 |

## 前端依赖

| 类型 | 名称 | 用途 | 是否复刻必须 | 备注 |
|---|---|---|---|---|
| UI 框架 | React 18 | 主 UI | 是 | 可继续使用 |
| 构建 | Vite 5 | 前端构建 | 是 | 可继续使用 |
| 类型 | TypeScript 5 | 类型安全 | 是 | 当前严格 TS |
| 样式 | TailwindCSS 3 | UI 样式 | 是或替代 | 当前大量 utility classes |
| 状态 | Zustand 5 | config/server/permission 状态 | 可保留 | 简洁适合本地 app |
| 图标 | lucide-react | UI icons | 可保留 | 已广泛使用 |
| 通知 | sonner | toast | 可保留 | 错误提示 |
| 测试 | Vitest + Testing Library | 前端单测 | 建议保留 | 当前已有多处 characterization tests |

## Rust 后端依赖

| 类型 | 名称 | 用途 | Go 复刻替代 |
|---|---|---|---|
| Runtime | tokio | async runtime | Go goroutine/context |
| 本地壳 | tauri | IPC、窗口、托盘、平台集成 | Web 版不需要，改为 Gin/Echo REST + WebSocket |
| FTP | libunftp/unftp-sbe-fs | FTP server/storage backend | `github.com/fclairamb/ftpserverlib` 等，需验证 |
| 并发 Map | dashmap | session/状态 | sync.Map/map+mutex |
| 类型生成 | ts-rs | Rust -> TS | OpenAPI/ogen/oapi-codegen 或手写 shared types |
| EXIF | nom-exif | EXIF/RAW parser | Go EXIF 库 + HEIF/RAW 单独方案 |
| 图像 | image/heic | 预处理 | Go image + 外部 HEIF 支持或平台原生 |
| HTTP | reqwest | AI API | Web 版不需要 AI provider 调用 |
| 证书 | rcgen | FTPS 自签名 | Web 版不做 FTPS |

## Android 原生依赖

| 类型 | 名称 | 用途 | 是否复刻必须 |
|---|---|---|---|
| Kotlin Bridge | `PermissionBridge` | 权限检查和授权跳转 | Android 必须 |
| Kotlin Bridge | `GalleryBridge` | 删除/分享 | Android P1 |
| Kotlin Bridge | `GalleryBridgeV2` | 分页和缩略图队列 | Android P1 |
| Kotlin Bridge | `MediaStoreBridge` | Rust JNI 写入 MediaStore | Android P0 |
| Kotlin Activity | `ImageViewerActivity` | 内置图片查看器 | Android P1 |
| Kotlin Service | `FtpForegroundService` | 前台保活 | Android P0 |
| Coordinator | `AndroidServiceStateCoordinator` | Rust 与 Service 状态同步 | Android P0 |

## 缺失项

| 项目 | 状态 | 影响 |
|---|---|---|
| `.env.example` | 未发现 | 新团队不易了解构建变量 |
| Dockerfile/docker-compose | 未发现 | 本地应用可接受；如改 Web 服务需补 |
| CI 配置 | 未发现 | 发布质量依赖人工构建 |
| 数据库 schema/migration | 不存在 | 复刻时不要误建复杂 DB |
| OpenAPI 文档 | 不存在 | IPC 契约靠代码和 ts-rs |
| 完整生成 TS bindings | 当前 `src/types/index.ts` 引用多类绑定，`src-tauri/bindings/` 可能不完整 | 需要运行 `./build.sh gen-types` 或修复生成流程 |

## 安全配置注意事项

- `tauri.conf.json` 中 `csp: null`，WebView 安全策略过宽。
- `assetProtocol.scope.allow=["**"]`，本地文件访问范围过宽。
- Android release keystore 如果环境变量缺失，会用默认密码 `cameraftp123` 生成。
- AI API Key 明文保存。
- FTP 默认匿名模式，适合局域网便利性但不是安全默认。

## 复刻建议

- 为 Go + React Web 定义 `.env.example`，列出 `DATABASE_URL`、`STORAGE_ROOT`、`FTP_HOST`、`FTP_PORT`、`FTP_PASSIVE_PORTS`、`SESSION_SECRET`、默认系统账号初始化变量、默认 FTP 账号初始化变量等。
- 引入 OpenAPI 3.0 描述 REST API 和事件 payload。
- Web 版配置进入 PostgreSQL `app_settings`，不要继续依赖本地单机 `config.json` 作为唯一配置源。
- 新增 Dockerfile、docker-compose、数据库 migration 和 CI 配置。

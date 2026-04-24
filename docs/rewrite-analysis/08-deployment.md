# 运行与部署分析

## 本地开发启动方式

### 前端

证据：`package.json`。

- `dev`: `vite`
- `build`: `tsc && vite build`
- `test`: `vitest run`
- `tauri`: `tauri`

注意：项目根目录的 `AGENTS.md` 明确要求不要直接使用 `bun` 或 `cargo.exe build`，验证代码改动应运行 `./build.sh windows android`。本分析任务只改文档，未运行构建。

### 后端

当前没有独立后端服务。Rust 后端是 Tauri 应用的一部分，随桌面/Android 应用启动。

主要入口：

- Rust library 入口：`src-tauri/src/lib.rs::run`
- Rust binary 入口：`src-tauri/src/main.rs`
- Tauri IPC command 注册：`src-tauri/src/lib.rs`

### 数据库

无数据库、无 migrations、无 ORM。

持久化数据：

- 应用配置 JSON。
- Windows 保存目录内图片文件。
- Android MediaStore 中的图片/视频/非媒体记录。
- AI 结果保存到 `save_path/AIEdit`。
- 日志文件。

### 依赖服务

- 本地网络和端口。
- Android 系统 MediaStore、通知、前台服务。
- Windows Shell、注册表、托盘。
- 火山引擎 SeedEdit API（AI 功能需要公网和 API Key）。

## 当前构建方式

### 统一入口

证据：`build.sh`。

推荐命令：

```bash
./build.sh windows android
```

行为：

- 解析目标 `windows`、`android`、`frontend`、`gen-types`。
- 构建前生成 TS bindings。
- 先构建前端，再并行构建 Windows 和 Android。
- 输出到 `out/`。

### Windows 构建

证据：`scripts/build-windows.sh`。

- 必须找到 `cargo.exe`。
- 构建 target：`x86_64-pc-windows-msvc`。
- 输出：
  - Release：`out/CameraFTP_v{VERSION}.exe`
  - Debug：`out/CameraFTP_v{VERSION}-debug.exe`
- 构建前会尝试终止正在运行的 `cameraftp.exe` 或目标 exe。

注意：

- `build-windows.sh` 实际通过 `get_tool_cmd "cargo"` 优先选择 `cargo.exe`。
- 根 `AGENTS.md` 要求始终使用 `cargo.exe` 构建 Windows artifact。

### Android 构建

证据：`scripts/build-android.sh`。

- 构建目标仅 `arm64-v8a` / `aarch64`。
- 自动探测或要求：
  - `JAVA_HOME`
  - `ANDROID_HOME` / `ANDROID_SDK_ROOT`
  - `NDK_HOME`
  - `keytool`
- release/debug 都输出 APK：
  - Release：`out/CameraFTP_v{VERSION}.apk`
  - Debug：`out/CameraFTP_v{VERSION}-debug.apk`
- 默认会复制到 `DEPLOY_PATH`，默认 `/mnt/ext1/shared-files/nginx`。

风险：

- 如果没有 `keystore.properties`，脚本会自动生成 keystore。
- 默认密码是 `cameraftp123`，release 发布风险较高。

### 生产部署方式

当前项目不是服务器部署，而是构建分发本地应用：

- Windows：单个 exe artifact。
- Android：APK。
- 无 Docker、无后端服务、无数据库部署、无反向代理。

## 日志和运行文件

| 平台 | 路径 | 说明 |
|---|---|---|
| Windows | `dirs::data_dir()/cameraftp/logs/app.log` | Rust 日志 |
| Android | `/storage/emulated/0/DCIM/CameraFTP/logs/app.log` | Rust 日志，依赖外部存储可写 |
| Windows config | `%APPDATA%\cameraftp\config.json` | README 标注 |
| Android config | `/data/data/com.gjk.cameraftpcompanion/files/config.json` | README 标注 |
| Android media | `/storage/emulated/0/DCIM/CameraFTP` | 固定存储目录 |

## Go + React Web 重写后的推荐部署方式

### 推荐形态

用户已明确只做 Web，且 Docker Compose 是正式部署方式。因此推荐形态是：Go 服务端 + React Web + PostgreSQL + 本地数据卷。原项目中的 Android/Windows 原生能力作为源码参考，不进入 Web 版交付范围。

### 后端

- Go API 服务提供 REST 和 WebSocket。
- Go FTP 服务监听相机可访问的服务器地址和端口。
- 支持配置 PASV 端口范围，便于 Docker/NAT/防火墙部署。
- 后端目录建议见 `09-go-react-rewrite-plan.md`。
- API 使用默认系统账号登录认证，不做 RBAC。

### 前端

- React + TypeScript + Vite 构建静态资源。
- 通过 Nginx 或 Go embed 静态资源部署。
- 只需要照片和系统信息两个主页面，路由方案保持简单即可。

### 数据库

- 推荐 PostgreSQL。
- 核心持久化：默认系统账号、默认 FTP 账号、系统配置、媒体资产、传输事件。

### 文件存储

- 服务器本地数据卷。
- 浏览器通过受控 HTTP 接口访问媒体文件。

### 反向代理

- 推荐 Nginx/Caddy/Traefik。
- Web HTTPS、API 反代、静态资源缓存。
- FTP 控制端口和 PASV 端口不通过普通 HTTP 反代，需要在 Docker 端口映射、防火墙和运行环境中单独开放。

### CI/CD

建议新增：

- 前端：typecheck、unit tests、build。
- Go：`go test ./...`、`go vet ./...`、lint。
- 后端镜像构建和数据库迁移测试。
- Docker Compose 正式部署 smoke test。
- FTP 上传集成测试，覆盖 PASV 端口映射。

## 复刻部署路线图

1. 第一阶段：Go API + FTP server + React Web + PostgreSQL + Docker Compose，完成上传入库、照片展示、系统信息和 WebSocket 实时刷新。
2. 第二阶段：强化 EXIF 时间解析、照片时间分组、hash 去重、PASV 部署检查和本地数据卷可靠性。
3. 第三阶段：发布自动化、监控告警、备份恢复和安全加固。

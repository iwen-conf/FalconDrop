# 页面与路由分析

## 总体结论

原项目前端没有使用 React Router 或 TanStack Router，而是通过内部 tab 切换 Home、Gallery、Config。Web 重写后不需要复刻这些原生平台页面，也不需要标准后台管理台的多页面结构。

已确认的 Web 版只需要两个主页面：

- 照片
- 系统信息

## 页面总表

| 页面 | 路由建议 | 所属模块 | 主要功能 | 复刻优先级 |
|---|---|---|---|---|
| 照片 | `/photos` 或默认首页 `/` | 媒体库 | 展示上传的所有照片，按 EXIF 时间分类查看，预览和删除 | P0 |
| 系统信息 | `/system` | 系统状态 | 展示系统版本、系统 hash、系统时间、系统账号、FTP 账号、FTP 运行状态 | P0 |
| 登录 | `/login` | 认证 | 默认系统账号登录 | P0 |

不做 Config、AI Prompt、用户管理、审计日志、Android 权限、Windows 预览窗口等页面。

## 页面：照片

- 页面用途：作为主工作页面，展示 FTP 上传后识别为照片的资产。
- 可访问角色：已登录的默认系统账号。
- 页面主要区域：
  - 时间分组导航或分组标题。
  - 照片网格/列表。
  - 照片预览。
  - 删除操作。
- 页面字段：
  - 原文件名。
  - EXIF 拍摄时间。
  - 上传时间。
  - 文件大小。
  - 文件 hash。
- 页面操作：
  - 按 EXIF 时间分组浏览。
  - 点击照片预览。
  - 删除照片，直接删除数据库记录和本地文件。
  - WebSocket 收到新增/删除事件后刷新当前列表。
- 调用接口：
  - `GET /api/photos`
  - `GET /api/photos/{id}`
  - `GET /api/photos/{id}/content`
  - `DELETE /api/photos/{id}`
  - `GET /api/ws`
- 状态变化：
  - `photo-added` 插入对应 EXIF 时间分组。
  - `photo-deleted` 从当前列表移除。
  - `asset-overwritten` 更新同 hash 覆盖记录。
- 复刻注意事项：
  - 上传允许所有格式，但照片页只展示可识别为照片的资产。
  - 排序和分组优先使用 EXIF 时间；无 EXIF 时间的照片需要后端给出兜底时间字段。
  - 同名不同 hash 不覆盖，UI 需要能展示同名照片。

## 页面：系统信息

- 页面用途：展示系统层面的运行信息和账号信息。
- 可访问角色：已登录的默认系统账号。
- 页面主要区域：
  - 系统版本与系统 hash。
  - 系统时间。
  - 系统账号。
  - FTP 账号。
  - FTP 匿名状态。
  - FTP 运行状态和连接信息。
  - 存储路径和数据卷状态。
- 页面字段：
  - `version`
  - `buildHash`
  - `systemTime`
  - `systemAccount.username`
  - `ftpAccount.username`
  - `ftpAccount.anonymousEnabled`
  - `ftp.host`
  - `ftp.port`
  - `ftp.passivePorts`
  - `storage.root`
  - `storage.writable`
- 页面操作：
  - 查看连接信息。
  - 可选：启动/停止 FTP。
  - 可选：更新默认系统账号密码。
  - 可选：更新默认 FTP 账号密码和匿名策略。
- 调用接口：
  - `GET /api/system/info`
  - `GET /api/ftp/status`
  - `POST /api/ftp/start`
  - `POST /api/ftp/stop`
  - `PUT /api/system/account`
  - `PUT /api/ftp/account`
- 复刻注意事项：
  - 不做 RBAC，多用户或角色页面。
  - 匿名 FTP 不限制 IP，页面必须清晰展示当前匿名状态。
  - FTPS 不在范围内，不需要证书配置项。

## 页面：登录

- 页面用途：默认系统账号登录。
- 可访问角色：未登录用户。
- 页面字段：
  - 用户名。
  - 密码。
- 页面操作：
  - 登录。
  - 退出后回到登录页。
- 调用接口：
  - `POST /api/auth/login`
  - `POST /api/auth/logout`
  - `GET /api/auth/me`

## 导航模型

| 导航项 | 显示条件 | 目标 |
|---|---|---|
| 照片 | 已登录 | `/photos` 或 `/` |
| 系统信息 | 已登录 | `/system` |

## 前端实现建议

- 页面展示文案除专业术语外一律使用中文。
- 专业术语允许保留英文，例如 `Docker Compose`、`PostgreSQL`、`WebSocket`、`FTP`、`EXIF`、`hash`、`API`。
- 可以使用 React Router，也可以用简单 tab 状态；页面数量少，不需要复杂路由架构。
- 前端模块建议只拆 `features/auth`、`features/photos`、`features/system-info`。
- WebSocket 只负责事件增量，页面初次加载必须先调用 REST 快照接口。
- 不引入 AI、用户管理、审计、对象存储配置等前端模块。

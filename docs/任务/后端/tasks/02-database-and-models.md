# 02 数据库与模型计划

## 依据文档

- `docs/rewrite-analysis/03-data-model.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

建立 Web MVP 的 PostgreSQL schema，覆盖唯一系统账号、唯一 FTP 账号、应用配置、媒体资产和传输事件。

## 核心表

### `system_accounts`

| 字段 | 说明 |
|---|---|
| `id` | 主键，建议固定单行 |
| `username` | 默认系统账号用户名 |
| `password_hash` | 密码 hash |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

### `ftp_account`

| 字段 | 说明 |
|---|---|
| `id` | 主键，建议固定单行 |
| `username` | 默认 FTP 用户名 |
| `password_hash` | FTP 密码 hash |
| `anonymous_enabled` | 是否允许匿名访问 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

### `app_settings`

| 字段 | 说明 |
|---|---|
| `key` | 配置键 |
| `value_json` | JSON 配置值 |
| `updated_at` | 更新时间 |

建议配置键：

- `ftp.port`
- `ftp.passive_ports`
- `ftp.public_host`
- `storage.root`

### `media_assets`

| 字段 | 说明 |
|---|---|
| `id` | 主键 |
| `original_filename` | 原文件名 |
| `storage_path` | 内部存储相对路径 |
| `content_hash` | 文件内容 hash |
| `size` | 文件大小 |
| `mime_type` | MIME |
| `is_photo` | 是否可识别为照片 |
| `exif_taken_at` | EXIF 拍摄时间 |
| `fallback_taken_at` | 兜底排序时间 |
| `uploaded_at` | 上传时间 |
| `updated_at` | 更新时间 |

### `transfer_events`

| 字段 | 说明 |
|---|---|
| `id` | 主键 |
| `asset_id` | 关联媒体资产，可为空用于失败事件 |
| `event_type` | uploaded、overwritten、deleted、failed |
| `original_filename` | 原文件名 |
| `content_hash` | 文件 hash |
| `remote_addr` | FTP 客户端地址 |
| `message` | 可读说明 |
| `created_at` | 创建时间 |

## 约束

- `system_accounts` 首期只允许一条有效记录。
- `ftp_account` 首期只允许一条有效记录。
- `media_assets.storage_path` 唯一。
- 同名同 hash 上传应更新或覆盖同一内容记录。
- 同名不同 hash 上传必须生成不同 `storage_path`，保留两条资产记录。
- 照片页只查询 `is_photo=true`。

## 初始化数据

1. 启动或 migration 后检查默认系统账号是否存在。
2. 不存在时从环境变量创建默认系统账号。
3. 检查默认 FTP 账号是否存在。
4. 不存在时从环境变量创建默认 FTP 账号和匿名策略。
5. 默认密码缺失时启动应明确失败，避免生成不可追踪的弱密码。

## 实施步骤

1. 编写初始 migration。
2. 编写账号和配置 seed 逻辑。
3. 编写 repository 或 query 层。
4. 编写模型 DTO，区分 DB model 和 API response。
5. 编写数据库单元测试或集成测试。

## 验收标准

- migration 可在空库执行成功。
- 默认系统账号和 FTP 账号能初始化。
- `media_assets` 可表达同名同 hash 覆盖和同名不同 hash 并存。
- `transfer_events` 能记录上传、覆盖、删除。

## 不做项

- 不建 `roles`。
- 不建 `user_roles`。
- 不建 `audit_logs`。
- 不建 `ai_edit_jobs`。
- 不建对象存储表。

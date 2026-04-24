# 02 数据库与模型计划

## 依据文档

- `docs/rewrite-analysis/03-data-model.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

建立 Web MVP 的 PostgreSQL schema，覆盖唯一系统账号、唯一 FTP 账号、应用配置、媒体资产和传输事件。

## migration 策略

- migration 文件必须可重复应用到空库，失败时阻止 app ready。
- 表结构、索引、约束和 enum/check constraint 放在 migration 中，不只写在 Go model。
- seed 逻辑放在应用启动流程中，便于读取环境变量并避免把默认密码写死到 migration。
- migration 命令需要能在本地和 Compose 中运行。

## 核心表

### `system_accounts`

| 字段 | 说明 |
|---|---|
| `id` | 主键，建议固定单行 UUID 或固定 bigint |
| `username` | 默认系统账号用户名 |
| `password_hash` | 密码 hash |
| `password_updated_at` | 密码更新时间 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

建议约束：

- `username` 非空且唯一。
- 使用固定主键或 partial unique index 保证首期只有一条有效记录。
- `password_hash` 只保存 hash，不能保存明文或可逆加密值。

### `ftp_account`

| 字段 | 说明 |
|---|---|
| `id` | 主键，建议固定单行 |
| `username` | 默认 FTP 用户名 |
| `password_hash` | FTP 密码 hash |
| `anonymous_enabled` | 是否允许匿名访问 |
| `password_updated_at` | 密码更新时间 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

建议约束：

- `username` 非空。
- `anonymous_enabled=false` 时，应用层必须保证 `username` 和 `password_hash` 非空。
- 首期只有一个 FTP 账号，不建账号列表。

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
- `storage.tmp_root`

建议规则：

- `value_json` 保存结构化值，例如 `"30000-30009"` 或 `{ "start": 30000, "end": 30009 }`，最终格式需与配置解析统一。
- settings 修改必须更新 `updated_at`。
- 不保存 `SESSION_SECRET`、默认密码、数据库连接串等 secret。

### `media_assets`

| 字段 | 说明 |
|---|---|
| `id` | 主键 |
| `original_filename` | 原文件名 |
| `relative_dir` | FTP 相对目录或归档目录 |
| `storage_path` | 内部存储相对路径 |
| `content_hash` | 文件内容 hash |
| `hash_algorithm` | hash 算法，首期建议 `sha256` |
| `size` | 文件大小 |
| `mime_type` | MIME |
| `is_photo` | 是否可识别为照片 |
| `exif_taken_at` | EXIF 拍摄时间 |
| `fallback_taken_at` | 兜底排序时间 |
| `taken_at_effective` | 用于排序的有效时间，可由查询层用 `COALESCE` 生成 |
| `uploaded_at` | 上传时间 |
| `last_seen_at` | 同名同 hash 覆盖或重复上传时更新时间 |
| `deleted_at` | 如选择软删则使用；P0 可物理删除 |
| `updated_at` | 更新时间 |

建议索引：

- `UNIQUE(storage_path)`。
- `INDEX(is_photo, exif_taken_at DESC, fallback_taken_at DESC)`。
- `INDEX(content_hash)`。
- `INDEX(original_filename, content_hash)`，支持同名同 hash 覆盖查找。
- `INDEX(uploaded_at DESC)`。

### `transfer_events`

| 字段 | 说明 |
|---|---|
| `id` | 主键 |
| `asset_id` | 关联媒体资产，可为空用于失败事件 |
| `event_type` | uploaded、overwritten、deleted、failed |
| `original_filename` | 原文件名 |
| `content_hash` | 文件 hash |
| `remote_addr` | FTP 客户端地址 |
| `bytes` | 本次传输字节数 |
| `message` | 可读说明 |
| `created_at` | 创建时间 |

建议约束：

- `event_type` 使用 check constraint 或 enum：`uploaded`、`overwritten`、`deleted`、`failed`。
- `asset_id` 删除后可为空或保留历史引用；如果使用外键，删除策略需明确是 `SET NULL` 还是先写事件再删资产。
- `message` 只存人可读摘要，不存大段异常栈。

## 约束

- `system_accounts` 首期只允许一条有效记录。
- `ftp_account` 首期只允许一条有效记录。
- `media_assets.storage_path` 唯一。
- 同名同 hash 上传应更新或覆盖同一内容记录。
- 同名不同 hash 上传必须生成不同 `storage_path`，保留两条资产记录。
- 照片页只查询 `is_photo=true`。
- 浏览器访问内容时只使用 `id`，任何 API response 都不能暴露宿主机绝对路径。
- 所有时间字段使用 `timestamptz`。

## 查询与事务边界

### 上传入库事务

1. storage 层完成文件落盘并计算 hash。
2. 在事务中查找同名同 hash 记录。
3. 同名同 hash：更新 size、mime、时间字段、`last_seen_at`，写 `transfer_events.overwritten`。
4. 同名不同 hash 或新文件：插入新 `media_assets`，写 `transfer_events.uploaded`。
5. 事务提交后发布 WebSocket 事件。

### 删除事务

1. 查询资产并锁定记录。
2. 删除本地文件。
3. 在事务中写 `transfer_events.deleted` 并删除资产记录。
4. 事务提交后发布 `photo-deleted`。
5. 文件删除失败时不删除数据库记录。

## API DTO 边界

| DB 字段 | API 暴露 | 说明 |
|---|---|---|
| `storage_path` | 否 | 服务器内部相对路径，不返回给前端 |
| `content_hash` | 是 | 用于展示和同名区分 |
| `exif_taken_at` | 是 | 可为空 |
| `fallback_taken_at` | 是 | 必填 |
| `taken_at_effective` | 可选 | 前端也可自行按规则计算 |
| `deleted_at` | 否 | P0 若物理删除则不存在 |

## 初始化数据

1. 启动或 migration 后检查默认系统账号是否存在。
2. 不存在时从环境变量创建默认系统账号。
3. 检查默认 FTP 账号是否存在。
4. 不存在时从环境变量创建默认 FTP 账号和匿名策略。
5. 默认密码缺失时启动应明确失败，避免生成不可追踪的弱密码。
6. 初始化基础 settings：FTP 控制端口、PASV 端口、public host、本地存储根目录和临时目录。

## 实施步骤

1. 编写初始 migration。
2. 编写账号和配置 seed 逻辑。
3. 编写 repository 或 query 层。
4. 编写模型 DTO，区分 DB model 和 API response。
5. 编写数据库单元测试或集成测试。
6. 编写上传入库和删除一致性的事务测试。
7. 编写索引覆盖的关键查询测试：照片分页、按时间分组、同名同 hash 查找。

## 验收标准

- migration 可在空库执行成功。
- 默认系统账号和 FTP 账号能初始化。
- `media_assets` 可表达同名同 hash 覆盖和同名不同 hash 并存。
- `transfer_events` 能记录上传、覆盖、删除。
- `/api/photos` 所需查询不依赖扫描本地文件系统。
- API response 不暴露服务器真实路径。
- 删除失败不会产生“API 返回成功但数据库/文件不一致”的静默状态。

## 不做项

- 不建 `roles`。
- 不建 `user_roles`。
- 不建 `refresh_tokens`，除非后续认证方案变更为双 token。
- 不建 `audit_logs`。
- 不建 `ai_edit_jobs`。
- 不建对象存储表。

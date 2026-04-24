# 03 认证与账号计划

## 依据文档

- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/06-permission-model.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

实现唯一默认系统账号登录和唯一默认 FTP 账号管理。Web 登录保护 HTTP API，FTP 账号用于相机 FTP 客户端连接。

## 范围裁剪

- 本项目已确认不做多用户，因此 Lazycat 通用基线中的注册页不进入 P0。
- 首期认证只服务唯一默认系统账号；登录后拥有全部系统操作权限。
- 首期建议 HttpOnly Cookie session；如后续改为 `access_token + refresh_token`，必须同步更新前端会话计划、数据库 token 表和测试。
- FTP 账号不是 Web 用户，不参与 HTTP 登录。

## HTTP API

| 方法 | 路径 | 用途 |
|---|---|---|
| `POST` | `/api/auth/login` | 默认系统账号登录 |
| `POST` | `/api/auth/logout` | 退出登录 |
| `GET` | `/api/auth/me` | 当前系统账号 |
| `PUT` | `/api/system/account` | 更新默认系统账号 |
| `PUT` | `/api/ftp/account` | 更新默认 FTP 账号和匿名策略 |

## 会话方案

首期建议使用 HttpOnly Cookie session：

- 前端不接触 token。
- API middleware 统一校验 session。
- `SameSite=Lax`。
- 生产环境启用 `Secure`，本地开发允许关闭。
- session secret 来自环境变量。
- session id 必须不可预测，并在服务端可失效。
- logout 必须让当前 session 失效。
- session 过期或无效时统一返回 401 + `AUTH_REQUIRED` 或 `SESSION_INVALID`。

## session 存储建议

首期可二选一：

| 方案 | 适用 | 要求 |
|---|---|---|
| 签名 Cookie session | 单实例部署 | Cookie 内容不包含敏感信息，签名 secret 来自环境变量 |
| DB session 表 | 需要服务端主动失效更清晰 | 建表保存 session hash、expires_at、revoked_at |

如果采用签名 Cookie session，修改系统账号密码后必须通过 session version、password timestamp 或全局 session secret rotation 让旧会话失效。

## 密码规则

- 系统账号密码和 FTP 密码都使用安全 hash。
- 明文密码只在请求处理过程中存在。
- 非匿名 FTP 模式下用户名和密码必填。
- 不允许像原项目一样在非匿名配置不完整时静默降级为匿名。
- 密码更新接口不返回明文密码。
- 日志中禁止输出密码、hash、Cookie、session id。
- 密码 hash 参数需要集中定义，便于未来升级成本。

## 请求与响应草案

### `POST /api/auth/login`

```json
{
  "username": "admin",
  "password": "..."
}
```

成功返回：

```json
{
  "account": {
    "username": "admin",
    "updatedAt": "2026-04-25T10:00:00Z"
  }
}
```

### `PUT /api/system/account`

```json
{
  "username": "admin",
  "currentPassword": "...",
  "newPassword": "..."
}
```

- `currentPassword` 必须正确。
- `newPassword` 为空时仅修改用户名；如果实现阶段决定禁止单独改用户名，需要在 API 契约中固定。
- 保存成功后刷新当前 session 对应的账号摘要。

### `PUT /api/ftp/account`

```json
{
  "username": "camera",
  "password": "...",
  "anonymousEnabled": true
}
```

- `anonymousEnabled=false` 时 `username` 和 `password` 必填。
- `anonymousEnabled=true` 时仍可保存默认账号，方便用户后续关闭匿名。
- 如果 FTP server 正在运行，后端需要决定是热更新认证器还是要求重启；该行为必须体现在响应或错误码中。

## FTP 账号规则

- 只保留一个默认 FTP 账号。
- 支持匿名访问。
- 匿名访问不限制 IP。
- 匿名开启状态必须可通过 `/api/system/info` 和 `/api/ftp/status` 返回给前端展示。
- 匿名开启时，FTP 认证器允许匿名登录；是否接受任意用户名密码由 FTP 库能力决定，但文档和 UI 要表达“匿名已开启”。
- 匿名关闭时，FTP 认证器必须校验唯一默认账号密码。

## API 鉴权边界

| 接口 | 是否需要 Web 登录 |
|---|---|
| `/healthz`、`/readyz` | 否 |
| `POST /api/auth/login` | 否 |
| `POST /api/auth/logout` | 是或允许幂等 |
| `GET /api/auth/me` | 是 |
| `/api/photos/**` | 是 |
| `/api/assets/**` | 是 |
| `/api/system/info` | 是 |
| `/api/ftp/**` | 是 |
| `/api/ws` | 是 |

## 实施步骤

1. 实现 password hash 和 verify。
2. 实现 session 创建、读取、销毁。
3. 实现 auth middleware。
4. 实现 `/api/auth/login`、`/api/auth/logout`、`/api/auth/me`。
5. 实现默认系统账号更新。
6. 实现默认 FTP 账号更新。
7. 将 FTP 运行时认证器接入数据库中的 FTP 账号和匿名策略。
8. 编写登录、未登录、密码错误、FTP 非匿名校验测试。
9. 编写敏感信息日志测试或代码审查清单，确保密码和 session 不进入日志。
10. 编写系统账号密码更新后的旧密码/旧会话失效测试。

## 错误码

- `AUTH_REQUIRED`
- `AUTH_INVALID_CREDENTIALS`
- `SYSTEM_ACCOUNT_INVALID`
- `FTP_ACCOUNT_INVALID`
- `SESSION_INVALID`
- `SESSION_EXPIRED`
- `CURRENT_PASSWORD_INVALID`
- `ACCOUNT_UPDATE_REQUIRES_RELOGIN`

## 验收标准

- 默认系统账号可登录。
- 未登录访问受保护 API 返回 401。
- 修改系统账号密码后旧密码失效。
- 登出后当前会话不可继续访问受保护 API。
- 匿名 FTP 关闭且账号密码缺失时拒绝保存。
- FTP 认证器能读取最新账号配置。
- `/api/system/info` 和 `/api/ftp/status` 能返回匿名状态和 FTP 用户名摘要，不返回密码 hash。

## 不做项

- 不做注册。
- 不做找回密码。
- 不做多用户。
- 不做 RBAC。
- 不做 OAuth/OIDC。
- 不做审计日志。

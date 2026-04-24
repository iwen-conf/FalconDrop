# 03 认证与账号计划

## 依据文档

- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/06-permission-model.md`
- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

实现唯一默认系统账号登录和唯一默认 FTP 账号管理。Web 登录保护 HTTP API，FTP 账号用于相机 FTP 客户端连接。

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

## 密码规则

- 系统账号密码和 FTP 密码都使用安全 hash。
- 明文密码只在请求处理过程中存在。
- 非匿名 FTP 模式下用户名和密码必填。
- 不允许像原项目一样在非匿名配置不完整时静默降级为匿名。

## FTP 账号规则

- 只保留一个默认 FTP 账号。
- 支持匿名访问。
- 匿名访问不限制 IP。
- 匿名开启状态必须可通过 `/api/system/info` 和 `/api/ftp/status` 返回给前端展示。

## 实施步骤

1. 实现 password hash 和 verify。
2. 实现 session 创建、读取、销毁。
3. 实现 auth middleware。
4. 实现 `/api/auth/login`、`/api/auth/logout`、`/api/auth/me`。
5. 实现默认系统账号更新。
6. 实现默认 FTP 账号更新。
7. 将 FTP 运行时认证器接入数据库中的 FTP 账号和匿名策略。
8. 编写登录、未登录、密码错误、FTP 非匿名校验测试。

## 错误码

- `AUTH_REQUIRED`
- `AUTH_INVALID_CREDENTIALS`
- `SYSTEM_ACCOUNT_INVALID`
- `FTP_ACCOUNT_INVALID`
- `SESSION_INVALID`

## 验收标准

- 默认系统账号可登录。
- 未登录访问受保护 API 返回 401。
- 修改系统账号密码后旧密码失效。
- 匿名 FTP 关闭且账号密码缺失时拒绝保存。
- FTP 认证器能读取最新账号配置。

## 不做项

- 不做注册。
- 不做多用户。
- 不做 RBAC。
- 不做 OAuth/OIDC。
- 不做审计日志。

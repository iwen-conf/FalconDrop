# 02 登录与会话计划

## 依据文档

- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/05-page-map.md`
- `docs/rewrite-analysis/06-permission-model.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

实现唯一默认系统账号登录。登录后拥有全部系统操作权限，不做 RBAC、多用户和角色区分。

## 范围裁剪

- 不做注册页、找回密码、多用户切换和角色菜单。
- 前端不保存密码、session secret 或 token。
- 如果后端采用 HttpOnly Cookie session，前端只通过 `/api/auth/me` 判断会话。
- 如果后端后续改为双 token，本文件需要同步补充 silent refresh、并发刷新锁和 refresh 失败兜底；P0 暂不做。

## API 契约

| 方法 | 路径 | 用途 |
|---|---|---|
| `POST` | `/api/auth/login` | 使用默认系统账号登录 |
| `POST` | `/api/auth/logout` | 退出登录 |
| `GET` | `/api/auth/me` | 获取当前登录账号 |

## 页面与组件

- `features/auth/LoginPage.tsx`
- `features/auth/useSession.ts`
- `features/auth/RequireAuth.tsx`
- `features/auth/authApi.ts`
- `features/auth/AuthLayout.tsx`
- `features/auth/loginSchema.ts`

## 数据模型

```ts
type CurrentAccount = {
  username: string;
  updatedAt: string;
};

type LoginRequest = {
  username: string;
  password: string;
};
```

## 表单校验

使用 React Hook Form + Zod：

```ts
const loginSchema = z.object({
  username: z.string().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
});
```

- 登录按钮在提交中显示 loading。
- Enter 键提交表单。
- 错误密码展示中文错误，不展示原始错误栈。
- 401、网络错误、服务不可用需要区分文案。

## 会话状态模型

```ts
type SessionState =
  | { status: 'checking' }
  | { status: 'anonymous' }
  | { status: 'authenticated'; account: CurrentAccount };
```

- 应用启动先进入 `checking`，调用 `/api/auth/me`。
- `authenticated` 才允许进入 `/photos` 和 `/system`。
- `anonymous` 访问受保护页面时跳转 `/login`，并记录目标地址用于登录后回跳。

## 实施步骤

1. 登录页提供用户名、密码输入和提交按钮。
2. 提交时调用 `/api/auth/login`，成功后刷新 `/api/auth/me` 并跳转 `/photos`。
3. 路由守卫在进入 `/photos`、`/system` 前调用 `/api/auth/me`。
4. API 返回 `AUTH_REQUIRED` 或 HTTP 401 时清理前端会话状态并跳转 `/login`。
5. 顶部导航提供退出按钮，调用 `/api/auth/logout` 后回到登录页。
6. 登录错误需要展示中文原因，不直接暴露内部错误码。
7. WebSocket 连接应在登录后建立，退出或 401 后关闭。
8. 使用 shadcn/ui 的 Form、Input、Button、Alert 等组件保持样式一致。

## 状态规则

- 前端不保存密码。
- 如果后端使用 HttpOnly Cookie session，前端只依赖 `/api/auth/me` 判断登录态。
- 刷新页面后必须通过 `/api/auth/me` 恢复会话。
- 退出登录后清理 TanStack Query cache，避免重新登录前看到旧照片或系统信息。
- 登录页不显示注册入口。

## 错误文案

| 错误 | 前端文案方向 |
|---|---|
| `AUTH_INVALID_CREDENTIALS` | 用户名或密码不正确 |
| `AUTH_REQUIRED` | 请先登录 |
| `SESSION_EXPIRED` | 登录已过期，请重新登录 |
| 网络错误 | 无法连接服务，请检查后端是否运行 |
| 其他错误 | 使用后端中文 `message`，附带 requestId 供排查 |

## 验收标准

- 默认账号可登录。
- 错误密码显示中文错误提示。
- 未登录不能访问照片页和系统信息页。
- 退出后再访问受保护页面会回到登录页。
- 刷新页面后已登录会话可以恢复。
- 401 会关闭 WebSocket 并清理业务 Query cache。

## 不做项

- 不做注册。
- 不做找回密码。
- 不做多用户切换。
- 不做角色、权限、菜单配置。

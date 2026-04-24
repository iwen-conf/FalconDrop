# 02 登录与会话计划

## 依据文档

- `docs/rewrite-analysis/04-api-map.md`
- `docs/rewrite-analysis/05-page-map.md`
- `docs/rewrite-analysis/06-permission-model.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

实现唯一默认系统账号登录。登录后拥有全部系统操作权限，不做 RBAC、多用户和角色区分。

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

## 实施步骤

1. 登录页提供用户名、密码输入和提交按钮。
2. 提交时调用 `/api/auth/login`，成功后刷新 `/api/auth/me` 并跳转 `/photos`。
3. 路由守卫在进入 `/photos`、`/system` 前调用 `/api/auth/me`。
4. API 返回 `AUTH_REQUIRED` 或 HTTP 401 时清理前端会话状态并跳转 `/login`。
5. 顶部导航提供退出按钮，调用 `/api/auth/logout` 后回到登录页。
6. 登录错误需要展示中文原因，不直接暴露内部错误码。

## 状态规则

- 前端不保存密码。
- 如果后端使用 HttpOnly Cookie session，前端只依赖 `/api/auth/me` 判断登录态。
- 刷新页面后必须通过 `/api/auth/me` 恢复会话。

## 验收标准

- 默认账号可登录。
- 错误密码显示中文错误提示。
- 未登录不能访问照片页和系统信息页。
- 退出后再访问受保护页面会回到登录页。

## 不做项

- 不做注册。
- 不做找回密码。
- 不做多用户切换。
- 不做角色、权限、菜单配置。

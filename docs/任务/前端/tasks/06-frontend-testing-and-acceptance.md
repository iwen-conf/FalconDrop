# 06 前端测试与验收计划

## 依据文档

- `docs/rewrite-analysis/09-go-react-rewrite-plan.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

为前端首期交付建立可重复验证标准，覆盖登录、照片分组、删除、系统信息展示和 WebSocket 增量刷新。

## 单元测试

- 登录表单：必填、错误密码、成功跳转。
- 路由守卫：未登录跳转 `/login`，已登录进入目标页。
- 照片分组：优先 EXIF 时间，缺失 EXIF 时使用兜底时间。
- 同名照片：同名不同 hash 不被前端去重。
- 删除流程：确认、成功移除、失败保留。
- API 错误：401、404、存储错误、FTP 端口错误。

## 组件测试

- 照片空状态。
- 照片网格 loading/error/ready 状态。
- 预览层元信息展示。
- 系统信息卡片。
- 匿名 FTP 风险提示。
- PASV 和存储状态提示。

## E2E 验收路径

1. 打开 `/login`。
2. 使用默认系统账号登录。
3. 进入 `/system`，确认系统版本、hash、系统时间、系统账号、FTP 账号可见。
4. 启动 FTP 服务。
5. 使用 FTP 客户端上传照片。
6. 进入 `/photos`，确认照片通过 WebSocket 出现。
7. 刷新页面，确认照片仍存在。
8. 删除照片，确认 UI 移除。
9. 再次刷新，确认照片不再出现。

## 验收命令

```bash
npm run typecheck
npm run test
npm run build
```

## 验收标准

- 所有测试命令通过。
- 两个主页面中文文案完整。
- 没有 Android、Windows、Tauri、AI、FTPS、RBAC 入口。
- E2E 主路径可复现。

## 后续 P1 测试

- 照片搜索筛选。
- 缩略图性能。
- 部署检查状态块。
- 系统日志展示。

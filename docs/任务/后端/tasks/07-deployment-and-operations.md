# 07 部署与运维计划

## 依据文档

- `docs/rewrite-analysis/07-config-and-dependencies.md`
- `docs/rewrite-analysis/08-deployment.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

提供可作为正式部署方式的 Docker Compose，包含 Web/API 服务、PostgreSQL、本地数据卷、FTP 控制端口和 PASV 端口映射。

## Compose 服务

- `app`：Go API + FTP server + React 静态资源。
- `postgres`：PostgreSQL。

## 端口

```txt
HTTP: 8080
FTP control: 2121
FTP PASV: 30000-30009
```

实际端口可通过 `.env` 调整，但 Compose 必须明确映射控制端口和 PASV 端口范围。

## 数据卷

```txt
postgres-data:/var/lib/postgresql/data
falcondrop-uploads:/data/falcondrop/uploads
falcondrop-tmp:/data/falcondrop/tmp
```

## 部署检查

启动时必须检查：

1. PostgreSQL 可连接。
2. migration 已执行。
3. `STORAGE_ROOT` 存在且可写。
4. `TMP_ROOT` 存在且可写。
5. FTP 控制端口配置合法。
6. PASV 端口范围配置合法。
7. `SESSION_SECRET` 非空。
8. 默认账号初始化变量可用。

## 运维提示

- FTP 控制端口和 PASV 端口不是 HTTP 反向代理端口，需要在 Docker 映射、防火墙和运行环境中单独开放。
- 匿名 FTP 已确认支持且不限制 IP，部署时应依赖内网边界或外部防火墙控制访问面。
- 日志输出到 stdout，由 Docker 或宿主机日志系统收集。
- 媒体文件只能通过受控 API 访问，不暴露宿主机真实路径。

## 实施步骤

1. 编写 `deployments/Dockerfile`。
2. 编写 `deployments/docker-compose.yml`。
3. 编写 `.env.example`。
4. 实现 app 容器启动命令和 migration 执行策略。
5. 配置本地数据卷。
6. 配置 FTP 控制端口和 PASV 端口映射。
7. 编写部署 smoke test 文档或脚本。
8. 在系统信息 API 中返回存储和 FTP 部署状态。

## 验收标准

- `docker compose up` 可以启动 app 和 PostgreSQL。
- 默认系统账号可登录。
- FTP 默认账号和匿名模式可连接。
- FTP 上传文件后数据卷和 PostgreSQL 均有记录。
- 前端能通过 `/api/photos/{id}/content` 读取照片，不暴露真实路径。

## 不做项

- 不部署 Redis。
- 不部署 MinIO。
- 不部署 AI worker。
- 不在首期内置 HTTPS 证书管理。

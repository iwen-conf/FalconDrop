# 07 部署与运维计划

## 依据文档

- `docs/rewrite-analysis/07-config-and-dependencies.md`
- `docs/rewrite-analysis/08-deployment.md`
- `docs/rewrite-analysis/10-priority-and-roadmap.md`
- `docs/rewrite-analysis/11-risks-and-open-questions.md`
- `docs/rewrite-analysis/12-confirmed-web-scope.md`

## 目标

提供可作为正式部署方式的 Docker Compose，包含 Web/API 服务、PostgreSQL、本地数据卷、FTP 控制端口和 PASV 端口映射。

## 部署形态

- `app` 容器同时提供 Go HTTP API、WebSocket、React 静态资源和 FTP server。
- `postgres` 容器提供唯一数据库。
- 上传文件和临时文件使用持久化 volume。
- FTP 控制端口和 PASV 端口必须用 Docker 端口映射暴露，不能依赖 HTTP 反向代理。

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

## `.env.example` 必填项

```txt
POSTGRES_DB=falcondrop
POSTGRES_USER=falcondrop
POSTGRES_PASSWORD=change-me
DATABASE_URL=postgres://falcondrop:change-me@postgres:5432/falcondrop?sslmode=disable
HTTP_ADDR=:8080
STORAGE_ROOT=/data/falcondrop/uploads
TMP_ROOT=/data/falcondrop/tmp
FTP_HOST=0.0.0.0
FTP_PUBLIC_HOST=
FTP_PORT=2121
FTP_PASSIVE_PORTS=30000-30009
SESSION_SECRET=change-me
COOKIE_SECURE=false
DEFAULT_SYSTEM_USERNAME=admin
DEFAULT_SYSTEM_PASSWORD=change-me
DEFAULT_FTP_USERNAME=camera
DEFAULT_FTP_PASSWORD=change-me
DEFAULT_FTP_ANONYMOUS_ENABLED=true
```

示例 secret 必须标注需要替换，不能作为生产默认值静默使用。

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

## 容器健康检查

| 服务 | 检查 |
|---|---|
| `postgres` | `pg_isready` |
| `app` | `GET /readyz` |

`/readyz` 至少覆盖数据库连接、migration 状态和存储可写性。FTP 未启动不应导致 `/readyz` 失败，因为 FTP 可由用户在系统页启动。

## 运维提示

- FTP 控制端口和 PASV 端口不是 HTTP 反向代理端口，需要在 Docker 映射、防火墙和运行环境中单独开放。
- 匿名 FTP 已确认支持且不限制 IP，部署时应依赖内网边界或外部防火墙控制访问面。
- 日志输出到 stdout，由 Docker 或宿主机日志系统收集。
- 媒体文件只能通过受控 API 访问，不暴露宿主机真实路径。
- PostgreSQL 数据卷和上传数据卷需要一起备份，单独备份其中之一会造成媒体记录和文件不一致。
- PASV 端口范围越大，并发 FTP 数据连接余量越大；首期默认 `30000-30009`。

## Dockerfile 要求

- 多阶段构建：前端构建产物进入 Go app 镜像，或 Go embed 静态资源。
- 最终镜像不包含源码构建缓存。
- 运行用户尽量非 root；如果 FTP 端口使用小于 1024，需额外能力或改用高端口，首期默认 2121 避免特权端口。
- app 启动命令必须在 migration、seed 或 storage check 失败时非零退出。

## Lazycat 打包衔接

本任务只要求 Docker Compose 正式部署。若后续进入 Lazycat lpk 打包，需要补齐：

- `package.yml`
- `lzc-build.yml`
- `lzc-manifest.yml`
- app service、PostgreSQL、数据卷和 FTP/PASV 端口映射。

这些不进入当前 P0 文档实现范围，但 Compose 端口和数据卷命名应避免后续迁移成本。

## 实施步骤

1. 编写 `deployments/Dockerfile`。
2. 编写 `deployments/docker-compose.yml`。
3. 编写 `.env.example`。
4. 实现 app 容器启动命令和 migration 执行策略。
5. 配置本地数据卷。
6. 配置 FTP 控制端口和 PASV 端口映射。
7. 编写部署 smoke test 文档或脚本。
8. 在系统信息 API 中返回存储和 FTP 部署状态。
9. 编写备份恢复说明：PostgreSQL dump + uploads volume。
10. 编写匿名 FTP 网络边界说明。

## 验收标准

- `docker compose up` 可以启动 app 和 PostgreSQL。
- 默认系统账号可登录。
- FTP 默认账号和匿名模式可连接。
- FTP 上传文件后数据卷和 PostgreSQL 均有记录。
- 前端能通过 `/api/photos/{id}/content` 读取照片，不暴露真实路径。
- `docker compose -f deployments/docker-compose.yml config` 通过。
- `/readyz` 在数据库和存储都正常时返回成功。

## 不做项

- 不部署 Redis。
- 不部署 MinIO。
- 不部署 AI worker。
- 不在首期内置 HTTPS 证书管理。
- 不在本任务内提交 Lazycat lpk，上架发布另开任务。

# NPanel Backend

[English](README.md) | [简体中文](README.zh-CN.md)

NPanel Backend 是 NPanel 生态的 Go 服务核心，提供管理端、用户端、认证、支付、订阅、节点与服务端上报等 API 能力。

NPanel 由 [npanel-dev](https://github.com/npanel-dev) 组织维护。项目目标是构建高性能的分布式连接基础设施，在保证后端架构清晰、运维体验简单的同时，逐步形成面板、前端、节点、SDK、客户端与订阅模板等完整生态。

官方频道：[官网 Website](https://npanel.dev/) | [Telegram 频道](https://t.me/mynpanel) | [Telegram 群组](https://t.me/NPanelChat)

相关链接：

- 官网：[npanel.dev](https://npanel.dev)
- Telegram 频道：[t.me/mynpanel](https://t.me/mynpanel)
- Telegram 群组：[t.me/NPanelChat](https://t.me/NPanelChat)
- GitHub 组织：[github.com/npanel-dev](https://github.com/npanel-dev)
- 后端仓库：[github.com/npanel-dev/NPanel-backend](https://github.com/npanel-dev/NPanel-backend)
- 前端仓库：[github.com/npanel-dev/NPanel-frontend](https://github.com/npanel-dev/NPanel-frontend)

## 系统架构

NPanel Backend 基于 Kratos、Proto、Ent ORM、MySQL/PostgreSQL、Redis 与异步任务队列构建，采用分层架构，便于扩展和维护。

```text
Admin / User / Node Clients
          |
      HTTP / gRPC
          |
   internal/server
          |
   internal/service
          |
    internal/biz
          |
 internal/data + ent
          |
    MySQL/PostgreSQL + Redis

异步任务: internal/queue
通用能力: pkg/*
接口契约: api/*
启动入口: cmd/npanel
运行配置: configs/*
```

核心目录：

- `cmd/npanel`：应用启动入口，负责配置加载、日志初始化、Wire 依赖注入与服务启动。
- `api`：Proto 契约与生成的 HTTP/gRPC 代码，覆盖 admin、public、auth、common、server 等 API。
- `internal/server`：HTTP/gRPC 服务注册、中间件、兼容路由与 CORS。
- `internal/service`：接口服务层。
- `internal/biz`：用户、订阅、订单、支付、工单、公告、营销、节点、系统设置等领域逻辑。
- `internal/data`：MySQL/PostgreSQL、Redis、Ent 仓储、运行时配置同步与队列初始化。
- `internal/queue`：邮件、短信、流量统计、配额更新、订单生命周期、订阅检查等异步任务。
- `pkg`：支付、邮件、短信、JWT、缓存、模板渲染、追踪、客户端适配与工具函数等通用包。

## 功能特性

- 用户与认证：邮箱、手机号、验证码、找回密码、管理员登录与 OAuth 集成。
- 订阅与账户管理：套餐、节点组、设备状态、流量、用户资料、余额与状态控制。
- 支付能力：Stripe、Alipay F2F、EPay、CryptoSaaS 兼容 EPay 流程、兑换码、优惠券与余额支付。
- 运营模块：公告、文档、广告、工单、营销、日志、系统设置与后台控制台数据。
- 节点与服务端接口：节点/服务端上报、运行时配置同步、订阅模板与兼容接口。
- HTTP 与 gRPC 双协议服务。
- 首次启动自动执行数据库迁移和默认数据初始化。

## Docker Compose 快速上手

这是本地或小型服务器部署的推荐方式。默认 Compose 会构建后端镜像，并同时启动 MySQL、Redis 与 NPanel Backend。

环境要求：

- Docker 与 Compose v2
- 宿主机 `8081` 和 `9012` 端口可用；如果端口被占用，可使用下面的端口覆盖方式

启动：

```bash
docker compose up -d --build
```

使用 PostgreSQL 17：

```bash
docker compose -f docker-compose.postgres.yml up -d --build
```

PostgreSQL 模式下的 `ps`、`logs`、`restart`、`down` 命令也应带上 `-f docker-compose.postgres.yml`。

Linux 一键安装脚本也可交互选择数据库，非交互模式可显式指定：

```bash
curl -fsSL https://raw.githubusercontent.com/npanel-dev/NPanel-backend/dev/scripts/install.sh \
  | sudo bash -s -- --mode docker --database postgres
```

查看状态：

```bash
docker compose ps
docker compose logs -f npanel
```

默认访问地址：

- HTTP API：`http://127.0.0.1:8081`
- gRPC：`127.0.0.1:9012`

`configs/config.docker.yaml` 中的默认管理员：

- 邮箱：`admin@npanel.dev`
- 密码：`123456`

首次登录后请立即修改默认密码。

如果宿主机端口已被占用：

```bash
NPANEL_HTTP_PORT=18081 NPANEL_GRPC_PORT=19012 docker compose up -d --build
```

停止服务：

```bash
docker compose down
```

停止服务并删除本地测试数据卷：

```bash
docker compose down -v
```

## 配置与持久化

Compose 部署会使用 Docker volume 持久化运行数据：

- `npanel_mysql`：MySQL 数据
- `npanel_postgres`：PostgreSQL 数据（PostgreSQL Compose 模式）
- `npanel_redis`：Redis 数据
- `npanel_logs`：后端日志

容器内读取配置的位置是：

```text
/data/conf/config.yaml
```

Compose 模式下对应宿主机文件：

```text
./configs/config.docker.yaml
```

生产部署前建议修改 `configs/config.docker.yaml`：

- 修改 `server.auth.jwt_secret`
- 修改 `app.admin.email`
- 修改 `app.admin.password`
- 将 `app.site.host` 设置为真实域名
- 检查所用 Compose 文件中的数据库和 Redis 凭据

修改配置后重启后端：

```bash
docker compose restart npanel
```

## 单容器模式

仅在你已经有外部 MySQL/PostgreSQL 和 Redis 时使用。`data.database.driver` 支持 `mysql`、`postgres`（也接受 `postgresql` 别名）。

构建镜像：

```bash
docker build -t npanel .
```

准备配置目录：

```bash
mkdir -p ./npanel-config
cp configs/config.docker.yaml ./npanel-config/config.yaml
```

编辑 `./npanel-config/config.yaml`，将 `data.database.source` 和 `data.redis.addr` 指向你的外部服务。

运行：

```bash
docker run -d \
  --name npanel \
  -p 8081:8081 \
  -p 9012:9012 \
  -v "$(pwd)/npanel-config:/data/conf:ro" \
  npanel
```

查看日志：

```bash
docker logs -f npanel
```

修改配置后重启：

```bash
docker restart npanel
```

## 升级流程

使用源码和 Compose 部署时：

```bash
./scripts/docker-upgrade.sh
```

PostgreSQL Compose 部署请指定对应文件：

```bash
NPANEL_COMPOSE_FILE=docker-compose.postgres.yml ./scripts/docker-upgrade.sh
```

脚本会执行 `git pull --ff-only`、重新构建 NPanel 后端镜像，并只重建 `npanel` 后端容器。它不会拉取、升级或重建数据库和 Redis。

如果部署目录存在本地改动，不希望脚本执行 `git pull`，可以运行：

```bash
SKIP_GIT_PULL=1 ./scripts/docker-upgrade.sh
```

如果要指定本地构建的后端版本：

```bash
NPANEL_VERSION=v1.0.8 ./scripts/docker-upgrade.sh
```

使用单容器部署时：

```bash
docker build -t npanel .
docker stop npanel || true
docker rm npanel || true
docker run -d --name npanel -p 8081:8081 -p 9012:9012 -v "$(pwd)/npanel-config:/data/conf:ro" npanel
```

生产环境升级前，请先备份数据库和配置文件。

## Release 打包

维护者可以直接在 GitHub Actions 中发布二进制版本：

1. 打开 `Actions` -> `Release`
2. 点击 `Run workflow`
3. 输入版本号，例如 `v1.0.8`

该 workflow 会构建 64 位 Linux、macOS、Windows 版本，并把压缩包和 `SHA256SUMS` 上传到 GitHub Release。

如果需要在本地生成同样的发布包：

```bash
./scripts/build-release.sh v1.0.8
```

## 本地开发

环境要求：

- Go `1.25+`；当前仓库声明的 toolchain 为 `go1.26.4`
- MySQL 或 PostgreSQL，以及 Redis
- 用于重新生成 Proto 的 `protoc`
- `make init` 安装的 Go 代码生成工具

安装工具：

```bash
make init
```

修改 Proto 后生成 API 和内部配置代码：

```bash
make api
make config
```

需要重新生成 Wire/Ent 或整理模块时：

```bash
go generate ./...
go mod tidy
```

构建：

```bash
go build -o ./bin/npanel ./cmd/npanel
```

运行：

```bash
./bin/npanel -conf ./configs
```

也可以直接运行：

```bash
go run ./cmd/npanel -conf ./configs
```

默认的 `configs/config.yaml` 是开发模板。启动前请检查 `data.database.source`、`data.redis.addr`、`server.auth.jwt_secret` 和 `app.admin`。

## 诊断命令

查看容器状态：

```bash
docker compose ps
```

查看后端日志：

```bash
docker compose logs -f npanel
```

进入后端容器查看文件：

```bash
docker compose exec npanel /bin/sh
ls -la /app
ls -la /data/conf
```

查看宿主机端口占用：

```bash
lsof -nP -iTCP:8081 -sTCP:LISTEN
lsof -nP -iTCP:9012 -sTCP:LISTEN
```

## 常见问题

### 容器启动后立即退出怎么办？

先查看日志：

```bash
docker compose logs npanel
```

常见原因包括配置文件错误、数据库不可达、Redis 不可达、宿主机端口被占用。

### 修改配置后没有生效怎么办？

后端在启动时读取配置。修改配置后需要重启后端容器：

```bash
docker compose restart npanel
```

### 8081 或 9012 端口被占用怎么办？

启动时覆盖宿主机端口：

```bash
NPANEL_HTTP_PORT=18081 NPANEL_GRPC_PORT=19012 docker compose up -d --build
```

### 如何备份部署？

请备份数据库数据和配置文件。Compose 部署至少保留：

```text
configs/config.docker.yaml
docker-compose.yml
configs/config.postgres.yaml
docker-compose.postgres.yml
```

## License

MIT

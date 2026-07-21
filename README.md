# NPanel Backend

[English](README.md) | [简体中文](README.zh-CN.md)

NPanel Backend is the Go service core of the NPanel ecosystem. It provides the management, public, authentication, payment, subscription, node, and server APIs used by NPanel deployments.

NPanel is built by the [npanel-dev](https://github.com/npanel-dev) organization. The project goal is to provide high-performance distributed connectivity infrastructure with a clear operations experience, strong backend foundations, and an ecosystem that can grow across panel, frontend, node, SDK, client, and subscription-template projects.

Official channels: [Website](https://npanel.dev/) | [Telegram Channel](https://t.me/mynpanel) | [Telegram Group](https://t.me/NPanelChat)

Links:

- Website: [npanel.dev](https://npanel.dev)
- Telegram channel: [t.me/mynpanel](https://t.me/mynpanel)
- Telegram group: [t.me/NPanelChat](https://t.me/NPanelChat)
- GitHub organization: [github.com/npanel-dev](https://github.com/npanel-dev)
- Backend repository: [github.com/npanel-dev/NPanel-backend](https://github.com/npanel-dev/NPanel-backend)
- Frontend repository: [github.com/npanel-dev/NPanel-frontend](https://github.com/npanel-dev/NPanel-frontend)

## System Architecture

NPanel Backend uses a layered Go architecture based on Kratos, Proto contracts, Ent ORM, MySQL/PostgreSQL, Redis, and asynchronous workers.

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

Async workers: internal/queue
Shared packages: pkg/*
API contracts: api/*
Bootstrap: cmd/npanel
Runtime config: configs/*
```

Core modules:

- `cmd/npanel`: application entrypoint, config loading, logging bootstrap, Wire dependency injection, and server startup.
- `api`: Proto contracts and generated HTTP/gRPC code for admin, public, auth, common, and server APIs.
- `internal/server`: HTTP/gRPC server registration, middleware, compatibility routes, and CORS.
- `internal/service`: API service layer.
- `internal/biz`: domain logic for users, subscriptions, orders, payments, tickets, announcements, marketing, nodes, and system settings.
- `internal/data`: MySQL/PostgreSQL, Redis, Ent repositories, runtime config sync, and queue bootstrap.
- `internal/queue`: scheduled and asynchronous jobs for email, SMS, traffic statistics, quota updates, order lifecycle, and subscription checks.
- `pkg`: reusable packages for payment, email, SMS, JWT, cache, template rendering, tracing, adapters, and utilities.

## Features

- User and authentication flows: email, phone, verification code, password reset, admin login, and OAuth integrations.
- Subscription and account management: plans, node groups, device state, traffic, profiles, balances, and status control.
- Payment support: Stripe, Alipay F2F, EPay, CryptoSaaS-compatible EPay flow, redemption codes, coupons, and balance payment.
- Operations modules: announcements, documents, ads, tickets, marketing, logs, system settings, and admin console data.
- Node and server APIs: node/server reporting, runtime config sync, subscription templates, and compatibility endpoints.
- Dual protocol support: HTTP and gRPC.
- Automatic first-start database migration and default data initialization.

## Quick Start With Docker Compose

This is the recommended way to start NPanel Backend locally or on a small server. The default Compose file builds the backend image and starts MySQL, Redis, and NPanel together.

Requirements:

- Docker with Compose v2
- Ports `8081` and `9012` available on the host, or use the override shown below

Start:

```bash
docker compose up -d --build
```

To use PostgreSQL 17:

```bash
docker compose -f docker-compose.postgres.yml up -d --build
```

In PostgreSQL mode, also pass `-f docker-compose.postgres.yml` to `ps`, `logs`, `restart`, and `down`.

The Linux one-click installer also offers an interactive database choice. For non-interactive PostgreSQL installation:

```bash
curl -fsSL https://raw.githubusercontent.com/npanel-dev/NPanel-backend/dev/scripts/install.sh \
  | sudo bash -s -- --mode docker --database postgres
```

Check status:

```bash
docker compose ps
docker compose logs -f npanel
```

Default endpoints:

- HTTP API: `http://127.0.0.1:8081`
- gRPC: `127.0.0.1:9012`

Default admin account from `configs/config.docker.yaml`:

- Email: `admin@npanel.dev`
- Password: `123456`

Change the default password immediately after first login.

If host ports are already in use:

```bash
NPANEL_HTTP_PORT=18081 NPANEL_GRPC_PORT=19012 docker compose up -d --build
```

Stop the stack:

```bash
docker compose down
```

Stop and remove local test data volumes:

```bash
docker compose down -v
```

## Configuration And Persistence

The Compose stack persists runtime data in Docker volumes:

- `npanel_mysql`: MySQL data
- `npanel_postgres`: PostgreSQL data (PostgreSQL Compose mode)
- `npanel_redis`: Redis data
- `npanel_logs`: backend logs

The container reads configuration from:

```text
/data/conf/config.yaml
```

In Compose mode this is mounted from:

```text
./configs/config.docker.yaml
```

Edit `configs/config.docker.yaml` before production use:

- Change `server.auth.jwt_secret`
- Change `app.admin.email`
- Change `app.admin.password`
- Set `app.site.host` to your real domain
- Review database and Redis credentials in the selected Compose file

After editing config, restart:

```bash
docker compose restart npanel
```

## Single Container Mode

Use this only when you already have external MySQL/PostgreSQL and Redis. `data.database.driver` supports `mysql` and `postgres` (the `postgresql` alias is also accepted).

Build the image:

```bash
docker build -t npanel .
```

Prepare a config directory:

```bash
mkdir -p ./npanel-config
cp configs/config.docker.yaml ./npanel-config/config.yaml
```

Edit `./npanel-config/config.yaml` and point `data.database.source` and `data.redis.addr` to your external services.

Run:

```bash
docker run -d \
  --name npanel \
  -p 8081:8081 \
  -p 9012:9012 \
  -v "$(pwd)/npanel-config:/data/conf:ro" \
  npanel
```

Logs:

```bash
docker logs -f npanel
```

Restart after config changes:

```bash
docker restart npanel
```

## Upgrade

For Compose deployments from source:

```bash
./scripts/docker-upgrade.sh
```

For a PostgreSQL Compose deployment, select its Compose file explicitly:

```bash
NPANEL_COMPOSE_FILE=docker-compose.postgres.yml ./scripts/docker-upgrade.sh
```

The script runs `git pull --ff-only`, rebuilds the NPanel backend image, and recreates only the `npanel` container. It does not pull, upgrade, or recreate the database or Redis.

If your deployment directory has local changes and you want to skip `git pull`, run:

```bash
SKIP_GIT_PULL=1 ./scripts/docker-upgrade.sh
```

To upgrade to a specific backend build version while rebuilding locally:

```bash
NPANEL_VERSION=v1.0.8 ./scripts/docker-upgrade.sh
```

For single-container deployments:

```bash
docker build -t npanel .
docker stop npanel || true
docker rm npanel || true
docker run -d --name npanel -p 8081:8081 -p 9012:9012 -v "$(pwd)/npanel-config:/data/conf:ro" npanel
```

Always back up the database and your config directory before upgrading production systems.

## Release Builds

Maintainers can publish binary release packages from GitHub Actions:

1. Open `Actions` -> `Release`
2. Click `Run workflow`
3. Enter a version such as `v1.0.8`

The workflow builds 64-bit Linux, macOS, and Windows packages and uploads them to the GitHub Release together with `SHA256SUMS`.

To build the same packages locally:

```bash
./scripts/build-release.sh v1.0.8
```

## Local Development

Requirements:

- Go `1.25+`; this repository currently declares toolchain `go1.26.4`
- MySQL or PostgreSQL, plus Redis
- `protoc` for Proto regeneration
- Go code generation tools installed by `make init`

Install tooling:

```bash
make init
```

Generate API and internal config code when Proto files change:

```bash
make api
make config
```

Regenerate Wire/Ent and tidy modules when needed:

```bash
go generate ./...
go mod tidy
```

Build:

```bash
go build -o ./bin/npanel ./cmd/npanel
```

Run:

```bash
./bin/npanel -conf ./configs
```

Or:

```bash
go run ./cmd/npanel -conf ./configs
```

The default `configs/config.yaml` is a development template. Review `data.database.source`, `data.redis.addr`, `server.auth.jwt_secret`, and `app.admin` before starting.

## Diagnostics

Container status:

```bash
docker compose ps
```

Backend logs:

```bash
docker compose logs -f npanel
```

Inspect files inside the backend container:

```bash
docker compose exec npanel /bin/sh
ls -la /app
ls -la /data/conf
```

Check host port listeners:

```bash
lsof -nP -iTCP:8081 -sTCP:LISTEN
lsof -nP -iTCP:9012 -sTCP:LISTEN
```

## FAQ

### The container exits immediately.

Check `docker compose logs npanel`. Common causes are invalid config, an unreachable database, unreachable Redis, or occupied host ports.

### My config changes did not take effect.

The backend reads config at startup. Restart the backend container:

```bash
docker compose restart npanel
```

### Port 8081 or 9012 is already in use.

Use host port overrides:

```bash
NPANEL_HTTP_PORT=18081 NPANEL_GRPC_PORT=19012 docker compose up -d --build
```

### How do I back up the deployment?

Back up the database and the config file. With Compose, also keep a copy of:

```text
configs/config.docker.yaml
docker-compose.yml
configs/config.postgres.yaml
docker-compose.postgres.yml
```

## License

MIT

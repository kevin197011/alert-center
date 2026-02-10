# Alert Center

Enterprise alert rule and notification management platform: multi-channel alerts, Prometheus/VictoriaMetrics integration, SLA, on-call, and RBAC.

## Features

- **Alert rules**: Expressions, severity, labels, templates; bind to channels and data sources
- **Channels**: Lark, Telegram, email, webhook
- **Data sources**: Prometheus / VictoriaMetrics with health checks
- **Silences**: Time windows and matchers
- **SLA**: Response/resolution targets; breach tracking and notifications
- **On-call**: Schedules, rotations, assignments, escalation, reports
- **Tickets**: Optional link to alerts; status and assignee
- **Real-time**: WebSocket push for live alerts
- **Auth**: JWT + RBAC (admin / manager / user); audit logs

## Tech Stack

| Layer   | Stack |
|--------|--------|
| Backend | Go 1.21, Gin, pgx, JWT, Viper, Zap, Swagger |
| Frontend | React 18, TypeScript, Vite, Ant Design 5, Zustand, TanStack Query |
| Data     | PostgreSQL 15, Redis 7 |

## Quick Start (Docker Compose)

**Prerequisites**: Docker and Docker Compose.

```bash
git clone <repo-url>
cd alert-center
docker-compose up -d
```

- **Web UI**: http://localhost:3000  
- **API**: http://localhost:8080 (e.g. `/health`, `/api/v1/*`)  
- **Swagger**: http://localhost:8080/swagger/index.html  

Web container proxies `/api` to the API service; no extra proxy needed.

### Environment (API)

API is configured via environment variables in `docker-compose.yml` (overriding `config.yaml` when set):

| Variable | Description | Default (compose) |
|----------|-------------|-------------------|
| `APP_HOST` | Bind address | `0.0.0.0` |
| `APP_PORT` | HTTP port | `8080` |
| `DATABASE_HOST` | PostgreSQL host | `postgres` |
| `DATABASE_PORT` | PostgreSQL port | `5432` |
| `DATABASE_USERNAME` | DB user | `postgres` |
| `DATABASE_PASSWORD` | DB password | `postgres` |
| `DATABASE_NAME` | DB name | `alert_center` |
| `REDIS_HOST` / `REDIS_PORT` | Redis | `redis` / `6379` |
| `JWT_SECRET` | JWT signing secret | Change in production |

For full options, see `backend/config.yaml.example`.

## Project Layout

```
alert-center/
├── backend/           # Go API + in-process worker
│   ├── cmd/api/       # HTTP server, migrations, worker
│   ├── internal/      # handlers, services, repository, middleware
│   ├── pkg/           # shared packages
│   ├── config.yaml.example
│   └── Dockerfile.api
├── frontend/          # React SPA (Vite)
│   ├── src/pages/    # Dashboard, Rules, Channels, Templates, etc.
│   ├── src/services/ # API client
│   └── Dockerfile.web
├── openspec/          # OpenSpec and project context (openspec/project.md)
├── docker-compose.yml
└── docs/              # Project docs (architecture, API, runbooks, dev)
```

## Development

- **Run locally**: Prefer `docker-compose up` so API, web, Postgres, and Redis run together. Frontend dev server can proxy to `http://localhost:8080` (see `frontend/vite.config.ts`).
- **Backend only**: From `backend/`, copy `config.yaml.example` to `config.yaml`, point DB/Redis to local or containers, then `go run cmd/api/main.go`.
- **Conventions**: See `.cursor/rules` and `openspec/project.md` (code style, commits, tests, docs under `docs/`).

## Documentation

- **Backend**: [backend/README.md](backend/README.md) — API examples, config, build.
- **Project context**: [openspec/project.md](openspec/project.md) — purpose, stack, conventions, domain.
- **Docs index**: `docs/README.md` (when present) — architecture, API, runbooks, dev guides.

## License

MIT.

# Alert Center AI Docs

This document is an AI-friendly, comprehensive description of the Alert Center codebase, including architecture, data model, runtime flows, and key implementation details. It is derived from the repository source and is intended to help automated agents understand and navigate the system.

## 1. System Overview

Alert Center is an enterprise alert rule and notification management platform. It integrates with Prometheus/VictoriaMetrics to evaluate alert rules, sends multi-channel notifications, provides silences, SLA tracking, on-call scheduling, escalation flows, tickets, audit logs, and a real-time WebSocket stream for live updates.

### Core capabilities
- Alert rules: PromQL expressions, severity, labels/annotations, templates, business groups.
- Channels: Lark/Telegram/Webhook (email type is modeled, sending is not currently implemented in channel binding service).
- Data sources: Prometheus/VictoriaMetrics endpoints with health checks.
- Silences: Time-window + label matchers.
- SLA: Configurable response/resolution targets, breach tracking.
- On-call: Schedules, rotations, assignments, escalation.
- Tickets: Optional alert-linked issues.
- Real-time: WebSocket push for alerts, SLA breaches, ticket events.
- Auth: JWT + RBAC.

## 2. Tech Stack

- Backend: Go 1.21, Gin, pgx, Viper, JWT, Zap/standard log, Swagger.
- Frontend: React 18, TypeScript, Vite 5, Ant Design 5, Zustand, TanStack Query, React Router 6.
- Data: PostgreSQL 15, Redis 7.
- Deployment: Docker Compose with nginx front for SPA + API proxy.

## 3. Repository Layout

```
alert-center/
├── backend/                # Go API + in-process worker
│   ├── cmd/api/main.go      # API entry, migrations, worker bootstrap
│   ├── internal/
│   │   ├── handlers/        # HTTP handlers
│   │   ├── services/        # business logic
│   │   ├── repository/      # data access with pgx
│   │   ├── middleware/      # auth, rbac, cors, logging
│   │   └── models/          # domain models
│   ├── pkg/response/        # unified JSON response
│   └── config.yaml.example
├── frontend/               # React SPA
│   ├── src/
│   │   ├── pages/           # route pages
│   │   ├── components/      # UI components
│   │   ├── services/api.ts  # API client
│   │   ├── store/           # Zustand auth
│   │   └── hooks/           # WebSocket hook
│   └── nginx.conf           # SPA + API proxy
├── docker-compose.yml
└── docs/                   # this documentation
```

## 4. Runtime Topology

```
Browser (React SPA)
  ├─ REST: /api/v1/*  ───────> Go API (Gin)
  └─ WebSocket: /api/v1/ws  -> Go API WS hub

Go API
  ├─ PostgreSQL 15 (primary store)
  ├─ Redis 7 (optional cache/queue)
  ├─ Prometheus/VictoriaMetrics (query API)
  └─ Notification channels (Lark/Telegram/Webhook)

In-process worker (same Go process)
  └─ Evaluates rules periodically and sends notifications
```

## 5. Backend Architecture

### 5.1 Entry point
- `backend/cmd/api/main.go`
- Responsibilities:
  - Load config via Viper (file + env).
  - Connect PostgreSQL.
  - Run migrations (inline SQL).
  - Seed defaults (admin user, business groups, template).
  - Initialize repositories/services/handlers.
  - Start HTTP server and worker goroutine.

### 5.2 HTTP routing
- `initRouter` in `main.go`.
- `GET /health` for health checks.
- `GET /swagger/*` for API docs.
- `GET /api/v1/ws` for WebSocket.
- `POST /api/v1/auth/login` public.
- `/api/v1/*` protected by JWT middleware.

### 5.3 Middleware
- `RecoveryMiddleware`, `LoggerMiddleware`, `CORSMiddleware`, `RequestIDMiddleware`.
- `AuthMiddleware` validates JWT, injects `user_id`, `username`, `role`.
- `RoleMiddleware` and `PermissionMiddleware` exist but are not wired in the router by default.

### 5.4 Handlers -> Services -> Repository

Handlers are thin, use `response.Success/Error` for JSON.
Services encapsulate logic and call repositories (pgx) or other services.
Repositories execute SQL against Postgres.

### 5.5 In-process worker
- `startWorker` bootstraps `AlertNotificationWorker`.
- Worker flow:
  1. List enabled alert rules (status=1).
  2. Evaluate each rule via Prometheus/VictoriaMetrics HTTP API.
  3. Apply effective time window and exclusion windows.
  4. Track pending state for `for_duration`.
  5. Create `alert_history` record on firing.
  6. Render template if assigned.
  7. Send notifications via bound channels.
  8. Detect recovery (no longer firing) and mark resolved + notify.

Key files:
- `backend/internal/services/alert_notification_worker.go`
- `backend/internal/services/alert_evaluator.go`
- `backend/internal/services/prometheus_client.go`
- `backend/internal/services/alert_channel_binding_service.go`

## 6. Database Model

Migrations are inline SQL in `backend/cmd/api/main.go`.

Core tables:
- `users` – accounts, roles, status, last_login.
- `business_groups` – hierarchy for ownership.
- `alert_rules` – rule definition and evaluation windows.
- `alert_channels` – channel configs and type.
- `alert_channel_bindings` – rule-to-channel mapping.
- `alert_templates` – message templates.
- `alert_history` – firing/resolved history.
- `operation_logs` – audit logs.
- `data_sources` – Prometheus/VictoriaMetrics endpoints.
- `alert_silences` – silence windows + matchers.
- `sla_configs`, `alert_slas`, `sla_breaches` – SLA targets and breaches.
- `oncall_*` – schedules, members, assignments, escalations.
- `alert_escalations`, `alert_escalation_logs`, `user_escalations` – alert escalation rules/logs.
- `tickets` – ticketing.

Model definitions: `backend/internal/models/*.go`.

## 7. Core Domain Flows

### 7.1 Alert evaluation and notification
1. Worker fetches enabled rules.
2. For each rule, query data source with PromQL expression.
3. If results > threshold (currently: `value > 0`), construct firing alerts.
4. Track in-memory pending map until `for_duration` is satisfied.
5. Insert `alert_history` row (status=firing).
6. Render template with dynamic label/annotation formatting.
7. Send to bound channels.
8. On recovery, mark history as resolved and send recovery notification.

### 7.2 WebSocket notifications
- `WebSocketHandler` maintains clients and broadcast channel.
- Sends message types: `alert`, `sla_breach`, `ticket`.
- Worker emits `alert` notifications on firing/resolved and SLA breach notifications during checks.
- Frontend hook `useWebSocket` connects to `/api/v1/ws`, shows toast and keeps local lists.

### 7.3 SLA
- SLA configs provide response and resolution targets by severity.
- SLA breaches tracked in `sla_breaches`.
- Handlers/services expose list, stats, and trigger checks.

### 7.4 On-call scheduling
- Schedules stored in `oncall_schedules`.
- Members in `oncall_members`.
- Assignments in `oncall_assignments`.
- APIs: generate rotations, get coverage, validate schedule.

### 7.5 Tickets
- `tickets` table provides create/update/resolve/close.
- WebSocket ticket updates exist.

## 8. API Surface (High Level)

Base path: `/api/v1`.

- Auth: `POST /auth/login`, `GET /profile`.
- Rules: `GET/POST/PUT/DELETE /alert-rules`, `POST /alert-rules/test-expression`.
- Channels: `GET/POST/PUT/DELETE /channels`, `POST /channels/:id/test`.
- Templates: `GET/POST/PUT/DELETE /templates`.
- History: `GET /alert-history`.
- Silences: `GET/POST/PUT/DELETE /silences`, `POST /silences/check`.
- Data sources: `GET/POST/PUT/DELETE /data-sources`, `POST /data-sources/:id/health-check`.
- SLA: `/sla/configs`, `/sla/alerts/:id`, `/sla/report`, `/sla/breaches`.
- On-call: `/oncall/*`.
- Correlation: `/correlation/*`.
- Escalations: `/escalations*`.
- Tickets: `/tickets*`.
- Statistics: `/statistics`, `/dashboard`.
- Audit logs: `/audit-logs`.

## 9. Frontend Architecture

### 9.1 App entry
- `src/main.tsx`: React root, QueryClientProvider, Ant Design ConfigProvider.
- `src/App.tsx`: router + `PrivateRoute` guard based on Zustand token.

### 9.2 API client
- `src/services/api.ts`: Axios instance, injects JWT in Authorization header, redirects to `/login` on 401.
- Contains TypeScript interfaces mirroring backend responses.

### 9.3 State
- `src/store/auth.ts`: Zustand store with persist.

### 9.4 Real-time
- `src/hooks/useWebSocket.ts`: connects to `/api/v1/ws` and stores recent alerts, SLA breaches, tickets.

### 9.5 UI layout
- `src/components/Layout/`: Ant Design layout + menu; supports dark mode and locale toggle.

## 10. Security & Auth

- JWT in `Authorization: Bearer <token>` header.
- Claims include `user_id`, `username`, `role`.
- RBAC permissions defined in middleware but not enforced globally in routes by default.

## 11. Configuration

- Backend config read from `backend/config.yaml` or env.
- `docker-compose.yml` wires env vars for DB, Redis, JWT secret.
- Frontend proxy uses nginx to forward `/api/*` to API container.

## 12. Known Implementation Notes

- Migrations are inline SQL in main process; no separate migration tooling.
- Email channel exists in model but is not currently implemented in channel sender.
- Rule evaluation threshold uses `value > 0`; no per-rule threshold expression parser yet.
- `data_sources` table exists, but worker evaluation uses rule `data_source_url` directly.

## 13. Key Files Index (Absolute Paths)

- Entry + router + migrations: `/Users/kevin/projects/src/alert-center/backend/cmd/api/main.go`
- Services: `/Users/kevin/projects/src/alert-center/backend/internal/services/`
- Handlers: `/Users/kevin/projects/src/alert-center/backend/internal/handlers/`
- Repositories: `/Users/kevin/projects/src/alert-center/backend/internal/repository/repository.go`
- Middleware: `/Users/kevin/projects/src/alert-center/backend/internal/middleware/`
- Models: `/Users/kevin/projects/src/alert-center/backend/internal/models/`
- Frontend app entry: `/Users/kevin/projects/src/alert-center/frontend/src/main.tsx`
- Frontend router: `/Users/kevin/projects/src/alert-center/frontend/src/App.tsx`
- Frontend API client: `/Users/kevin/projects/src/alert-center/frontend/src/services/api.ts`
- WebSocket hook: `/Users/kevin/projects/src/alert-center/frontend/src/hooks/useWebSocket.ts`
- Nginx proxy: `/Users/kevin/projects/src/alert-center/frontend/nginx.conf`

## 14. How to Extend

- New domain behavior: add repository, service, handler; wire route in `initRouter`.
- New data model: add SQL in `runMigrations`.
- New channel type: add config schema + implement send function in `alert_channel_binding_service.go`.
- Frontend: add page under `src/pages/`, route in `App.tsx`, API in `services/api.ts`.



## 15. Integration Test Scripts

Integration smoke tests are split into fast/slow stages for CI.

Scripts:
- `/Users/kevin/projects/src/alert-center/scripts/integration_smoke_fast.py`
- `/Users/kevin/projects/src/alert-center/scripts/integration_smoke_slow.py`
- `/Users/kevin/projects/src/alert-center/scripts/integration_smoke.py` (wrapper)

Fast stage (CRUD + channel connectivity):
- Templates CRUD
- Channels: webhook, Lark, Telegram (using local stub endpoints)
- Data sources CRUD + health check (read-only)
- Silences CRUD + check
- Tickets CRUD

Slow stage (business chain):
- Stub Prometheus + webhook receiver
- Create SLA config (0-minute thresholds for immediate breach)
- Create template + channel + rule and bind
- Wait for firing -> verify alert history + webhook delivery
- Switch to recovery -> verify resolved history + webhook delivery
- Trigger SLA breach check -> verify breach records
- Capture WebSocket notifications for alert lifecycle

Usage:
```bash
# run both
python3 /Users/kevin/projects/src/alert-center/scripts/integration_smoke.py all

# fast only
python3 /Users/kevin/projects/src/alert-center/scripts/integration_smoke.py fast

# slow only
python3 /Users/kevin/projects/src/alert-center/scripts/integration_smoke.py slow
```

Notes:
- Prometheus is stubbed; the scripts do not write to any real Prometheus endpoint.
- Telegram tests use `api_base` in channel config to target a local stub; no external network calls.
Lark tests use a local stub webhook that returns `{code:0}`.

## 16. UI Smoke Tests

UI smoke tests validate login + key pages load, and table/rendering checks for Dashboard + Statistics.

Location:
- `/Users/kevin/projects/src/alert-center/frontend/tests/ui-smoke.spec.ts`

Run (from frontend):
```bash
cd /Users/kevin/projects/src/alert-center/frontend
npm install
npx playwright install chromium
npm run test:ui-smoke
```

Env overrides:
- `UI_BASE_URL` (default `http://localhost:3000`)
- `UI_ADMIN_USER` / `UI_ADMIN_PASS` (defaults `admin` / `admin123`)

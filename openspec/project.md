# Project Context

## Purpose

Alert Center is an enterprise alert rule and notification management platform. It connects to Prometheus/VictoriaMetrics, manages alert rules, channels, silences, escalations, on-call schedules, SLA targets, and tickets. It provides real-time alert push (WebSocket), audit logs, statistics, and RBAC.

## Tech Stack

- **Backend**: Go 1.21, Gin, pgx (PostgreSQL), JWT, Viper, Zap, Swagger
- **Frontend**: React 18, TypeScript, Vite 5, Ant Design 5, Zustand, TanStack Query, React Router 6
- **Data**: PostgreSQL 15, Redis 7
- **Run**: Docker Compose (development and deployment)

## Project Conventions

### Code Style

- **Comments**: English only (code comments and script logs).
- **User-facing text**: Use i18n; do not hardcode UI strings.
- **Commits**: Conventional Commits (feat/fix/docs/refactor/test/chore/ci etc.).
- **Go**: Follow standard Go style; use golangci-lint.
- **TypeScript/React**: ESLint + Prettier; prefer functional components and hooks.

### Architecture Patterns

- **Backend**: Handlers → Services → Repository; config via Viper (file + env); migrations in `main.go` (inline SQL).
- **Frontend**: Page-level routes; API layer in `services/api.ts`; auth state in Zustand; TanStack Query for server state.
- **API**: REST under `/api/v1`; JWT in `Authorization: Bearer <token>`; WebSocket at `/api/v1/ws` for live alerts.
- **Permissions**: RBAC (admin / manager / user); middleware enforces auth and roles.

### Testing Strategy

- Tests should be non-interactive and CI-friendly.
- Test scripts: Ruby 3.1+ (per workspace rules); no interactive prompts.
- Core business logic: aim for ≥80% coverage; critical paths fully covered.
- Use fixtures/factories; avoid order-dependent tests and production data.

### Git Workflow

- Branch per feature/fix; merge via Pull Request.
- Keep main stable; commit messages in Chinese or English (team-consistent).

## Domain Context

- **Alert rule**: Prometheus-style expression, for_duration, severity, labels, annotations; belongs to a business group; can bind to channels and template.
- **Channel**: Lark, Telegram, email, webhook; has type-specific config (e.g. webhook URL).
- **Silence**: Time window + matchers; alerts matching are not sent.
- **SLA**: Response/resolution time targets per severity; breach tracking and notifications.
- **On-call**: Schedules, rotations, assignments, escalation; reports and “who is on call”.
- **Escalation**: Alert handoff between users; accept/reject/resolve flow.
- **Ticket**: Optional link from alert; status, assignee, resolve/close.

## Important Constraints

- **Sensitive data**: No secrets in code; use environment variables or secret managers.
- **Development**: All run and test via Docker Compose; do not run backend/frontend directly on host unless documented.
- **Documentation**: Project docs live under repo root `docs/`; avoid scattered `docs/` in subprojects.

## External Dependencies

- **PostgreSQL**: Primary store for users, rules, channels, history, SLA, on-call, tickets, audit logs.
- **Redis**: Optional; used for caching/queue if configured.
- **Prometheus / VictoriaMetrics**: Data sources for rule evaluation (HTTP query API).
- **Lark / Telegram / SMTP / Webhook**: Notification channels; configured per channel.

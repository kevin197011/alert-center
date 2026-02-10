# Skills 与项目匹配分析

基于你已安装的 Cursor/Codex Skills，对 **alert-center** 项目做一次「是否都合理」的匹配分析：哪些 skill 已用上、哪些尚未落地、以及建议的改进点。

---

## 项目画像（简要）

| 维度 | 现状 |
|------|------|
| 技术栈 | Go (Gin) + React 18 (Vite, Ant Design) + PostgreSQL 15 + Redis 7 |
| 部署 | Docker Compose，无 Kubernetes/Ansible |
| CI/CD | **无** `.github/workflows`，无自动化流水线 |
| 敏感信息 | `docker-compose.yml` 内写死 DB 密码、JWT_SECRET；无 `env_file` / `.env` |
| 可观测 | 后端有 `/metrics`（Prometheus），**无** Grafana 配置、**无** Sentry |
| 测试 | Ruby 冒烟脚本、Playwright UI 冒烟；无 CI 自动跑 |
| 文档 | `docs/` 有架构与 AI 文档，无 Runbook/API 开放文档 |

---

## 一、Skill 与项目匹配度总览

| Skill | 匹配度 | 说明 |
|-------|--------|------|
| **senior-devops** | ⚠️ 部分 | 容器化已用；缺 CI、缺规范化的密钥与部署流程 |
| **grafana-dashboards** | 🔵 储备 | 项目有 Prometheus，暂无 Grafana；适合后续做大盘时用 |
| **ansible-automation** | 🔵 储备 | 当前未用 Ansible；若以后做主机/多机部署再启用 |
| **agent-browser** | ✅ 已用 | Playwright 做 UI 冒烟，与 skill 能力一致 |
| **frontend-design / web-design-guidelines / ui-ux-pro-max** | ✅ 已用 | React + Ant Design 前端，做界面与设计时都会用到 |
| **vercel-react-best-practices** | ✅ 部分 | 技术栈是 Vite+React 非 Next，但 React 性能与写法规范通用 |
| **gh-fix-ci** | 🟡 待落地 | 仓库尚无 GitHub Actions；**加上 CI 后**该 skill 才真正有用 |
| **security-best-practices** | ⚠️ 部分 | 与运维/密钥/权限强相关，但项目存在明显安全缺口（见下） |
| **sentry** | 🔵 储备 | 未集成 Sentry；若接入错误监控，该 skill 可指导实现 |
| **doc** | ✅ 已用 | 文档集中在 `docs/`，与规范一致；可进一步用 skill 做 Runbook/API 文档 |
| **skill-creator / create-rule** | ✅ 已用 | 已有 `.cursor/rules` 与自定义 skill，用于扩展约束 |

---

## 二、结论：是否都合理？

- **整体合理**：你的 skills 覆盖了「智能运维 + 告警中心」所需的几块：DevOps、前端、安全、文档、可观测（Grafana/Sentry 偏储备）、CI（gh-fix-ci 待 CI 落地）。没有明显「多余」或与项目无关的 skill。
- **主要缺口在「项目实践」**：部分 skills（如 senior-devops、security-best-practices、gh-fix-ci）的价值要在**项目补齐 CI、密钥管理、可观测**之后才能完全发挥。

因此：**Skills 组合合理；建议在项目侧做少量对齐改进，让这些 skills 真正约束和辅助开发。**

---

## 三、按 Skill 的改进建议

### 1. senior-devops

- **现状**：Docker Compose 已用；无 CI、无流水线、密钥写在 compose 里。
- **建议**：
  - 增加 **GitHub Actions**（或其它 CI）：至少「构建 + 单元/冒烟测试」自动化，便于后续用 **gh-fix-ci** 修 CI。
  - 敏感配置改为 **环境变量 + `env_file`**，compose 中不写死密码/JWT；与 **security-best-practices** 一致。

### 2. security-best-practices

- **现状**：`docker-compose.yml` 中 `POSTGRES_PASSWORD`、`JWT_SECRET` 明文；`config.yaml.example` 占位符合理，但运行时若直接改 `config.yaml` 易把密钥提交进库。
- **建议**：
  - 使用 **`.env` + `env_file`** 注入 DB 密码、JWT secret、Lark/Telegram 等；将 `.env` 加入 `.gitignore`，仓库内保留 `.env.example`。
  - 冒烟测试中的默认账号密码（如 `admin/admin123`）仅限本地/测试环境，并在文档中说明「上线必须修改」。

### 3. gh-fix-ci

- **现状**：无 `.github/workflows`，CI skill 暂无用武之地。
- **建议**：新增至少一个 workflow（如 `ci.yml`）：检出 → 构建 backend/frontend → 跑测试（Go test、Ruby 冒烟、可选 Playwright）。之后 CI 出问题时即可用 **gh-fix-ci** 规范修复。

### 4. grafana-dashboards

- **现状**：有 Prometheus `/metrics`，无 Grafana。
- **建议**：若计划做运维大盘，可在仓库加 `grafana/provisioning` 或单独 repo 管理 dashboard JSON；届时 **grafana-dashboards** skill 可指导设计与命名。

### 5. sentry

- **现状**：未集成 Sentry。
- **建议**：若需要错误追踪与告警联动，可在后端/前端接入 Sentry；**sentry** skill 可用来约束集成方式与错误分类。

### 6. doc

- **现状**：`docs/` 结构清晰，有架构说明与 AI 文档。
- **建议**：需要 Runbook、API 开放文档或对外 .docx 时，用 **doc** skill 统一风格与模板。

### 7. 其它（agent-browser、frontend、vercel-react、create-rule）

- 与当前技术栈和用法一致，**保持即可**。若引入 Next.js 或 Vercel 部署，可再强化 vercel-react-best-practices 与对应 deploy skill。

---

## 四、优先可做的 3 件事（与 skills 对齐）

1. **加 CI**：在 `.github/workflows` 增加一个最小可行 CI（build + test），便于 **gh-fix-ci** 与 **senior-devops** 发挥作用。
2. **密钥外置**：compose 与 backend 通过 `env_file`/环境变量读取 DB、JWT、通道密钥，并增加 `.env.example`；符合 **security-best-practices**。
3. **（可选）可观测**：按需接入 Sentry 或增加 Grafana 配置，让 **sentry** / **grafana-dashboards** 从「储备」变为「在用」。

---

## 五、总结表

| 维度 | 与 Skills 的匹配 | 建议 |
|------|------------------|------|
| 技能组合 | 合理，无多余 | 保持 |
| CI/CD | 缺 CI，gh-fix-ci 未用上 | 增加 GitHub Actions |
| 安全 | 密钥在 compose 内，security skill 未完全落地 | 用 .env + env_file |
| 可观测 | 有 Prometheus，无 Grafana/Sentry | 按需接 Sentry/Grafana |
| 文档与前端 | 已与 doc / frontend / react skills 对齐 | 继续用现有规范 |

整体而言：**你的 skills 选型对智能运维和告警中心项目是合理的；通过补上 CI、密钥管理和（可选）可观测，可以让这些 skills 更好地约束和辅助开发。**

# 智能运维开发 — Skills 推荐与安装指南

本文档针对「智能运维 / AIOps」开发背景，整理适合本地安装的 Cursor/Codex Skills，用于约束和辅助告警中心等运维系统开发。

---

## 本次已安装（openai/skills → ~/.cursor/skills）

| Skill | 说明 |
|-------|------|
| **gh-fix-ci** | 修复 GitHub CI 配置与流水线失败 |
| **security-best-practices** | 安全最佳实践 |
| **sentry** | Sentry 错误监控与集成 |
| **doc** | 文档编写与 .docx 维护 |

**使用前请重启 Cursor** 以加载新 skills。

---

## 一、你当前已具备的 Skills（.cursor/skills）

| Skill | 用途 |
|-------|------|
| **senior-devops** | CI/CD、基础设施即代码、容器化、云平台、流水线、监控 |
| **grafana-dashboards** | Grafana 大盘与可观测性可视化 |
| **ansible-automation** | Ansible 剧本、角色、配置管理、部署与补丁 |
| **agent-browser** | 浏览器自动化、表单、截图、E2E |
| **frontend-design** | 前端界面与设计 |
| **vercel-react-best-practices** | React/Next 性能与最佳实践 |
| **web-design-guidelines** | UI/UX、可访问性、设计审计 |
| **skill-creator** | 自定义 Skill 编写 |
| **create-rule** | Cursor 规则编写 |

以上已覆盖：DevOps 流水线、监控大盘、自动化、前端与规则扩展，和告警中心技术栈（Go + React + Prometheus + 多通道通知）高度相关。

---

## 二、从 openai/skills 推荐新增（适合智能运维）

以下来自 [openai/skills](https://github.com/openai/skills) 的 **curated** 列表，建议按需安装到本地。

| Skill | 说明 | 推荐理由 |
|-------|------|----------|
| **gh-fix-ci** | 修复 GitHub CI 配置与流水线失败 | 与 CI/CD、流水线排障强相关，约束“改 CI 就按规范来” |
| **security-best-practices** | 安全最佳实践 | 运维系统涉及密钥、权限、审计，需安全约束 |
| **sentry** | Sentry 错误监控与集成 | 与告警/可观测性场景一致，可作错误源与告警联动参考 |
| **doc** | 文档编写与维护 | 运维系统需 Runbook、API 文档、架构说明，统一文档风格 |

可选（按技术栈选）：

- **cloudflare-deploy** / **vercel-deploy** / **render-deploy** / **netlify-deploy**：若前端或边缘部署走这些平台，可装对应 deploy skill。
- **security-threat-model** / **security-ownership-map**：做安全与责任矩阵时可装。

---

## 三、安装到本地（Codex / Cursor）

默认安装目录为 `$CODEX_HOME/skills`（未设置时一般为 `~/.codex/skills`）。  
若希望 Cursor 也使用同一批 skills，可把 `CODEX_HOME` 指到 Cursor 的 skills 目录，或在 Cursor 的 skills 目录下建符号链接。

### 3.1 使用官方安装脚本（推荐）

在项目根或任意目录执行（需 Python 3，且脚本在 `$CODEX_HOME/skills/.system/skill-installer/scripts/` 下）：

```bash
# 安装到 Cursor（推荐，--dest 指定目录）
INSTALLER="$HOME/.codex/skills/.system/skill-installer/scripts/install-skill-from-github.py"

python3 "$INSTALLER" --repo openai/skills --path skills/.curated/gh-fix-ci --dest "$HOME/.cursor/skills"
python3 "$INSTALLER" --repo openai/skills --path skills/.curated/security-best-practices --dest "$HOME/.cursor/skills"
python3 "$INSTALLER" --repo openai/skills --path skills/.curated/sentry --dest "$HOME/.cursor/skills"
python3 "$INSTALLER" --repo openai/skills --path skills/.curated/doc --dest "$HOME/.cursor/skills"
```

若只装部分，按需执行上面某一行即可。

### 3.2 安装到 Codex 默认目录（~/.codex/skills）

不指定 `--dest` 时，会安装到 `$CODEX_HOME/skills`（默认 `~/.codex/skills`）：

```bash
python3 "$HOME/.codex/skills/.system/skill-installer/scripts/install-skill-from-github.py" \
  --repo openai/skills --path skills/.curated/gh-fix-ci
# 重复 --path 可一次装多个，或多次执行上述命令
```

### 3.3 安装后

- **Codex**：重启 Codex 以加载新 skills。
- **Cursor**：若 skills 在 Cursor 目录或通过链接被 Cursor 识别，重启 Cursor 后即可生效。

---

## 四、用 Cursor 规则进一步约束「智能运维」开发

除 Skills 外，可在项目里用 `.cursor/rules` 或 `AGENTS.md` 约定：

- **Prometheus/告警**：写 PromQL、告警规则、抑制与路由时，引用项目内已有规则与命名规范。
- **运维安全**：密钥/凭证不落库、最小权限、审计日志与敏感字段脱敏。
- **可观测性**：新增服务/接口时，约定指标、日志与告警级别（与现有 grafana-dashboards、告警中心一致）。

这样 Skills 负责「能力」，规则负责「项目内约束」，一起支撑智能运维开发大业。

---

## 五、参考

- openai/skills 仓库: https://github.com/openai/skills  
- Cursor as an AI SRE: https://blog.niradler.com/cursor-as-an-ai-sre  
- 本仓库通用开发规范: [00-general-development-rules.mdc](.cursor/rules/00-general-development-rules.mdc)

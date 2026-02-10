# Alert Center - 告警规则管理平台

<div align="center">

![Alert Center](https://img.shields.io/badge/Alert-Center-blue)
![Go](https://img.shields.io/badge/Go-1.21-blue)
![React](https://img.shields.io/badge/React-18-blue)
![Ant Design](https://img.shields.io/badge/Ant%20Design-5.0-blue)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-blue)

</div>

## 📋 项目简介

Alert Center 是一个企业级告警规则综合管理平台，支持多种告警渠道配置、Prometheus/VictoriaMetrics 对接、业务组权限划分等功能。

## ✨ 核心特性

- **多渠道告警**: 支持飞书、Telegram、邮件、Webhook 等多种告警渠道
- **自定义模板**: 灵活的告警模板配置，支持 Markdown/Text/HTML
- **数据源管理**: Prometheus、VictoriaMetrics 多数据源配置和健康检查
- **告警统计**: 每日趋势、级别分布、TOP 活跃规则统计
- **业务组管理**: 按业务组划分告警规则权限
- **实时告警**: WebSocket 实时推送告警通知
- **报表导出**: 支持告警数据和审计日志导出
- **用户管理**: 完整的用户 CRUD、角色分配
- **审计日志**: 完整的操作日志记录和查询
- **认证授权**: JWT 认证 + RBAC 权限控制

## 🏗️ 技术架构

### 后端 (Golang)
- **框架**: Gin
- **数据库**: PostgreSQL 15
- **ORM**: pgx
- **认证**: JWT

### 前端 (React)
- **框架**: React 18 + TypeScript
- **UI 组件**: Ant Design 5
- **状态管理**: Zustand
- **数据获取**: TanStack Query

## 🚀 快速开始

### 前置条件

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- npm/yarn/pnpm

### 1. 克隆项目

```bash
git clone https://github.com/your-repo/alert-center.git
cd alert-center
```

### 2. 后端配置

```bash
# 复制配置模板
cp config.yaml.example config.yaml

# 修改配置
vim config.yaml

# 安装依赖
go mod tidy

# 运行服务
go run cmd/api/main.go
```

### 3. 前端配置

```bash
cd alert-center-web

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

### 4. Docker 部署

```bash
# 构建并启动所有服务
docker-compose up -d
```

## 📁 项目结构

```
alert-center/
├── cmd/
│   └── api/                    # API 服务入口
├── internal/
│   ├── config/                 # 配置加载
│   ├── handlers/               # HTTP 处理器
│   ├── middleware/             # 中间件
│   ├── models/                 # 数据模型
│   ├── repository/             # 数据访问层
│   └── services/               # 业务逻辑层
├── pkg/                        # 公共包
├── migrations/                 # 数据库迁移
├── deployments/                # Docker/K8s 部署
├── docs/                       # 文档
├── alert-center-web/           # 前端项目
│   ├── src/
│   │   ├── components/        # 组件
│   │   ├── pages/             # 页面
│   │   ├── services/          # API 服务
│   │   ├── store/             # 状态管理
│   │   ├── hooks/             # 自定义 Hooks
│   │   └── utils/             # 工具函数
│   └── public/                # 静态资源
└── config.yaml               # 配置文件
```

## 📖 API 文档

### 认证

```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

### 告警规则

```bash
# 列表
GET /api/v1/alert-rules?page=1&page_size=10&severity=critical

# 创建
POST /api/v1/alert-rules
{
  "name": "CPU 使用率告警",
  "expression": "rate(cpu_usage[5m]) > 0.8",
  "severity": "warning",
  "group_id": "uuid"
}

# 更新
PUT /api/v1/alert-rules/{id}

# 删除
DELETE /api/v1/alert-rules/{id}

# 导出
GET /api/v1/alert-rules/export
```

### 告警渠道

```bash
# 列表
GET /api/v1/channels

# 创建
POST /api/v1/channels
{
  "name": "飞书告警机器人",
  "type": "lark",
  "config": {
    "webhook_url": "https://..."
  }
}
```

## 🎨 界面预览

### 仪表盘
- 实时告警统计
- 告警趋势图表
- 快速操作入口

### 告警规则管理
- 规则列表与筛选
- 可视化规则配置
- 批量操作支持

### 告警渠道配置
- 多渠道独立配置
- 模板变量支持
- 测试发送功能

## 🔐 权限模型

| 角色 | 描述 |
|------|------|
| admin | 系统管理员 |
| manager | 业务组管理员 |
| user | 普通用户 |

## 📊 监控集成

### Prometheus

```yaml
scrape_configs:
  - job_name: 'alert-center'
    static_configs:
      - targets: ['alert-center:8080']
```

### VictoriaMetrics

直接配置数据源 URL 即可自动对接。

## 🧪 测试

```bash
# 后端测试
go test ./...

# 前端测试
cd alert-center-web
npm run test
```

## 📦 发布

```bash
# 构建后端
go build -o bin/api cmd/api/main.go

# 构建前端
cd alert-center-web
npm run build
```

## 🤝 贡献指南

1. Fork 本仓库
2. 创建分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT License - 详见 LICENSE 文件。

## 🆘 支持

如有问题，请提交 Issue 或联系维护团队。

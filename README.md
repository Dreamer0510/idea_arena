# Idea Arena — AI 创业点子辩论引擎

AI 驱动的创业点子辩论与评估系统。通过多 Agent 辩论机制（创想者 → 审判官 → 裁判 → 终极仲裁），对创业方向进行多轮迭代打磨，最终输出结构化可行性报告。

## 技术栈

| 层 | 技术 |
|---|---|
| **后端** | Go 1.21, Kratos v2, GORM, SQLite/MySQL, JWT |
| **前端** | Next.js 15 (App Router), TailwindCSS, Zustand, Recharts, Lucide |
| **部署** | Docker, docker-compose, Nginx, 一键部署脚本 |

## 项目结构

```
idea_arena/
├── backend/                     # Go Kratos 后端
│   ├── cmd/server/              # 程序入口
│   ├── configs/
│   │   ├── config.example.yaml  # 配置模板（已脱敏）
│   │   └── config.yaml          # 实际配置（gitignore）
│   └── internal/
│       ├── biz/                 # 业务逻辑层（辩论、Idea、用户）
│       ├── conf/                # 配置解析
│       ├── data/                # 数据访问层（GORM models & repos）
│       ├── server/              # HTTP/gRPC 服务器
│       ├── service/             # API 路由处理
│       │   ├── auth.go          # 认证（登录/登出/改密/当前用户）
│       │   ├── idea.go          # Idea CRUD
│       │   ├── debate.go        # 辩论控制 & SSE 推送
│       │   ├── admin.go         # 后台管理（设置/关键词/插件/标签/话题发现）
│       │   ├── admin_ext.go     # 扩展管理（统计/模型配置/Agent/用户/发布）
│       │   ├── admin_provider.go# 服务商管理
│       │   ├── queue.go         # 话题队列
│       │   └── registry.go      # 健康检查 & 服务注册
│       └── pkg/
│           ├── llm/             # LLM 调用（多服务商、Agent 定义）
│           ├── auth/            # JWT 中间件
│           └── search/          # 多源搜索（Bing/百度/Tavily/Serper）
├── frontend/                    # Next.js 15 前端
│   ├── .env.example             # 环境变量模板
│   └── src/
│       ├── app/
│       │   ├── (auth)/login/    # 登录页
│       │   ├── (main)/          # 用户端页面
│       │   │   ├── page.tsx     # 首页（提交点子）
│       │   │   ├── ideas/       # 点子列表 & 详情页
│       │   │   └── monitor/     # 实时监控大盘
│       │   └── admin/           # 管理后台
│       │       ├── page.tsx     # 仪表盘（统计概览）
│       │       ├── providers/   # 服务商管理
│       │       ├── agents/      # Agent 人设配置
│       │       ├── models/      # 模型配置
│       │       ├── ideas/       # Idea 管理
│       │       ├── debates/     # 辩论管理
│       │       ├── discovery/   # 话题发现
│       │       ├── publishing/  # 内容发布
│       │       ├── users/       # 用户管理
│       │       ├── settings/    # 系统设置
│       │       └── account/     # 账号管理（修改密码）
│       ├── components/          # UI 组件
│       ├── hooks/               # React Hooks
│       ├── lib/                 # 工具库（API Client 等）
│       ├── stores/              # Zustand 状态管理
│       └── types/               # TypeScript 类型
├── proto/                       # Protobuf 定义
├── docs/                        # 设计文档
├── deployments/                 # 部署配置
├── docker-compose.yaml          # Docker 编排
├── deploy.example.sh            # 部署脚本模板（已脱敏）
├── Makefile                     # 常用命令
└── README.md
```

---

## 快速开始

### 前置要求

- **Go** 1.21+
- **Node.js** 20+
- **Docker** & **docker-compose**（生产部署）

### 1. 克隆项目

```bash
git clone <repo-url>
cd idea_arena
```

### 2. 配置文件

```bash
# 后端配置
cp backend/configs/config.example.yaml backend/configs/config.yaml

# 前端环境变量
cp frontend/.env.example frontend/.env.local
```

编辑 `backend/configs/config.yaml`，填入以下必要信息：

> **💡 推荐**: 如果你还没有稳定的 LLM API，可以试试 [ai123.one](https://ai123.one/) — 聚合了市面上几乎所有主流模型，创建一个 API Key 即可自由切换不同模型，个人实测稳定且性价比不错。

| 配置项 | 说明 | 示例 |
|--------|------|------|
| `llm.base_url` | LLM API 地址 | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| `llm.api_key` | LLM API 密钥 | `sk-xxxx` |
| `llm.default_model` | 默认模型 | `qwen-plus` |
| `llm.models.*` | 各角色专用模型 | 按需配置 |
| `auth.jwt_secret` | JWT 签名密钥 | 修改为随机字符串 |
| `notify.feishu.webhook_url` | 飞书通知（可选） | Webhook 地址 |
| `search.*.api_key` | 搜索引擎密钥（可选） | 按需配置 |

### 3. 开发模式

```bash
# 安装依赖
make install    # go mod tidy + npm install

# 终端 1: 启动后端（监听 :8000）
make dev-backend

# 终端 2: 启动前端（监听 :3000）
make dev-frontend
```

访问 http://localhost:3000

### 4. Docker 一键启动

```bash
# 构建并启动
make up

# 查看日志
make logs

# 停止
make down

# 清理（包括数据）
make clean
```

### 5. 默认账号

首次启动后，系统会自动创建管理员账号：

- **用户名**: `admin`
- **密码**: `admin123`

> 请登录后在「账号管理」页面立即修改密码。

---

## 核心功能

### 多 Agent 辩论系统

| Agent | 角色 | 职责 |
|-------|------|------|
| **创想者** 💥 | proposer | 提出创业方案，回应质疑，迭代改进 |
| **毒舌审判官** ⚖️ | opponent | 犀利审查，提出 TODO，追问细节 |
| **裁判** 🎯 | judge | 独立仲裁 TODO 状态，过程评分 |
| **终极仲裁** ⚖️ | judge | 输出结构化可行性报告和最终裁决 |
| **元数据提取** 📋 | utility | 提取产品名称、描述、市场信息 |
| **辩论摘要** 📝 | utility | 生成辩论过程摘要 |

辩论流程：提案 → 审查 → 反驳/改进 → 裁判评分 → 多轮迭代 → 终极仲裁报告

### 多服务商 LLM 管理

- 支持配置多个 LLM 服务商（OpenAI 兼容 API）
- 可为每个 Agent 独立指定服务商和模型
- 在线刷新模型列表、测试连接
- 支持可搜索的模型下拉选择

### 话题发现

- 多渠道爬虫自动发现 AI 创业话题
- AI 分析推荐有价值的话题
- 一键提交话题进入辩论队列
- 内置插件：`52pojie`、`keyword_search`、`trend_search`、`social_pain`、`academic_frontier`、`funding_signal`、`llm_creative`
- 自动辩论前语义去重阈值：`>0.85` 自动过滤，`0.75~0.85` 人工复核，`<0.75` 自动流程继续
- 统一规则评分：痛点 / 趋势 / 可行 / 变现 / 新颖（0-10）

### 内容发布

- AI 生成社交媒体推文（小红书、公众号等）
- 人工复审、修改、发布工作流

### 监控大盘

- 实时辩论进度可视化
- 评分趋势图表（Recharts）
- 系统运行状态概览

### 其他

- **多源搜索**: Bing + 百度 + Tavily + Serper
- **三维评分**: 可行性 / 经济性 / 利润潜力
- **SSE 实时推送**: 辩论过程实时流式展示
- **暗色/亮色主题**: 一键切换
- **JWT 认证**: 登录、登出、修改密码
- **用户管理**: 管理员可管理用户账号

---

## 管理后台

管理后台位于 `/admin`，提供左侧边栏导航：

| 页面 | 路径 | 功能 |
|------|------|------|
| 仪表盘 | `/admin` | 统计概览（Idea 数、辩论数、通过率等） |
| 服务商管理 | `/admin/providers` | 添加/编辑 LLM 服务商，测试连接，刷新模型 |
| Agent 人设 | `/admin/agents` | 配置各 Agent 的名称、人设提示词、温度、模型 |
| 模型配置 | `/admin/models` | 全局默认模型和角色模型映射 |
| Idea 管理 | `/admin/ideas` | 查看/管理所有 Idea |
| 辩论管理 | `/admin/debates` | 查看辩论历史和状态 |
| 话题发现 | `/admin/discovery` | 话题爬取、AI 推荐、提交辩论 |
| 内容发布 | `/admin/publishing` | AI 生成推文、复审发布 |
| 用户管理 | `/admin/users` | 管理用户账号和角色 |
| 系统设置 | `/admin/settings` | 搜索关键词、插件配置等 |
| 账号管理 | `/admin/account` | 修改当前用户密码 |

---

## API 参考

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/login` | 登录 |
| POST | `/api/v1/auth/logout` | 登出 |
| GET | `/api/v1/auth/me` | 当前用户信息 |
| PUT | `/api/v1/auth/password` | 修改密码 |

### Idea & 辩论

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/ideas` | Idea 列表 |
| GET | `/api/v1/ideas/{id}` | Idea 详情 |
| POST | `/api/v1/ideas` | 创建 Idea |
| PUT | `/api/v1/ideas/{id}` | 更新 Idea |
| DELETE | `/api/v1/ideas/{id}` | 删除 Idea |
| POST | `/api/v1/debate/{id}/start` | 启动辩论 |
| POST | `/api/v1/debate/{id}/intervene` | 人工介入辩论 |
| GET | `/api/v1/debate/{id}/status` | 辩论状态 |
| GET | `/api/v1/debate/{id}/stream` | 辩论 SSE 流 |

### 话题队列

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/queue/submit` | 提交话题 |
| GET | `/api/v1/queue/list` | 队列列表 |

### 管理后台

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/stats/overview` | 统计概览 |
| GET/PUT | `/api/v1/admin/settings` | 系统设置 |
| CRUD | `/api/v1/admin/keywords` | 搜索关键词 |
| GET/PUT | `/api/v1/admin/plugins/{name}/toggle` | 插件管理 |
| CRUD | `/api/v1/admin/tags` | 标签管理 |
| POST | `/api/v1/admin/discovery/run` | 手动话题发现 |
| POST | `/api/v1/admin/discovery/auto` | 自动话题发现 |
| GET | `/api/v1/admin/topics` | 话题列表 |
| POST | `/api/v1/admin/topics/{id}/submit` | 提交话题辩论 |
| GET/PUT | `/api/v1/admin/models/config` | 模型配置 |
| POST | `/api/v1/admin/models/test` | 测试模型连接 |
| GET/PUT | `/api/v1/admin/agents` | Agent 配置 |
| CRUD | `/api/v1/admin/providers` | 服务商管理 |
| POST | `/api/v1/admin/providers/{id}/test` | 测试服务商 |
| POST | `/api/v1/admin/providers/{id}/models` | 刷新模型列表 |
| GET | `/api/v1/admin/providers/{id}/models` | 获取模型列表 |
| CRUD | `/api/v1/admin/users` | 用户管理 |
| CRUD | `/api/v1/admin/posts` | 发布管理 |
| POST | `/api/v1/admin/posts/generate` | AI 生成推文 |

### 系统

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/healthz` | 健康检查 |
| GET | `/api/v1/registry/info` | 项目信息 |
| GET | `/api/v1/registry/routes` | 路由列表 |

---

## 服务器部署

### 一键部署

```bash
# 1. 从模板创建部署脚本
cp deploy.example.sh deploy.sh

# 2. 编辑 deploy.sh，填入服务器信息
#    - SERVER_IP: 服务器 IP
#    - SERVER_PASS: 服务器密码
#    - DOMAIN: 域名

# 3. 安装 sshpass（macOS）
brew install hudochenkov/sshpass/sshpass

# 4. 执行部署
bash deploy.sh
```

部署脚本会自动完成：打包 → 上传 → 解压 → 配置管理 → Nginx 配置 → Docker 构建 → 启动 → 健康检查

### 配置管理

部署时配置文件不会被覆盖：

- 配置存储在 `data/config.yaml`（持久化目录）
- 首次部署自动从 `config.example.yaml` 创建
- 后续部署保留已有配置不覆盖

### 数据持久化

| 数据 | 服务器路径 |
|------|-----------|
| SQLite 数据库 | `/opt/idea-arena/data/sqlite/idea_arena.db` |
| MySQL 数据库（Docker） | `/opt/idea-arena/data/mysql/` |
| MySQL 数据库（外部） | 通过 `data.database.source` DSN 接入 |
| 后端配置 | `/opt/idea-arena/data/config.yaml` |
| 后端日志 | `/opt/idea-arena/data/logs/backend/` |

### 运维命令

```bash
# SSH 到服务器后
cd /opt/idea-arena
docker compose ps                # 查看状态
docker compose logs -f backend   # 后端日志
docker compose logs -f frontend  # 前端日志
docker compose restart           # 重启服务
docker compose down && docker compose up -d --build  # 重新构建
# 如启用 MySQL Docker（有 docker-compose.mysql.yaml 时）
docker compose -f docker-compose.yaml -f docker-compose.mysql.yaml ps
```

---

## 配置说明

### 后端配置 (`backend/configs/config.yaml`)

```yaml
server:
  http:
    addr: 0.0.0.0:8000      # HTTP 监听地址
    timeout: 10s

data:
  database:
    driver: sqlite           # sqlite 或 mysql
    source: data/idea_arena.db # sqlite: 文件路径；mysql: DSN

auth:
  jwt_secret: your-secret    # JWT 签名密钥（请修改）
  jwt_expire: 24h

llm:
  base_url: https://...      # LLM API 地址（OpenAI 兼容）
  api_key: sk-xxx            # API 密钥
  default_model: qwen-plus   # 默认模型
  models:                    # 各角色模型（可选，留空使用 default_model）
    proposer: model-a
    opponent: model-b
    judge: model-c
    utility: model-d

debate:
  max_rounds: 20             # 最大辩论轮次
  max_concurrent: 2          # 最大并发辩论数
  timeout: 1800s             # 单次辩论超时
  graduation_score: 7.5      # 毕业分数线

notify:
  feishu:
    enabled: false
    webhook_url: ""          # 飞书机器人 Webhook
```

### 数据库切换（本地 SQLite / 线上 MySQL）

本项目后端支持通过 `backend/configs/config.yaml` 动态切换数据库：

- 本地开发（默认）：
```yaml
data:
  database:
    driver: sqlite
    source: data/idea_arena.db
```
- 线上部署（推荐）：
```yaml
data:
  database:
    driver: mysql
    # Docker 同网络建议：mysql:3306；外部数据库可填 127.0.0.1:3306 或实际地址
    source: user:password@tcp(mysql:3306)/idea_arena?charset=utf8mb4&parseTime=True&loc=Local
```

> 说明：项目中的长文本字段（如 `ideas.debate_log`）在 MySQL 使用 `LONGTEXT`，可存储包含 Emoji 的长内容。

### 前端环境变量 (`frontend/.env.local`)

```env
NEXT_PUBLIC_API_URL=http://localhost:8000
NEXT_PUBLIC_APP_NAME=Idea Arena
NEXT_PUBLIC_APP_DESCRIPTION=AI 创业点子辩论引擎
```

---

## License

MIT

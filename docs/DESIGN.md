# Idea Arena — AI 创业点子辩论引擎

## 项目概述

Idea Arena 是一个 AI 驱动的创业点子辩论与评估系统。通过多 Agent 辩论机制，对创业方向进行多轮迭代打磨，最终输出可行性报告和开发 Prompt。

核心理念：不是让 AI 说"好主意"，而是让多个 AI Agent 互相挑战、反复打磨，用真实搜索数据支撑论点，最终给出客观评分。

## 用户故事

1. **作为用户**，我希望提交一个创业话题，让 AI 系统自动进行多轮辩论，**以便**获得客观的可行性评估
2. **作为用户**，我希望查看所有已评估项目的排行榜（按分数排序），**以便**快速找到高潜力项目
3. **作为用户**，我希望查看每个项目的详细辩论过程和最终报告，**以便**理解评分背后的逻辑
4. **作为用户**，我希望手动提交话题到辩论队列（灵光一现），**以便**评估自己的创业点子
5. **作为管理员**，我希望通过登录系统管理话题和辩论流程，**以便**维护系统运行
6. **作为用户**，我希望获得 AI 推荐的技术栈和开发 Prompt，**以便**快速启动项目开发
7. **作为用户**，我希望收到飞书通知（高分项目 + 每日汇总），**以便**及时了解有价值的项目

## 功能清单

### P0 — MVP 必须

- **多 Agent 辩论引擎**：创想者（提案）→ 审判官（反驳）→ 多轮攻防 → 判官精修 → 最终裁决
- **多源搜索**：Bing + 百度 + Tavily + Serper/Google，用真实数据支撑辩论
- **三维评分**：可行性 / 经济性 / 利润潜力，≥8 分自动毕业
- **多轮迭代**：最多 20 轮攻防，分数达标自动毕业
- **Ideas CRUD**：创建/读取/更新/删除创业点子
- **排行榜页面**：按分数排序展示所有已评估项目
- **详情页面**：展示辩论流程 + 可行性报告
- **用户认证**：JWT 登录/登出，路由鉴权

### P1 — 重要功能

- **并行辩论**：支持最多 2 场辩论同时进行
- **超时重试**：单场辩论超时自动重试，最多 3 次
- **判官精修**：辩论结束后由 AI 判官对方案进行精修优化
- **灵光一现**：手动提交话题，自动去重排队
- **技术选型生成**：辩论完成后异步生成 AI 推荐的技术栈
- **开发 Prompt 生成**：一键生成 Cursor / Windsurf / Claude Code 的项目启动 Prompt
- **暗色/亮色主题切换**

### P2 — 增强功能

- **自动打标**：AI 生成标签 + 向量嵌入
- **相似项目推荐**：基于向量相似度推荐相关项目（Qdrant）
- **飞书通知**：高分项目推送 + 每日汇总报告
- **统一后台管理接口**：预留 ProjectRegistry 接口

## 数据模型

### ideas 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| topic | TEXT | 辩论话题 |
| status | VARCHAR | 状态: pending/debating/graduated/failed |
| proposal | TEXT | 初始提案 |
| debate_log | JSON | 辩论记录（多轮攻防） |
| final_report | TEXT | 最终可行性报告 |
| score_feasibility | FLOAT | 可行性评分 (0-10) |
| score_economics | FLOAT | 经济性评分 (0-10) |
| score_profit | FLOAT | 利润潜力评分 (0-10) |
| score_overall | FLOAT | 综合评分 (0-10) |
| round_count | INT | 辩论轮数 |
| tech_stack | TEXT | AI 推荐技术栈 |
| dev_prompt | TEXT | 开发 Prompt |
| tags | JSON | AI 生成标签 |
| product_name | VARCHAR | 产品名称 |
| one_liner | VARCHAR | 一句话描述 |
| search_data | JSON | 搜索数据缓存 |
| judge_refined | TEXT | 判官精修后的方案 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### users 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| username | VARCHAR | 用户名 |
| password_hash | VARCHAR | 密码哈希 |
| role | VARCHAR | 角色: admin/user |
| created_at | TIMESTAMP | 创建时间 |

### manual_queue 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 自增主键 |
| topic | TEXT | 手动提交的话题 |
| status | VARCHAR | 状态: queued/processing/done |
| submitted_by | BIGINT FK | 提交者 |
| created_at | TIMESTAMP | 创建时间 |

## 页面/路由清单

| 路由 | 页面 | 说明 |
|------|------|------|
| `/` | 首页 | 话题提交 + 系统概览 |
| `/ideas` | 排行榜 | 按分数排序的项目列表 |
| `/ideas/[id]` | 详情页 | 辩论流程 + 报告 + 技术栈 + Prompt |
| `/login` | 登录页 | 用户登录 |

## API 端点清单

### 认证
- `POST /api/v1/auth/login` — 登录
- `POST /api/v1/auth/logout` — 登出
- `GET  /api/v1/auth/me` — 获取当前用户信息

### Ideas
- `GET    /api/v1/ideas` — 列表（支持分页、筛选、排序）
- `GET    /api/v1/ideas/:id` — 详情
- `POST   /api/v1/ideas` — 创建
- `PUT    /api/v1/ideas/:id` — 更新
- `DELETE /api/v1/ideas/:id` — 删除

### 辩论
- `POST /api/v1/debate/start/:id` — 启动辩论
- `GET  /api/v1/debate/status/:id` — 辩论状态（SSE 实时推送）

### 手动队列
- `POST /api/v1/queue/submit` — 提交话题
- `GET  /api/v1/queue/list` — 队列列表

### 系统
- `GET /healthz` — 健康检查
- `GET /api/v1/registry/info` — 项目注册信息（统一后台）

## 技术约束与决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 后端框架 | Go Kratos | 微服务友好，预留统一后台 |
| 前端框架 | Next.js 15 App Router | 与原项目一致，SSR 灵活 |
| UI 组件库 | shadcn/ui + TailwindCSS | 高定制性，符合 frontend-design skill |
| 数据库 | PostgreSQL | 生产级，支持 JSON 字段 |
| 缓存 | Redis | 辩论状态缓存、限流 |
| ORM | GORM | Go 生态最成熟 |
| 认证 | JWT | 无状态，支持多项目 SSO |
| LLM | Anthropic Messages API 兼容 | 可切换 MiniMax/Anthropic/OpenRouter |
| 搜索 | 多源聚合 | Bing + Baidu + Tavily + Serper |

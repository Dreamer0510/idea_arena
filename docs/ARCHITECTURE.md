# Idea Arena — 架构设计文档

## 系统架构总览

```
┌──────────────────────────────────────────────────────────────┐
│                     Nginx (port 80)                          │
│                  反向代理 → 前端 :3000 / 后端 :8000            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐          ┌──────────────────────────┐  │
│  │   Next.js 15     │  HTTP    │   Go Kratos API          │  │
│  │   (Frontend)     │ ◄──────► │   (Backend)              │  │
│  │   port 3000      │  JSON    │   port 8000 (HTTP)       │  │
│  │                  │          │   port 9000 (gRPC)       │  │
│  └──────────────────┘          └────────┬─────────────────┘  │
│                                         │                    │
│                          ┌──────────────┼──────────────┐     │
│                          ▼              ▼              ▼     │
│                    PostgreSQL        Redis       LLM APIs    │
│                    (数据持久化)    (缓存/限流)   (Anthropic等) │
│                                                              │
│                          ┌──────────────────────────┐        │
│                          │   Debate Engine (协程)     │        │
│                          │   运行在 Go 后端进程内      │        │
│                          │   - 话题调度器              │        │
│                          │   - 多 Agent 辩论           │        │
│                          │   - 多源搜索聚合            │        │
│                          └──────────────────────────┘        │
└──────────────────────────────────────────────────────────────┘
```

## 后端模块划分 (Kratos DDD 分层)

```
backend/
├── api/                          # Protobuf 生成的 Go 代码
│   ├── ideaarena/v1/             # 业务 API
│   └── registry/v1/              # 统一后台注册 API
├── cmd/
│   └── server/                   # 程序入口 + Wire DI
├── internal/
│   ├── conf/                     # 配置定义 (Protobuf)
│   ├── biz/                      # 业务逻辑层
│   │   ├── idea.go               # Idea 实体 + IdeaRepo 接口 + IdeaUsecase
│   │   ├── user.go               # User 实体 + UserRepo 接口 + UserUsecase
│   │   ├── debate.go             # Debate 引擎 Usecase
│   │   ├── queue.go              # ManualQueue 实体 + Usecase
│   │   └── search.go             # 搜索聚合 Usecase
│   ├── data/                     # 数据访问层
│   │   ├── data.go               # DB/Redis 初始化
│   │   ├── idea.go               # IdeaRepo 实现 (GORM)
│   │   ├── user.go               # UserRepo 实现
│   │   └── queue.go              # QueueRepo 实现
│   ├── service/                  # gRPC/HTTP handler 实现
│   │   ├── idea.go               # IdeaService
│   │   ├── auth.go               # AuthService
│   │   ├── debate.go             # DebateService (含 SSE)
│   │   ├── queue.go              # QueueService
│   │   └── registry.go           # RegistryService (统一后台)
│   ├── server/                   # 服务器配置
│   │   ├── http.go               # HTTP 服务器 + 中间件
│   │   └── grpc.go               # gRPC 服务器
│   └── pkg/                      # 内部共享包
│       ├── auth/jwt.go           # JWT 工具
│       ├── middleware/           # 中间件
│       └── llm/                  # LLM 客户端
│           ├── client.go         # 通用 LLM 客户端 (Anthropic Messages API)
│           └── agents.go         # Agent Prompt 定义
├── pkg/                          # 可导出公共包
│   └── registry/                 # 统一后台注册
└── migrations/                   # 数据库迁移
```

## 前端模块划分

```
frontend/src/
├── app/
│   ├── (auth)/login/             # 登录页
│   ├── (main)/                   # 主布局（带导航）
│   │   ├── page.tsx              # 首页（话题提交）
│   │   ├── ideas/
│   │   │   ├── page.tsx          # 排行榜
│   │   │   └── [id]/page.tsx     # 详情页
│   │   └── layout.tsx
│   ├── layout.tsx
│   └── providers.tsx
├── components/
│   ├── ui/                       # shadcn/ui
│   ├── layout/                   # 导航、主题切换
│   ├── ideas/                    # Idea 相关组件
│   │   ├── idea-card.tsx         # 项目卡片
│   │   ├── idea-list.tsx         # 项目列表
│   │   ├── debate-timeline.tsx   # 辩论时间线
│   │   ├── score-radar.tsx       # 三维评分雷达图
│   │   └── tech-stack-card.tsx   # 技术栈卡片
│   └── debate/
│       ├── submit-topic.tsx      # 话题提交表单
│       └── debate-status.tsx     # 辩论实时状态 (SSE)
├── lib/
│   ├── api-client.ts             # Axios 实例
│   └── auth.ts                   # Token 管理
├── hooks/
│   └── use-ideas.ts              # Ideas 数据 Hooks (TanStack Query)
├── stores/
│   └── auth-store.ts             # 认证状态
└── types/
    └── api.ts                    # API 类型定义
```

## 统一后台管理接口

每个基于此模板创建的子项目实现 `ProjectRegistry` gRPC 服务：

```protobuf
service ProjectRegistry {
  rpc GetProjectInfo(Empty) returns (ProjectInfo);      // 项目元信息
  rpc HealthCheck(Empty) returns (HealthStatus);        // 健康状态
  rpc ListRoutes(Empty) returns (RouteList);            // 路由清单
  rpc ListPermissions(Empty) returns (PermissionList);  // 权限清单
}
```

统一后台通过此接口发现、监控、管理所有子项目。

## 认证方案

- JWT Token：后端签发，前端 localStorage 存储
- 中间件鉴权：Kratos HTTP middleware 校验 Authorization header
- 默认管理员：系统初始化时自动创建 admin/admin123
- SSO 预留：JWT 密钥可配置为统一签发中心的公钥

## 辩论引擎设计

辩论引擎作为后端进程内的 **后台协程池** 运行：

```
DebateScheduler (定时扫描 pending ideas)
    │
    ├── DebateWorker #1 (goroutine)
    │     ├── SearchAggregator → 多源搜索
    │     ├── ProposerAgent   → 初始提案
    │     ├── OpponentAgent   → 审查反驳
    │     ├── ScorerAgent     → 快速评分
    │     ├── ... (循环最多 20 轮)
    │     ├── JudgeAgent      → 判官精修
    │     └── VerdictAgent    → 最终裁决
    │
    └── DebateWorker #2 (goroutine，最多 2 并行)
```

## 数据库索引策略

```sql
CREATE INDEX idx_ideas_status ON ideas(status);
CREATE INDEX idx_ideas_status_score ON ideas(status, score_overall DESC);
CREATE INDEX idx_ideas_status_created ON ideas(status, created_at DESC);
CREATE INDEX idx_ideas_created ON ideas(created_at DESC);
CREATE UNIQUE INDEX idx_users_username ON users(username);
CREATE INDEX idx_queue_status ON manual_queue(status);
```

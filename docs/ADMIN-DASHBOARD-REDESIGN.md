# Idea Arena 管理后台重设计方案

## 一、设计目标

参考 Sub2API（v2.pincc.ai）的 UI 风格，将现有单页 Tab 式管理后台重构为专业的多页面 Sidebar 导航后台，同时新增以下核心能力：

1. **模型配置** — 后台动态配置各角色使用的 LLM 模型
2. **Agent 人设** — 后台可编辑的 Agent 系统提示词
3. **权限管理** — 用户角色和操作权限
4. **社交推文** — AI 生成 + 人工复审 + 多平台发布
5. **数据仪表盘** — 统计卡片 + 趋势图表

---

## 二、页面结构（Sidebar 导航）

```
/admin
├── /dashboard          # 仪表盘（首页）
├── /ideas              # 点子管理（列表/搜索/操作）
├── /debates            # 辩论监控（实时状态）
├── /discovery          # 话题发现（原 topics + keywords + plugins + tags）
│   ├── topics          # 发现的话题
│   ├── keywords        # 搜索关键词
│   ├── tags            # 约束标签
│   └── plugins         # 渠道插件
├── /models             # 模型配置
│   ├── providers       # API 提供商（base_url, api_key）
│   └── roles           # 角色模型分配
├── /agents             # Agent 人设管理
├── /publishing         # 社交推文运营
│   ├── drafts          # 草稿箱（AI 生成的推文）
│   ├── review          # 待审核
│   ├── published       # 已发布
│   └── channels        # 渠道配置（小红书/公众号等）
├── /users              # 用户管理
└── /settings           # 系统设置
```

---

## 三、UI 布局规范

### 3.1 整体布局
```
┌──────────┬─────────────────────────────────┐
│          │  Header (面包屑 + 搜索 + 用户)   │
│  Sidebar │─────────────────────────────────│
│  (固定)   │                                 │
│  240px   │  Main Content                   │
│          │  (max-w-7xl mx-auto)            │
│          │                                 │
└──────────┴─────────────────────────────────┘
```

### 3.2 设计语言
- **配色**: 主色 Emerald/Teal (参考 sub2api 绿色调)，中性色 Slate
- **圆角**: 卡片 xl (12px)，按钮 lg (8px)，输入框 lg
- **阴影**: 卡片使用 shadow-sm，hover 时 shadow-md
- **字体**: 系统字体栈，标题 font-bold，正文 text-sm
- **间距**: 页面 p-6，卡片内 p-5，卡片间 gap-6

### 3.3 核心组件
- **StatCard** — 带图标的统计卡片（4列网格）
- **DataTable** — 带排序/筛选/分页的表格
- **Sidebar** — 可折叠的左侧导航
- **PageHeader** — 标题 + 描述 + 操作按钮
- **StatusBadge** — 状态标签（graduated/debating/failed 等）

---

## 四、各页面详细设计

### 4.1 Dashboard 仪表盘
**统计卡片（第一行）：**
- 引擎状态（运行中/空闲 + 当前辩论数）
- 今日产出（新增点子数 + 平均分）
- 累计辩论（总数 + 通过率）
- 毕业方案（graduated 数量）

**图表区域：**
- 每日产出趋势（30天折线图）
- 状态分布（饼图：graduated/promising/failed/debating/pending）
- 分数分布（柱状图）

**快捷操作：**
- 提交新点子、运行话题发现、查看最新辩论

### 4.2 模型配置页 `/admin/models`
**Provider 配置：**
| 字段 | 说明 |
|------|------|
| base_url | LLM API 基础 URL |
| api_key | API 密钥（脱敏显示） |
| default_model | 默认模型 |
| max_tokens | 最大 Token |
| timeout | 超时时间 |
| max_retries | 重试次数 |

**角色模型分配（表格/卡片）：**
| 角色 | 模型 | 说明 |
|------|------|------|
| 创想者 (proposer) | kimi-k2.5 | 提案和方案改进 |
| 审判官 (opponent) | glm-5 | 审查和质疑 |
| 裁判/仲裁 (judge) | qwen3.5-plus | 评分和裁决 |
| 工具 (utility) | MiniMax-M2.5 | 元数据/摘要 |

### 4.3 Agent 人设页 `/admin/agents`
每个 Agent 一张卡片，可编辑：
- **名称** + **Emoji**
- **角色标识** (proposer/opponent/judge/utility)
- **系统提示词** (大文本编辑器，支持 Markdown 预览)
- **温度** (Temperature slider)
- **启用/禁用**

### 4.4 社交推文页 `/admin/publishing`
**工作流：**
1. 选择毕业方案 → AI 生成推文草稿
2. 人工编辑/修改
3. 选择目标平台（小红书/公众号/微博）
4. 预览 → 确认发布

**推文列表：**
- 草稿/待审/已发布 三个 Tab
- 每条推文：标题、内容预览、目标平台、关联点子、状态

### 4.5 用户管理页 `/admin/users`
- 用户列表（ID、用户名、角色、创建时间）
- 创建用户
- 修改角色（admin/editor/viewer）
- 禁用/删除用户

---

## 五、后端 API 新增

### 模型配置
```
GET    /api/v1/admin/models/config     # 获取模型配置
PUT    /api/v1/admin/models/config     # 更新模型配置
POST   /api/v1/admin/models/test       # 测试模型连通性
```

### Agent 人设
```
GET    /api/v1/admin/agents            # 获取所有 Agent
GET    /api/v1/admin/agents/:id        # 获取单个 Agent
PUT    /api/v1/admin/agents/:id        # 更新 Agent
POST   /api/v1/admin/agents/:id/reset  # 重置为默认
```

### 社交推文
```
GET    /api/v1/admin/posts             # 推文列表
POST   /api/v1/admin/posts/generate    # AI 生成推文
PUT    /api/v1/admin/posts/:id         # 编辑推文
POST   /api/v1/admin/posts/:id/publish # 发布推文
DELETE /api/v1/admin/posts/:id         # 删除推文
GET    /api/v1/admin/posts/channels    # 发布渠道配置
PUT    /api/v1/admin/posts/channels    # 更新渠道配置
```

### 用户管理
```
GET    /api/v1/admin/users             # 用户列表
POST   /api/v1/admin/users             # 创建用户
PUT    /api/v1/admin/users/:id         # 更新用户
DELETE /api/v1/admin/users/:id         # 删除用户
```

### 统计数据
```
GET    /api/v1/admin/stats/overview    # 仪表盘统计
GET    /api/v1/admin/stats/trends      # 趋势数据
GET    /api/v1/admin/stats/distribution # 分布数据
```

---

## 六、数据库新增表

### agents_config（Agent 人设配置）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | varchar(50) PK | 角色标识 |
| name | varchar(100) | 显示名称 |
| emoji | varchar(10) | Emoji |
| role | varchar(20) | 角色分类 |
| system_prompt | text | 系统提示词 |
| temperature | real | 温度参数 |
| enabled | bool | 是否启用 |

### social_posts（社交推文）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 PK | 推文 ID |
| idea_id | int64 | 关联的 Idea |
| platform | varchar(50) | 目标平台 |
| title | text | 标题 |
| content | text | 内容 |
| status | varchar(20) | draft/review/published/rejected |
| published_at | timestamp | 发布时间 |
| created_at | timestamp | 创建时间 |

### social_channels（发布渠道）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 PK | 渠道 ID |
| platform | varchar(50) | 平台标识 |
| name | varchar(100) | 渠道名称 |
| config | text (JSON) | 渠道配置 |
| enabled | bool | 是否启用 |

---

## 七、实施计划

### Phase 1: 前端骨架重构（Sidebar + 路由 + 共享组件）
- 创建 AdminLayout 组件（Sidebar + Header + Main）
- 拆分路由：/admin/dashboard, /admin/ideas, ...
- 提取共享组件：StatCard, DataTable, PageHeader, StatusBadge
- 保留原有功能，仅改变布局

### Phase 2: Dashboard 仪表盘
- 新增后端统计 API
- 前端统计卡片 + 图表

### Phase 3: 模型 + Agent 配置
- 后端新增 API + 数据库表
- 前端配置页面
- Agent 人设从硬编码迁移到数据库

### Phase 4: 权限系统
- 后端 RBAC 中间件
- 用户管理页面

### Phase 5: 社交推文系统
- 后端推文 CRUD + AI 生成
- 前端推文管理页面
- 渠道配置

### Phase 6: 优化和测试
- UI 打磨和响应式适配
- 端到端测试

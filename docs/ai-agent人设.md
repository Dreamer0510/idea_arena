# Idea Arena — Agent 人设与模型选型方案 (v2)

> 最后更新: 2026-03-08 | commit: 2bb7f0f

## 一、执行链路总览

```
插件爬虫(9渠道) → 规则打分 → 语义去重 → [话题战略官]精选 → 创建Idea
→ 多源搜索 → [创想者]提案 → [元数据提取]异步
→ 多轮辩论循环(最多20轮):
    [审判官]审查+TODO → [创想者]回应 → [裁判]评分+裁定
→ [终极仲裁]可行性报告 → [辩论摘要] → 三档判定
```

## 二、Agent 名册与模型配置

所有 Agent 通过 `agent_configs` 表配置，统一走云雾API (yunwu.ai, provider_id=2)。

| Agent ID | 名称 | 模型 | Temp | 职责 |
|----------|------|------|------|------|
| `proposer` | 创想者 💥 | `claude-sonnet-4-6` | 0.85 | 创意商业方案生成，中英文双语处理 |
| `opponent` | 审判官 ⚖️ | `deepseek-v3.2` | 0.6 | 6维审查(含AI技术深度)，TODO管理 |
| `referee` | 裁判 🎯 | `gpt-4o` | 0.3 | 评分+TODO裁定，强制校验一致性 |
| `judge` | 终极仲裁 ⚖️ | `claude-sonnet-4-6` | 0.7 | 综合报告含AI实施评估 |
| `topic_strategist` | 话题战略官 🎯 | `deepseek-v3.2` | 0.5 | 翻译+AI赋能评估+精选决策 |
| `meta_extractor` | 元数据提取 📋 | `gpt-4o-mini` | 0.1 | JSON结构化提取 |
| `debate_summarizer` | 辩论摘要 📝 | `deepseek-v3.2-fast` | 0.3 | 100-200字精炼摘要 |

## 三、核心人设优化点 (v2 vs v1)

### 创想者
- **+语言规则**: 英文源自动本地化，不逐字翻译
- **+AI赋能思维**: 非AI话题必须找AI切入点，每个提案含"AI技术核心"段落
- **+提案格式**: 新增第3项"AI技术核心"

### 审判官
- **+AI技术深度维度**: 审查维度从5个扩展到6个
- **+AI创业常见坑**: 6个反模式检查清单
- **+中国市场适用性**: 英文源特别审查

### 裁判
- **+评分强制校验**: 4条自检规则防止分数虚高
- **+SCORE_DELTA**: 每轮输出分数变动日志，追踪评分趋势

### 终极仲裁
- **+AI技术实施评估**: 新增"🤖 AI技术实施评估"章节
  - 核心AI能力、技术选型、Build vs Buy、AI成本预估表

### 话题战略官 (新增)
- 替代原内联 `aiSelectTopics` prompt
- 三步处理: 内容翻译 → AI赋能评估(高/中/低) → 精选决策
- 严格语义去重 + 输出含 `ai_enablement` 和 `ai_angle` 字段

## 四、模型选型理由

| 模型 | 用于 | 选型理由 |
|------|------|---------|
| `claude-sonnet-4-6` | 创想者/仲裁 | 创意+结构化输出最强，中英文双语优秀 |
| `deepseek-v3.2` | 审判官/战略官 | 分析深度强、中文犀利、性价比高 |
| `gpt-4o` | 裁判 | JSON输出最可靠、评分一致性好 |
| `gpt-4o-mini` | 元数据提取 | 纯JSON提取，极快、超低成本 |
| `deepseek-v3.2-fast` | 辩论摘要 | 快速摘要、成本最低 |

## 五、代码位置

- Agent 人设: `backend/internal/pkg/llm/agents.go`
- 话题分析师(手动流程): `backend/internal/pkg/llm/agent_analyst.go`
- 辩论引擎: `backend/internal/biz/debate.go`
- 话题发现: `backend/internal/biz/discovery_usecase.go`
- 模型路由: `backend/internal/pkg/llm/provider_manager.go`
- DB配置: `agent_configs` 表 (provider_id=2 → yunwu.ai)
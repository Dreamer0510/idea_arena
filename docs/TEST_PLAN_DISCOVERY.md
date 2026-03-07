# 话题发现模块测试文档（执行版）

## 1. 测试目的

本轮测试目标有且仅有三件事：

1. 后端新增反馈加权逻辑可回归（单元测试）。
2. 管理后台发现页关键功能可用（看板、筛选、插件页）。
3. **所有插件都能真实抓到对应领域话题**（来源与链接域名可验证）。

---

## 2. 测试范围

### 2.1 本次纳入

- 后端单测：
  - `backend/internal/biz/discovery_feedback_test.go`
  - `backend/internal/data/discovery_feedback_test.go`
- 接口/联调：
  - `GET /api/v1/admin/topics/source-stats`
  - `POST /api/v1/admin/discovery/run`
  - `GET /api/v1/admin/topics`
  - `GET /api/v1/admin/plugins`
  - `PUT /api/v1/admin/plugins/{name}/toggle`
- 插件覆盖（全量 9 个）：
  - `52pojie`
  - `keyword_search`
  - `trend_search`
  - `social_pain`
  - `academic_frontier`
  - `funding_signal`
  - `policy_signal`
  - `demand_signal`
  - `llm_creative`

### 2.2 本次不纳入

- 其他业务模块（辩论、发布、用户管理等）。
- 性能压测与安全测试。

---

## 3. 执行约束（必须遵守）

1. 每次执行测试前，先完整阅读本文件。
2. 一次只执行一个测试项（一个 Test Case）。
3. 一个测试项结束后先汇报结果，不自动跳下一个。
4. 不在测试过程中引入无关改造任务。

---

## 4. 测试环境与前置条件

- 项目路径：`/Users/pannimao/workbase/ai/idea_arena`
- 后端端口：`8000`
- 前端端口：`3000`
- 管理员账号：`admin / admin123`
- 本地数据库：SQLite（`backend/data/idea_arena.db`）
- 工具依赖：`curl`、`jq`、`sqlite3`

### 4.1 一次性准备命令

```bash
cd /Users/pannimao/workbase/ai/idea_arena

# 清理残留进程
pkill -f 'go run ./cmd/server -conf configs/config.yaml' || true
pkill -f 'next dev --turbopack --hostname 0.0.0.0 --port 3000' || true

# 启动后端（单独终端）
cd /Users/pannimao/workbase/ai/idea_arena/backend
go run ./cmd/server -conf configs/config.yaml
```

### 4.2 鉴权与通用变量

```bash
BASE='http://127.0.0.1:8000'
TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

AUTH=(-H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json')
```

---

## 5. 插件测试总策略（关键）

为避免“多插件混跑导致无法判断来源”，每个插件都按单插件隔离流程执行：

1. 先禁用所有插件。
2. 只启用当前测试插件。
3. 清空 `discovered_topics`（避免去重导致 `new_topics=0` 假阴性）。
4. 触发一次 `discovery/run`。
5. 查询 `topics`，按 source 与 source_url 验证。

### 5.1 单插件隔离命令模板

```bash
# 1) 先禁用全部插件
for p in 52pojie keyword_search trend_search social_pain academic_frontier funding_signal policy_signal demand_signal llm_creative; do
  curl -s -X PUT "$BASE/api/v1/admin/plugins/$p/toggle" "${AUTH[@]}" -d '{"enabled":false}' >/dev/null
done

# 2) 只启用目标插件（示例：policy_signal）
PLUGIN='policy_signal'
curl -s -X PUT "$BASE/api/v1/admin/plugins/$PLUGIN/toggle" "${AUTH[@]}" -d '{"enabled":true}'

# 3) 清空发现表，避免历史去重干扰
sqlite3 /Users/pannimao/workbase/ai/idea_arena/backend/data/idea_arena.db "DELETE FROM discovered_topics;"

# 4) 跑一轮发现
curl -s -X POST "$BASE/api/v1/admin/discovery/run" "${AUTH[@]}"

# 5) 拉取结果
curl -s "$BASE/api/v1/admin/topics?page=1&page_size=200" -H "Authorization: Bearer $TOKEN" > /tmp/topics.json
```

---

## 6. 测试用例清单

### 6.1 单元测试 Case

#### TC-UT-BIZ-001：反馈因子构建

- 命令：
  ```bash
  cd /Users/pannimao/workbase/ai/idea_arena/backend
  go test ./internal/biz -run TestBuildSourceFeedbackFactors -count=1
  ```
- 预期：`ok`.

#### TC-UT-BIZ-002：反馈因子应用

- 命令：
  ```bash
  cd /Users/pannimao/workbase/ai/idea_arena/backend
  go test ./internal/biz -run TestApplySourceFeedbackFactor -count=1
  ```
- 预期：`ok`.

#### TC-UT-BIZ-003：抓取去重流程应用反馈

- 命令：
  ```bash
  cd /Users/pannimao/workbase/ai/idea_arena/backend
  go test ./internal/biz -run TestFetchAndDedupApplySourceFeedback -count=1
  ```
- 预期：`ok`.

#### TC-UT-DATA-001：来源反馈统计查询

- 命令：
  ```bash
  cd /Users/pannimao/workbase/ai/idea_arena/backend
  go test ./internal/data -run TestListSourceFeedbackStats -count=1
  ```
- 预期：`ok`.

---

### 6.2 插件能力验证 Case（每个插件 1 个）

> 以下每个 Case 都先执行“5.1 单插件隔离命令模板”，只替换 `PLUGIN`，再执行对应校验命令。

#### TC-PLG-001：`52pojie` 能抓到技术社区话题

- `PLUGIN='52pojie'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="52pojie")] | length' /tmp/topics.json
  jq '[.items[] | select(.source=="52pojie" and (.source_url|test("52pojie\\.cn")))] | length' /tmp/topics.json
  ```
- 通过标准：
  - 第一条结果 `>=1`
  - 第二条结果 `>=1`

#### TC-PLG-002：`keyword_search` 能抓到关键词搜索话题

- `PLUGIN='keyword_search'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="bing_keyword" or .source=="baidu_keyword")] | length' /tmp/topics.json
  jq '[.items[] | select((.source=="bing_keyword" or .source=="baidu_keyword") and (.snippet|test("\\[关键词:")))] | length' /tmp/topics.json
  ```
- 通过标准：两条结果都 `>=1`

#### TC-PLG-003：`trend_search` 能抓到趋势搜索话题

- `PLUGIN='trend_search'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="baidu_trend" or .source=="bing_trend_en" or .source=="bing_keyword")] | length' /tmp/topics.json
  jq '[.items[] | select((.source=="baidu_trend" or .source=="bing_trend_en" or .source=="bing_keyword") and (.title|length>0))] | length' /tmp/topics.json
  ```
- 通过标准：两条结果都 `>=1`

#### TC-PLG-004：`social_pain` 能抓到社交痛点话题

- `PLUGIN='social_pain'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="xiaohongshu_pain" or .source=="zhihu_pain" or .source=="reddit_pain")] | length' /tmp/topics.json
  jq '[.items[] | select((.source=="xiaohongshu_pain" or .source=="zhihu_pain" or .source=="reddit_pain") and (.source_url|test("xiaohongshu|zhihu|reddit"; "i")))] | length' /tmp/topics.json
  ```
- 通过标准：两条结果都 `>=1`

#### TC-PLG-005：`academic_frontier` 能抓到学术前沿话题

- `PLUGIN='academic_frontier'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="arxiv_frontier" or .source=="hf_papers_frontier" or .source=="paperswithcode_frontier")] | length' /tmp/topics.json
  jq '[.items[] | select((.source=="arxiv_frontier" or .source=="hf_papers_frontier" or .source=="paperswithcode_frontier") and (.source_url|test("arxiv|huggingface|paperswithcode"; "i")))] | length' /tmp/topics.json
  ```
- 通过标准：两条结果都 `>=1`

#### TC-PLG-006：`funding_signal` 能抓到融资信号话题

- `PLUGIN='funding_signal'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="36kr_funding" or .source=="techcrunch_funding" or .source=="crunchbase_funding")] | length' /tmp/topics.json
  jq '[.items[] | select((.source=="36kr_funding" or .source=="techcrunch_funding" or .source=="crunchbase_funding") and (.source_url|test("36kr|techcrunch|crunchbase"; "i")))] | length' /tmp/topics.json
  ```
- 通过标准：两条结果都 `>=1`

#### TC-PLG-007：`policy_signal` 能抓到政策信号话题

- `PLUGIN='policy_signal'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="gov_policy" or .source=="ndrc_policy" or .source=="miit_policy")] | length' /tmp/topics.json
  jq '[.items[] | select((.source=="gov_policy" or .source=="ndrc_policy" or .source=="miit_policy") and (.source_url|test("gov\\.cn|ndrc\\.gov\\.cn|miit\\.gov\\.cn"; "i")))] | length' /tmp/topics.json
  ```
- 通过标准：两条结果都 `>=1`

#### TC-PLG-008：`demand_signal` 能抓到付费需求话题

- `PLUGIN='demand_signal'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="linkedin_demand" or .source=="zhaopin_demand" or .source=="g2_review_demand")] | length' /tmp/topics.json
  jq '[.items[] | select((.source=="linkedin_demand" or .source=="zhaopin_demand" or .source=="g2_review_demand") and (.source_url|test("linkedin|zhaopin|g2"; "i")))] | length' /tmp/topics.json
  ```
- 通过标准：两条结果都 `>=1`

#### TC-PLG-009：`llm_creative` 能生成创意话题

- `PLUGIN='llm_creative'`
- 校验命令：
  ```bash
  jq '[.items[] | select(.source=="llm_creative" or .source=="collision")] | length' /tmp/topics.json
  jq '[.items[] | select((.source=="llm_creative" or .source=="collision") and (.title|length>0))] | length' /tmp/topics.json
  ```
- 通过标准：两条结果都 `>=1`

---

### 6.3 UI 基础联调 Case

#### TC-UI-001：来源命中率看板展示

- 目的：`/admin/discovery` 看板可见。
- 预期：有看板模块与数据表格。

#### TC-UI-002：近 7/30 天切换

- 目的：筛选按钮有效。
- 预期：按钮状态切换 + 表格值变化。

#### TC-UI-003：插件页显示新增插件

- 目的：插件页可见新增插件。
- 预期：`policy_signal`、`demand_signal` 可见。

---

## 7. 单项执行流程（后续每次严格按此流程）

1. 先完整阅读本文件。
2. 明确本次只执行一个 Case（例如 `TC-PLG-004`）。
3. 严格按该 Case 步骤执行。
4. 记录结果（通过/失败 + 输出 + 截图）。
5. 先汇报，再等“继续”指令执行下一个。

---

## 8. 结果记录模板

```text
Case ID:
执行时间:
执行人:
结果: 通过 / 失败
关键输出:
截图路径(如有):
问题与备注:
```

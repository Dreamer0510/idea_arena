"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  Search, Zap, Plus, Trash2, Power, ExternalLink,
  ThumbsUp, ThumbsDown, Compass, Tag, Plug, FlaskConical, ChevronDown, ChevronUp,
} from "lucide-react";
import apiClient from "@/lib/api-client";
import {
  PageHeader, Card, StatusBadge, Loading, EmptyState,
  PrimaryButton, SecondaryButton,
} from "@/components/admin/shared";
import type {
  SearchKeyword, CrawlerPlugin, DiscoveredTopic, DiscoveryTag, TopicSourceStat,
} from "@/types/api";

const SUB_TABS = [
  { id: "topics", label: "话题列表", icon: Compass },
  { id: "tags", label: "约束标签", icon: Tag },
  { id: "plugins", label: "渠道插件", icon: Plug },
] as const;

type SubTabID = (typeof SUB_TABS)[number]["id"];

export default function DiscoveryPage() {
  const [tab, setTab] = useState<SubTabID>("topics");

  return (
    <div className="space-y-6 max-w-7xl">
      <PageHeader
        title="话题发现"
        description="从多渠道自动发现热门创业话题，AI 分析推荐后提交辩论"
      />

      {/* Sub-tabs */}
      <div className="flex gap-1 border-b">
        {SUB_TABS.map((t) => {
          const Icon = t.icon;
          return (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`inline-flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                tab === t.id
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              }`}
            >
              <Icon className="h-4 w-4" />
              {t.label}
            </button>
          );
        })}
      </div>

      {tab === "topics" && <TopicsPanel />}
      {tab === "tags" && <TagsPanel />}
      {tab === "plugins" && <PluginsPanel />}
    </div>
  );
}

// ============================================================
// Topics Panel
// ============================================================
function TopicsPanel() {
  const [topics, setTopics] = useState<DiscoveredTopic[]>([]);
  const [sourceStats, setSourceStats] = useState<TopicSourceStat[]>([]);
  const [total, setTotal] = useState(0);
  const [filter, setFilter] = useState("");
  const [days, setDays] = useState(30);
  const [statsLoading, setStatsLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [autoRunning, setAutoRunning] = useState(false);
  const router = useRouter();

  const load = useCallback(() => {
    const params = new URLSearchParams({ page: "1", page_size: "50" });
    if (filter) params.set("status", filter);
    apiClient.get(`/api/v1/admin/topics?${params}`).then((r) => {
      setTopics(r.data.items || []);
      setTotal(r.data.total || 0);
    });
  }, [filter]);

  const loadSourceStats = useCallback(() => {
    setStatsLoading(true);
    apiClient.get(`/api/v1/admin/topics/source-stats?days=${days}`)
      .then((r) => setSourceStats(r.data.items || []))
      .finally(() => setStatsLoading(false));
  }, [days]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { loadSourceStats(); }, [loadSourceStats]);

  const runNow = async () => {
    setRunning(true);
    try {
      const r = await apiClient.post("/api/v1/admin/discovery/run");
      alert(`发现完成！新增 ${r.data.new_topics} 个话题`);
      load();
    } catch (e: unknown) {
      alert("发现失败: " + (e instanceof Error ? e.message : "未知错误"));
    } finally {
      setRunning(false);
    }
  };

  const runAuto = async () => {
    setAutoRunning(true);
    try {
      await apiClient.post("/api/v1/admin/discovery/auto");
      alert("全自动发现+辩论已在后台启动");
      load();
    } catch (e: unknown) {
      alert("全自动发现失败: " + (e instanceof Error ? e.message : "未知错误"));
    } finally {
      setAutoRunning(false);
    }
  };

  const submit = async (id: number) => {
    try {
      const r = await apiClient.post(`/api/v1/admin/topics/${id}/submit`);
      load();
      router.push(`/ideas/${r.data.idea_id}`);
    } catch (e: unknown) {
      alert("提交失败: " + (e instanceof Error ? e.message : "未知错误"));
    }
  };

  const dismiss = async (id: number) => {
    await apiClient.post(`/api/v1/admin/topics/${id}/dismiss`);
    load();
  };

  const sourceLabels: Record<string, string> = {
    "52pojie": "吾爱破解", bing_keyword: "Bing", baidu_keyword: "百度",
    baidu_hot: "百度热搜", github_trending: "GitHub Trending", hackernews_top: "Hacker News",
    hn_pain: "HN痛点", baidu_social_pain: "百度热搜痛点", xhs_explore: "小红书", xhs_baidu: "小红书via百度",
    bing_trend: "Bing热点", bing_trend_en: "Bing(EN)", baidu_trend: "百度热点",
    xiaohongshu_pain: "小红书痛点", zhihu_pain: "知乎痛点", reddit_pain: "Reddit痛点",
    arxiv_api: "arXiv API", hf_daily_papers: "HF每日论文", paperswithcode: "PapersWithCode",
    "36kr_funding": "36氪融资", techcrunch_funding: "TechCrunch融资", crunchbase_funding: "Crunchbase融资",
    gov_policy: "Gov政策", ndrc_policy: "发改委政策", miit_policy: "工信部政策",
    linkedin_demand: "LinkedIn需求", zhaopin_demand: "招聘需求", g2_review_demand: "G2评论需求",
    llm_creative: "AI创意", collision: "跨域碰撞",
  };

  return (
    <div className="space-y-4">
      {/* Source hit-rate board */}
      <Card
        title="来源命中率看板"
        desc="按来源统计话题命中质量（recommended + submitted 视为命中）"
        actions={(
          <div className="flex items-center gap-1">
            {[7, 30, 90].map((d) => (
              <button
                key={d}
                onClick={() => setDays(d)}
                className={`rounded-md px-2.5 py-1 text-xs font-medium ${
                  days === d ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                }`}
              >
                近{d}天
              </button>
            ))}
          </div>
        )}
      >
        {statsLoading ? (
          <Loading text="统计加载中..." />
        ) : sourceStats.length === 0 ? (
          <p className="text-sm text-muted-foreground py-3 text-center">暂无统计数据</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs text-muted-foreground border-b">
                  <th className="py-2 pr-3">来源</th>
                  <th className="py-2 px-3">总量</th>
                  <th className="py-2 px-3">命中</th>
                  <th className="py-2 px-3">已提交</th>
                  <th className="py-2 px-3">命中率</th>
                  <th className="py-2 px-3">提交率</th>
                </tr>
              </thead>
              <tbody>
                {sourceStats.map((row) => {
                  const hitCount = row.recommended_count + row.submitted_count;
                  const hitRate = `${(row.hit_rate * 100).toFixed(1)}%`;
                  const submitRate = `${(row.submit_rate * 100).toFixed(1)}%`;
                  return (
                    <tr key={row.source} className="border-b last:border-0">
                      <td className="py-2 pr-3 text-xs font-medium">{sourceLabels[row.source] || row.source}</td>
                      <td className="py-2 px-3 text-xs">{row.total_count}</td>
                      <td className="py-2 px-3 text-xs">{hitCount}</td>
                      <td className="py-2 px-3 text-xs">{row.submitted_count}</td>
                      <td className="py-2 px-3 text-xs font-semibold">{hitRate}</td>
                      <td className="py-2 px-3 text-xs">{submitRate}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {/* Controls */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          {["", "pending", "recommended", "submitted", "dismissed"].map((s) => (
            <button
              key={s}
              onClick={() => setFilter(s)}
              className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                filter === s
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:bg-accent"
              }`}
            >
              {s === "" ? "全部" : s === "pending" ? "待分析" : s === "recommended" ? "已推荐" : s === "submitted" ? "已提交" : "已忽略"}
            </button>
          ))}
          <span className="text-xs text-muted-foreground ml-2">共 {total} 条</span>
        </div>
        <div className="flex items-center gap-2">
          <SecondaryButton onClick={runNow} disabled={running || autoRunning} loading={running}>
            <Search className="h-4 w-4" /> 仅发现
          </SecondaryButton>
          <PrimaryButton onClick={runAuto} disabled={running || autoRunning} loading={autoRunning}>
            <Zap className="h-4 w-4" /> 全自动发现+辩论
          </PrimaryButton>
        </div>
      </div>

      {/* Topics List */}
      {topics.length === 0 ? (
        <EmptyState
          icon={<Compass className="h-10 w-10" />}
          title="暂无话题"
          description="点击「仅发现」开始探索热门话题"
        />
      ) : (
        <div className="space-y-3">
          {topics.map((t) => (
            <div key={t.id} className="rounded-xl border bg-card p-4 space-y-2">
              <div className="flex items-start justify-between gap-2">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <StatusBadge status={t.status} />
                    <span className="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                      {sourceLabels[t.source] || t.source}
                    </span>
                    {t.recommend_score > 0 && (
                      <span className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-bold ${
                        t.recommend_score >= 8 ? "bg-emerald-500/10 text-emerald-600" :
                        t.recommend_score >= 6 ? "bg-blue-500/10 text-blue-600" :
                        "bg-slate-500/10 text-slate-500"
                      }`}>
                        ⭐ {t.recommend_score.toFixed(1)}
                      </span>
                    )}
                    {t.popularity > 0 && <span className="text-xs text-muted-foreground">👁 {t.popularity}</span>}
                    {t.replies > 0 && <span className="text-xs text-muted-foreground">💬 {t.replies}</span>}
                  </div>
                  <h3 className="font-medium text-sm mt-1 leading-snug">{t.title}</h3>
                </div>
                {t.source_url && (
                  <a href={t.source_url} target="_blank" rel="noopener noreferrer" className="text-muted-foreground hover:text-foreground shrink-0">
                    <ExternalLink className="h-4 w-4" />
                  </a>
                )}
              </div>
              {t.recommendation && (
                <p className="text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2 leading-relaxed whitespace-pre-line">
                  {t.recommendation}
                </p>
              )}
              {(t.status === "pending" || t.status === "recommended") && (
                <div className="flex gap-2 pt-1">
                  <button onClick={() => submit(t.id)} className="inline-flex items-center gap-1 rounded-lg bg-primary/10 text-primary px-3 py-1.5 text-xs font-medium hover:bg-primary/20 transition-colors">
                    <ThumbsUp className="h-3.5 w-3.5" /> 提交辩论
                  </button>
                  <button onClick={() => dismiss(t.id)} className="inline-flex items-center gap-1 rounded-lg bg-muted text-muted-foreground px-3 py-1.5 text-xs font-medium hover:bg-accent transition-colors">
                    <ThumbsDown className="h-3.5 w-3.5" /> 忽略
                  </button>
                </div>
              )}
              {t.status === "submitted" && t.idea_id > 0 && (
                <a href={`/ideas/${t.idea_id}`} className="inline-flex items-center gap-1 text-xs text-primary hover:underline">查看辩论 →</a>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================
// Tags Panel
// ============================================================
const CATEGORY_OPTIONS = [
  { value: "constraint", label: "约束条件", desc: "影响所有渠道的话题筛选", emoji: "🎯" },
];

function TagsPanel() {
  const [tags, setTags] = useState<DiscoveryTag[]>([]);
  const [newTag, setNewTag] = useState("");
  const [newCategory, setNewCategory] = useState("constraint");
  const [adding, setAdding] = useState(false);

  const load = useCallback(() => {
    apiClient.get("/api/v1/admin/tags").then((r) => setTags(r.data.items || []));
  }, []);

  useEffect(() => { load(); }, [load]);

  const add = async () => {
    if (!newTag.trim()) return;
    setAdding(true);
    try {
      await apiClient.post("/api/v1/admin/tags", { tag: newTag.trim(), category: newCategory });
      setNewTag("");
      load();
    } finally { setAdding(false); }
  };

  const toggle = async (t: DiscoveryTag) => {
    await apiClient.put(`/api/v1/admin/tags/${t.id}`, { tag: t.tag, category: t.category, enabled: !t.enabled });
    load();
  };

  const remove = async (id: number) => {
    await apiClient.delete(`/api/v1/admin/tags/${id}`);
    load();
  };

  const grouped = CATEGORY_OPTIONS.map((cat) => ({
    ...cat, items: tags.filter((t) => t.category === cat.value),
  }));

  return (
    <Card title="约束标签与搜索词配置" desc="配置话题发现的约束条件和自定义搜索词">
      <div className="flex gap-2">
        <select value={newCategory} onChange={(e) => setNewCategory(e.target.value)}
          className="rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50">
          {CATEGORY_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.emoji} {c.label}</option>)}
        </select>
        <input value={newTag} onChange={(e) => setNewTag(e.target.value)} onKeyDown={(e) => e.key === "Enter" && add()}
          placeholder={newCategory === "constraint" ? "如：个人开发者、低成本启动" : "如：AI Agent 实际应用"}
          className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50" />
        <PrimaryButton onClick={add} disabled={adding || !newTag.trim()} loading={adding}>
          <Plus className="h-4 w-4" /> 添加
        </PrimaryButton>
      </div>
      {grouped.map((group) => (
        <div key={group.value} className="space-y-2">
          <div className="flex items-center gap-2 pt-2">
            <span className="text-lg">{group.emoji}</span>
            <h3 className="text-sm font-semibold">{group.label}</h3>
            <span className="text-xs text-muted-foreground">— {group.desc}</span>
          </div>
          {group.items.length === 0 ? (
            <p className="text-xs text-muted-foreground py-2 pl-7">暂无，请添加</p>
          ) : (
            <div className="divide-y rounded-lg border">
              {group.items.map((t) => (
                <div key={t.id} className="flex items-center justify-between px-4 py-2.5">
                  <div className="flex items-center gap-3">
                    <button onClick={() => toggle(t)}
                      className={`w-9 h-5 rounded-full transition-colors relative ${t.enabled ? "bg-primary" : "bg-muted"}`}>
                      <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform shadow ${t.enabled ? "left-[18px]" : "left-0.5"}`} />
                    </button>
                    <span className={`text-sm ${t.enabled ? "text-foreground" : "text-muted-foreground line-through"}`}>{t.tag}</span>
                  </div>
                  <button onClick={() => remove(t.id)} className="text-muted-foreground hover:text-destructive transition-colors">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      ))}
    </Card>
  );
}

// ============================================================
// Plugins Panel
// ============================================================
type PluginTestResult = {
  count: number;
  samples: Array<{
    title: string;
    url: string;
    source: string;
    snippet?: string;
    popularity?: number;
    replies?: number;
  }>;
};

type ProviderOption = {
  id: number;
  name: string;
  enabled: boolean;
};

type ProviderModel = {
  model_id: string;
  owned_by: string;
};

type KeywordPluginConfig = {
  ai_expansion: {
    enabled: boolean;
    provider_id: number;
    model: string;
    extra_per_keyword: number;
  };
};

const DEFAULT_KEYWORD_PLUGIN_CONFIG: KeywordPluginConfig = {
  ai_expansion: {
    enabled: false,
    provider_id: 0,
    model: "",
    extra_per_keyword: 2,
  },
};

function parseKeywordPluginConfig(raw?: string): KeywordPluginConfig {
  if (!raw) return DEFAULT_KEYWORD_PLUGIN_CONFIG;
  try {
    const parsed = JSON.parse(raw);
    const ai = parsed?.ai_expansion ?? {};
    return {
      ai_expansion: {
        enabled: Boolean(ai.enabled),
        provider_id: Number(ai.provider_id || 0),
        model: typeof ai.model === "string" ? ai.model : "",
        extra_per_keyword: Math.min(5, Math.max(1, Number(ai.extra_per_keyword || 2))),
      },
    };
  } catch {
    return DEFAULT_KEYWORD_PLUGIN_CONFIG;
  }
}

function PluginsPanel() {
  const [plugins, setPlugins] = useState<CrawlerPlugin[]>([]);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [testingPlugin, setTestingPlugin] = useState<string>("");
  const [testResults, setTestResults] = useState<Record<string, PluginTestResult>>({});
  const [providers, setProviders] = useState<ProviderOption[]>([]);
  const [providerModels, setProviderModels] = useState<Record<number, ProviderModel[]>>({});
  const [keywordPluginConfig, setKeywordPluginConfig] = useState<KeywordPluginConfig>(DEFAULT_KEYWORD_PLUGIN_CONFIG);
  const [savingKeywordConfig, setSavingKeywordConfig] = useState(false);
  const [modelQuery, setModelQuery] = useState("");
  const [modelDropdownOpen, setModelDropdownOpen] = useState(false);

  const [keywords, setKeywords] = useState<SearchKeyword[]>([]);
  const [newKw, setNewKw] = useState("");
  const [addingKw, setAddingKw] = useState(false);
  const [keywordsLoading, setKeywordsLoading] = useState(false);

  const loadPlugins = useCallback(() => {
    apiClient.get("/api/v1/admin/plugins").then((r) => setPlugins(r.data.items || []));
  }, []);

  const loadKeywords = useCallback(() => {
    setKeywordsLoading(true);
    apiClient.get("/api/v1/admin/keywords")
      .then((r) => setKeywords(r.data.items || []))
      .finally(() => setKeywordsLoading(false));
  }, []);

  useEffect(() => { loadPlugins(); }, [loadPlugins]);
  useEffect(() => { loadKeywords(); }, [loadKeywords]);

  useEffect(() => {
    apiClient.get("/api/v1/admin/providers")
      .then((r) => setProviders((r.data.items || []).filter((p: ProviderOption) => p.enabled)))
      .catch(() => setProviders([]));
  }, []);

  const loadModelsForProvider = useCallback(async (providerID: number) => {
    if (!providerID || providerModels[providerID]) return;
    try {
      const res = await apiClient.get(`/api/v1/admin/providers/${providerID}/models`);
      setProviderModels((prev) => ({ ...prev, [providerID]: res.data.items || [] }));
    } catch {
      setProviderModels((prev) => ({ ...prev, [providerID]: [] }));
    }
  }, [providerModels]);

  useEffect(() => {
    const keywordPlugin = plugins.find((p) => p.name === "keyword_search");
    if (!keywordPlugin) return;
    const parsed = parseKeywordPluginConfig(keywordPlugin.config);
    setKeywordPluginConfig(parsed);
    setModelQuery(parsed.ai_expansion.model || "");
    if (parsed.ai_expansion.provider_id > 0) {
      loadModelsForProvider(parsed.ai_expansion.provider_id);
    }
  }, [plugins, loadModelsForProvider]);

  const toggle = async (p: CrawlerPlugin) => {
    await apiClient.put(`/api/v1/admin/plugins/${p.name}/toggle`, { enabled: !p.enabled });
    loadPlugins();
  };

  const runTest = async (p: CrawlerPlugin) => {
    setTestingPlugin(p.name);
    setTestResults((prev) => {
      const next = { ...prev };
      delete next[p.name];
      return next;
    });
    try {
      const r = await apiClient.post(`/api/v1/admin/plugins/${p.name}/test`, { limit: 5 });
      setTestResults((prev) => ({
        ...prev,
        [p.name]: {
          count: r.data.count || 0,
          samples: r.data.samples || [],
        },
      }));
    } catch (e: unknown) {
      const backendError = (e as { response?: { data?: { error?: string } } })?.response?.data?.error;
      if (backendError?.includes("AI模型扩展关键词失败")) {
        alert("AI模型扩展关键词失败，请检查API稳定性");
      } else {
        alert(`插件测试失败：${backendError || (e instanceof Error ? e.message : "未知错误")}`);
      }
    } finally {
      setTestingPlugin("");
    }
  };

  const saveKeywordConfig = async (next: KeywordPluginConfig) => {
    setKeywordPluginConfig(next);
    setSavingKeywordConfig(true);
    try {
      await apiClient.put("/api/v1/admin/plugins/keyword_search/config", { config: next });
      loadPlugins();
    } catch (e: unknown) {
      alert(`保存插件配置失败：${e instanceof Error ? e.message : "未知错误"}`);
    } finally {
      setSavingKeywordConfig(false);
    }
  };

  const commitModelQuery = async (value?: string) => {
    if (!keywordPluginConfig.ai_expansion.enabled) return;
    if (keywordPluginConfig.ai_expansion.provider_id <= 0) return;
    const nextModel = (value ?? modelQuery).trim();
    if (nextModel === keywordPluginConfig.ai_expansion.model) return;
    await saveKeywordConfig({
      ...keywordPluginConfig,
      ai_expansion: {
        ...keywordPluginConfig.ai_expansion,
        model: nextModel,
      },
    });
  };

  const addKeyword = async () => {
    if (!newKw.trim()) return;
    setAddingKw(true);
    try {
      await apiClient.post("/api/v1/admin/keywords", { keyword: newKw.trim() });
      setNewKw("");
      loadKeywords();
    } finally {
      setAddingKw(false);
    }
  };

  const toggleKeyword = async (kw: SearchKeyword) => {
    await apiClient.put(`/api/v1/admin/keywords/${kw.id}`, { keyword: kw.keyword, enabled: !kw.enabled });
    loadKeywords();
  };

  const removeKeyword = async (id: number) => {
    await apiClient.delete(`/api/v1/admin/keywords/${id}`);
    loadKeywords();
  };

  const sourceEmoji: Record<string, string> = {
    "52pojie": "🔓", keyword_search: "🔍", trend_search: "🔥", social_pain: "🫂",
    academic_frontier: "🧪", funding_signal: "💰", policy_signal: "🏛️",
    demand_signal: "📈", llm_creative: "🧠",
  };

  const currentProviderModels = providerModels[keywordPluginConfig.ai_expansion.provider_id] || [];
  const normalizedModelQuery = modelQuery.trim().toLowerCase();
  const filteredProviderModels = currentProviderModels.filter((model) =>
    !normalizedModelQuery || model.model_id.toLowerCase().includes(normalizedModelQuery),
  );

  return (
    <Card title="渠道插件管理" desc="支持按插件展开编辑配置，并可直接执行连通性测试">
      {plugins.length === 0 ? (
        <p className="text-sm text-muted-foreground py-4 text-center">没有已注册的插件</p>
      ) : (
        <div className="grid gap-3">
          {plugins.map((p) => (
            <div key={p.name}
              className={`rounded-xl border transition-colors ${p.enabled ? "border-primary/40 bg-primary/5" : "bg-card"}`}>
              <div className="flex items-center justify-between gap-3 p-4">
                <div className="flex items-center gap-3">
                  <span className="text-2xl">{sourceEmoji[p.name] || "📡"}</span>
                  <div>
                    <div className="font-medium text-sm">{p.label || p.name}</div>
                    <div className="text-xs text-muted-foreground">ID: {p.name}</div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => runTest(p)}
                    disabled={testingPlugin === p.name}
                    className="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium bg-muted text-muted-foreground hover:bg-accent disabled:opacity-60"
                  >
                    <FlaskConical className="h-3.5 w-3.5" />
                    {testingPlugin === p.name ? "测试中" : "测试"}
                  </button>
                  <button onClick={() => toggle(p)}
                    className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${
                      p.enabled ? "bg-primary text-primary-foreground hover:bg-primary/90" : "bg-muted text-muted-foreground hover:bg-accent"
                    }`}>
                    <Power className="h-3.5 w-3.5" />
                    {p.enabled ? "运行中" : "已关闭"}
                  </button>
                  <button
                    onClick={() => setExpanded((prev) => ({ ...prev, [p.name]: !prev[p.name] }))}
                    className="inline-flex items-center gap-1 rounded-lg px-2 py-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                    title="展开编辑配置"
                  >
                    {expanded[p.name] ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                  </button>
                </div>
              </div>

              {testingPlugin === p.name && p.name === "keyword_search" && (
                <div className="px-4 pb-3 -mt-1">
                  <div className="rounded-lg border border-amber-400/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-200">
                    正在进行多搜索引擎并发检索与相关性匹配（Bing + 百度，各关键词会抓取更多候选结果后再筛选）。该过程可能耗时较长，请稍候。
                  </div>
                </div>
              )}

              {testingPlugin === p.name && p.name === "trend_search" && (
                <div className="px-4 pb-3 -mt-1">
                  <div className="rounded-lg border border-blue-400/40 bg-blue-500/10 px-3 py-2 text-xs text-blue-800 dark:text-blue-200">
                    正在并发抓取百度热搜、GitHub Trending、Hacker News 三个真实热点来源，可能需要数秒，请稍候。
                  </div>
                </div>
              )}

              {testResults[p.name] && (
                <div className="px-4 pb-3 -mt-1 space-y-2">
                  <div className="rounded-lg bg-muted/60 px-3 py-2 text-xs text-muted-foreground">
                    测试结果：抓取到 <span className="font-semibold text-foreground">{testResults[p.name].count}</span> 条数据。
                    <span className="ml-1">
                      {p.name === "keyword_search"
                        ? "以下为关键词插件并发抓取后，经过相关性匹配筛选得到的 Top 结果。"
                        : "以下为插件原始抓取结果（未经过去重、评分和推荐筛选）。"}
                    </span>
                  </div>

                  {testResults[p.name].samples.length > 0 && (
                    <div className="overflow-x-auto rounded-lg border bg-background">
                      <table className="w-full min-w-[860px] text-xs">
                        <thead>
                          <tr className="border-b bg-muted/40 text-left text-muted-foreground">
                            <th className="px-3 py-2 font-medium w-12">#</th>
                            <th className="px-3 py-2 font-medium">标题</th>
                            <th className="px-3 py-2 font-medium w-28">来源</th>
                            <th className="px-3 py-2 font-medium w-20">热度</th>
                            <th className="px-3 py-2 font-medium w-20">讨论</th>
                            <th className="px-3 py-2 font-medium">链接</th>
                            <th className="px-3 py-2 font-medium">摘要</th>
                          </tr>
                        </thead>
                        <tbody>
                          {testResults[p.name].samples.map((item, idx) => (
                            <tr key={`${p.name}-${idx}`} className="border-b last:border-0 align-top">
                              <td className="px-3 py-2 text-muted-foreground">{idx + 1}</td>
                              <td className="px-3 py-2 text-foreground leading-relaxed">{item.title || "-"}</td>
                              <td className="px-3 py-2 text-muted-foreground">{item.source || "-"}</td>
                              <td className="px-3 py-2 text-muted-foreground">{item.popularity ?? 0}</td>
                              <td className="px-3 py-2 text-muted-foreground">{item.replies ?? 0}</td>
                              <td className="px-3 py-2">
                                {item.url ? (
                                  <a
                                    href={item.url}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="text-primary hover:underline break-all"
                                  >
                                    {item.url}
                                  </a>
                                ) : "-"}
                              </td>
                              <td className="px-3 py-2 text-muted-foreground leading-relaxed">
                                {item.snippet || "-"}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}

              {expanded[p.name] && (
                <div className="border-t px-4 py-4 bg-background/50">
                  {p.name === "trend_search" ? (
                    <div className="space-y-2">
                      <h4 className="text-sm font-semibold">数据来源说明</h4>
                      <p className="text-xs text-muted-foreground leading-relaxed">
                        该插件从三个真实热点来源并发抓取，无需手动配置搜索词：
                      </p>
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
                        <div className="rounded-lg border bg-card p-3">
                          <div className="flex items-center gap-1.5 mb-1">
                            <span className="text-base">🔥</span>
                            <span className="text-xs font-semibold">百度热搜</span>
                          </div>
                          <p className="text-xs text-muted-foreground">实时热搜 API，自动筛选 AI/科技相关话题。降级方案：Bing RSS。</p>
                        </div>
                        <div className="rounded-lg border bg-card p-3">
                          <div className="flex items-center gap-1.5 mb-1">
                            <span className="text-base">🐙</span>
                            <span className="text-xs font-semibold">GitHub Trending</span>
                          </div>
                          <p className="text-xs text-muted-foreground">每日热门仓库，直接解析 GitHub Trending 页面。降级方案：Bing RSS。</p>
                        </div>
                        <div className="rounded-lg border bg-card p-3">
                          <div className="flex items-center gap-1.5 mb-1">
                            <span className="text-base">📰</span>
                            <span className="text-xs font-semibold">Hacker News</span>
                          </div>
                          <p className="text-xs text-muted-foreground">Firebase 公开 API 获取热门科技文章。降级方案：Bing RSS。</p>
                        </div>
                      </div>
                    </div>
                  ) : p.name === "keyword_search" ? (
                    <div className="space-y-3">
                      <div className="rounded-lg border bg-card p-3 space-y-3">
                        <div className="flex items-center justify-between">
                          <div>
                            <h4 className="text-sm font-semibold">AI 思路扩展</h4>
                            <p className="text-xs text-muted-foreground mt-1">
                              开启后会调用你选择的服务商和模型，对关键词做随机行业场景扩展，降低重复结果。
                            </p>
                          </div>
                          <button
                            onClick={() => saveKeywordConfig({
                              ...keywordPluginConfig,
                              ai_expansion: {
                                ...keywordPluginConfig.ai_expansion,
                                enabled: !keywordPluginConfig.ai_expansion.enabled,
                              },
                            })}
                            disabled={savingKeywordConfig}
                            className={`w-10 h-5 rounded-full transition-colors relative ${keywordPluginConfig.ai_expansion.enabled ? "bg-primary" : "bg-muted"}`}
                            title="启用/关闭 AI 思路扩展"
                          >
                            <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform shadow ${keywordPluginConfig.ai_expansion.enabled ? "left-[20px]" : "left-0.5"}`} />
                          </button>
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
                          <select
                            value={keywordPluginConfig.ai_expansion.provider_id}
                            onChange={async (e) => {
                              const providerID = Number(e.target.value);
                              if (providerID > 0) await loadModelsForProvider(providerID);
                              setModelQuery("");
                              setModelDropdownOpen(false);
                              saveKeywordConfig({
                                ...keywordPluginConfig,
                                ai_expansion: {
                                  ...keywordPluginConfig.ai_expansion,
                                  provider_id: providerID,
                                  model: "",
                                },
                              });
                            }}
                            className="rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                            disabled={!keywordPluginConfig.ai_expansion.enabled || savingKeywordConfig}
                          >
                            <option value={0}>选择服务商</option>
                            {providers.map((provider) => (
                              <option key={provider.id} value={provider.id}>{provider.name}</option>
                            ))}
                          </select>

                          <div className="relative">
                            <input
                              value={modelQuery}
                              onChange={(e) => {
                                setModelQuery(e.target.value);
                                setModelDropdownOpen(true);
                              }}
                              onFocus={() => setModelDropdownOpen(true)}
                              onBlur={() => {
                                setTimeout(() => {
                                  setModelDropdownOpen(false);
                                  void commitModelQuery();
                                }, 120);
                              }}
                              onKeyDown={(e) => {
                                if (e.key === "Enter") {
                                  e.preventDefault();
                                  void commitModelQuery();
                                  setModelDropdownOpen(false);
                                }
                              }}
                              placeholder="搜索并选择模型"
                              className="w-full rounded-lg border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                              disabled={!keywordPluginConfig.ai_expansion.enabled || savingKeywordConfig || keywordPluginConfig.ai_expansion.provider_id <= 0}
                            />
                            {modelDropdownOpen && keywordPluginConfig.ai_expansion.enabled && keywordPluginConfig.ai_expansion.provider_id > 0 && (
                              <div className="absolute z-20 mt-1 w-full rounded-lg border bg-background shadow-lg">
                                <div className="px-3 py-2 text-xs text-muted-foreground border-b">
                                  共 {currentProviderModels.length} 个模型，显示 {filteredProviderModels.length} 个匹配项
                                </div>
                                <div className="max-h-52 overflow-y-auto">
                                  {filteredProviderModels.length === 0 ? (
                                    <div className="px-3 py-2 text-xs text-muted-foreground">没有匹配模型，可直接回车使用输入值</div>
                                  ) : (
                                    filteredProviderModels.map((model) => (
                                      <button
                                        key={model.model_id}
                                        type="button"
                                        onMouseDown={(e) => {
                                          e.preventDefault();
                                          setModelQuery(model.model_id);
                                          setModelDropdownOpen(false);
                                          void commitModelQuery(model.model_id);
                                        }}
                                        className={`w-full text-left px-3 py-2 text-xs hover:bg-accent ${
                                          keywordPluginConfig.ai_expansion.model === model.model_id ? "bg-primary/10 text-primary" : ""
                                        }`}
                                        title={model.model_id}
                                      >
                                        {model.model_id}
                                      </button>
                                    ))
                                  )}
                                </div>
                              </div>
                            )}
                          </div>

                          <select
                            value={keywordPluginConfig.ai_expansion.extra_per_keyword}
                            onChange={(e) => saveKeywordConfig({
                              ...keywordPluginConfig,
                              ai_expansion: {
                                ...keywordPluginConfig.ai_expansion,
                                extra_per_keyword: Number(e.target.value),
                              },
                            })}
                            className="rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                            disabled={!keywordPluginConfig.ai_expansion.enabled || savingKeywordConfig}
                          >
                            <option value={1}>每词扩展 1 条</option>
                            <option value={2}>每词扩展 2 条</option>
                            <option value={3}>每词扩展 3 条</option>
                            <option value={4}>每词扩展 4 条</option>
                            <option value={5}>每词扩展 5 条</option>
                          </select>
                        </div>
                      </div>

                      <div>
                        <h4 className="text-sm font-semibold">搜索关键词配置（仅作用于该插件）</h4>
                        <p className="text-xs text-muted-foreground mt-1">
                          配置后将用于 `keyword_search` 的 Bing/百度抓取与测试。
                        </p>
                      </div>
                      <div className="flex gap-2">
                        <input
                          value={newKw}
                          onChange={(e) => setNewKw(e.target.value)}
                          onKeyDown={(e) => e.key === "Enter" && addKeyword()}
                          placeholder="输入关键词，如：AI 自动化办公"
                          className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                        />
                        <PrimaryButton onClick={addKeyword} disabled={addingKw || !newKw.trim()} loading={addingKw}>
                          <Plus className="h-4 w-4" /> 添加
                        </PrimaryButton>
                      </div>
                      {keywordsLoading ? (
                        <Loading text="关键词加载中..." />
                      ) : keywords.length === 0 ? (
                        <p className="text-sm text-muted-foreground py-2 text-center">暂无关键词，请先添加</p>
                      ) : (
                        <div className="divide-y rounded-lg border">
                          {keywords.map((kw) => (
                            <div key={kw.id} className="flex items-center justify-between px-4 py-2.5">
                              <div className="flex items-center gap-3">
                                <button onClick={() => toggleKeyword(kw)}
                                  className={`w-9 h-5 rounded-full transition-colors relative ${kw.enabled ? "bg-primary" : "bg-muted"}`}>
                                  <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform shadow ${kw.enabled ? "left-[18px]" : "left-0.5"}`} />
                                </button>
                                <span className={`text-sm ${kw.enabled ? "text-foreground" : "text-muted-foreground line-through"}`}>{kw.keyword}</span>
                              </div>
                              <button onClick={() => removeKeyword(kw.id)} className="text-muted-foreground hover:text-destructive transition-colors">
                                <Trash2 className="h-4 w-4" />
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      当前插件暂无可视化配置项，后续可在此区域继续扩展。
                    </p>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

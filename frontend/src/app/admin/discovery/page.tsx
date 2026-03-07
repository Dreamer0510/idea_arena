"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  Search, Zap, Plus, Trash2, Power, ExternalLink,
  ThumbsUp, ThumbsDown, Compass, Tag, Key, Plug,
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
  { id: "keywords", label: "搜索关键词", icon: Key },
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
      {tab === "keywords" && <KeywordsPanel />}
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
    bing_trend: "Bing热点", bing_trend_en: "Bing(EN)", baidu_trend: "百度热点",
    xiaohongshu_pain: "小红书痛点", zhihu_pain: "知乎痛点", reddit_pain: "Reddit痛点",
    arxiv_frontier: "arXiv前沿", hf_papers_frontier: "HF Papers", paperswithcode_frontier: "PapersWithCode",
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
  { value: "trend_query_cn", label: "中文搜索词", desc: "热点趋势抓取使用的中文搜索关键词", emoji: "🔍" },
  { value: "trend_query_en", label: "英文搜索词", desc: "热点趋势抓取使用的英文搜索关键词", emoji: "🌐" },
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
// Keywords Panel
// ============================================================
function KeywordsPanel() {
  const [keywords, setKeywords] = useState<SearchKeyword[]>([]);
  const [newKw, setNewKw] = useState("");
  const [adding, setAdding] = useState(false);

  const load = useCallback(() => {
    apiClient.get("/api/v1/admin/keywords").then((r) => setKeywords(r.data.items || []));
  }, []);

  useEffect(() => { load(); }, [load]);

  const add = async () => {
    if (!newKw.trim()) return;
    setAdding(true);
    try {
      await apiClient.post("/api/v1/admin/keywords", { keyword: newKw.trim() });
      setNewKw("");
      load();
    } finally { setAdding(false); }
  };

  const toggle = async (kw: SearchKeyword) => {
    await apiClient.put(`/api/v1/admin/keywords/${kw.id}`, { keyword: kw.keyword, enabled: !kw.enabled });
    load();
  };

  const remove = async (id: number) => {
    await apiClient.delete(`/api/v1/admin/keywords/${id}`);
    load();
  };

  return (
    <Card title="搜索关键词管理" desc="预设关键词将被定时用于 Bing/百度搜索，发现新的创业话题灵感">
      <div className="flex gap-2">
        <input value={newKw} onChange={(e) => setNewKw(e.target.value)} onKeyDown={(e) => e.key === "Enter" && add()}
          placeholder="输入关键词，如：AI 自动化办公"
          className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50" />
        <PrimaryButton onClick={add} disabled={adding || !newKw.trim()} loading={adding}>
          <Plus className="h-4 w-4" /> 添加
        </PrimaryButton>
      </div>
      {keywords.length === 0 ? (
        <p className="text-sm text-muted-foreground py-4 text-center">暂无关键词，添加一些吧 ↑</p>
      ) : (
        <div className="divide-y rounded-lg border mt-3">
          {keywords.map((kw) => (
            <div key={kw.id} className="flex items-center justify-between px-4 py-2.5">
              <div className="flex items-center gap-3">
                <button onClick={() => toggle(kw)}
                  className={`w-9 h-5 rounded-full transition-colors relative ${kw.enabled ? "bg-primary" : "bg-muted"}`}>
                  <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform shadow ${kw.enabled ? "left-[18px]" : "left-0.5"}`} />
                </button>
                <span className={`text-sm ${kw.enabled ? "text-foreground" : "text-muted-foreground line-through"}`}>{kw.keyword}</span>
              </div>
              <button onClick={() => remove(kw.id)} className="text-muted-foreground hover:text-destructive transition-colors">
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

// ============================================================
// Plugins Panel
// ============================================================
function PluginsPanel() {
  const [plugins, setPlugins] = useState<CrawlerPlugin[]>([]);

  const load = useCallback(() => {
    apiClient.get("/api/v1/admin/plugins").then((r) => setPlugins(r.data.items || []));
  }, []);

  useEffect(() => { load(); }, [load]);

  const toggle = async (p: CrawlerPlugin) => {
    await apiClient.put(`/api/v1/admin/plugins/${p.name}/toggle`, { enabled: !p.enabled });
    load();
  };

  const sourceEmoji: Record<string, string> = {
    "52pojie": "🔓", keyword_search: "🔍", trend_search: "🔥", llm_creative: "🧠",
  };

  return (
    <Card title="渠道插件管理" desc="开启或关闭话题发现渠道，每个渠道独立抓取并汇总分析">
      {plugins.length === 0 ? (
        <p className="text-sm text-muted-foreground py-4 text-center">没有已注册的插件</p>
      ) : (
        <div className="grid gap-3">
          {plugins.map((p) => (
            <div key={p.name}
              className={`flex items-center justify-between rounded-xl border p-4 transition-colors ${p.enabled ? "border-primary/30 bg-primary/5" : ""}`}>
              <div className="flex items-center gap-3">
                <span className="text-2xl">{sourceEmoji[p.name] || "📡"}</span>
                <div>
                  <div className="font-medium text-sm">{p.label || p.name}</div>
                  <div className="text-xs text-muted-foreground">ID: {p.name}</div>
                </div>
              </div>
              <button onClick={() => toggle(p)}
                className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${
                  p.enabled ? "bg-primary text-primary-foreground hover:bg-primary/90" : "bg-muted text-muted-foreground hover:bg-accent"
                }`}>
                <Power className="h-3.5 w-3.5" />
                {p.enabled ? "运行中" : "已关闭"}
              </button>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

"use client";

import { useMemo } from "react";
import Link from "next/link";
import {
  Activity, TrendingUp, Trophy, Zap, BarChart3, Clock,
  CheckCircle2, XCircle, Loader2, Swords, Target, ArrowUpRight, Vote
} from "lucide-react";
import { motion } from "motion/react";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  PieChart, Pie, Cell, ResponsiveContainer, LineChart, Line,
  Legend
} from "recharts";
import { useIdeas, useIdeaStats } from "@/hooks/use-ideas";
import type { IdeaInfo } from "@/types/api";
import { VerdictBadge } from "@/components/ideas/badges";

/* ───────── KPI Card ───────── */
function KPICard({ title, value, sub, icon, color, delay = 0 }: {
  title: string; value: string | number; sub?: string;
  icon: React.ReactNode; color: string; delay?: number;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }}
      transition={{ delay, duration: 0.5 }}
      className="rounded-2xl border bg-card p-5 flex items-start gap-4 hover:shadow-lg transition-shadow"
    >
      <div className={`w-11 h-11 rounded-xl flex items-center justify-center shrink-0 ${color}`}>
        {icon}
      </div>
      <div className="min-w-0">
        <div className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-1">{title}</div>
        <div className="text-2xl font-black font-mono tabular-nums tracking-tight">{value}</div>
        {sub && <div className="text-xs text-muted-foreground mt-0.5">{sub}</div>}
      </div>
    </motion.div>
  );
}

/* ───────── Chart Colors ───────── */
const STATUS_COLORS: Record<string, string> = {
  graduated: "#22c55e",
  promising: "#f59e0b",
  failed: "#ef4444",
  debating: "#3b82f6",
  pending: "#eab308",
};
const STATUS_LABELS: Record<string, string> = {
  graduated: "已毕业",
  promising: "有潜力",
  failed: "未通过",
  debating: "辩论中",
  pending: "等待中",
};

const CHART_THEME = {
  bg: "hsl(var(--card))",
  text: "hsl(var(--muted-foreground))",
  grid: "hsl(var(--border))",
  primary: "hsl(var(--primary))",
};

/* ───────── Main Page ───────── */
export default function MonitorPage() {
  const { data: statsData, isLoading: statsLoading } = useIdeaStats();
  const { data: recentData } = useIdeas({ page: 1, page_size: 10, sort_by: "created_at", sort_order: "desc" });

  const stats = useMemo(() => {
    if (!statsData) return null;
    const sc = statsData.status_counts || {};
    const statusCounts: Record<string, number> = {
      graduated: sc.graduated || 0,
      promising: sc.promising || 0,
      failed: sc.failed || 0,
      debating: sc.debating || 0,
      pending: sc.pending || 0,
    };

    const avgScores = statsData.avg_scores || {};
    const completed = statusCounts.graduated + statusCounts.promising + statusCounts.failed;
    const graduationRate = completed > 0 ? ((statusCounts.graduated + statusCounts.promising) / completed * 100) : 0;

    // Pie data
    const pieData = Object.entries(statusCounts)
      .filter(([, v]) => v > 0)
      .map(([k, v]) => ({ name: STATUS_LABELS[k] || k, value: v, fill: STATUS_COLORS[k] || "#666" }));

    // Score distribution from backend
    const bucketMap = statsData.score_buckets || {};
    const scoreBuckets = [
      { range: "0-2", count: bucketMap["0-2"] || 0 },
      { range: "2-4", count: bucketMap["2-4"] || 0 },
      { range: "4-6", count: bucketMap["4-6"] || 0 },
      { range: "6-8", count: bucketMap["6-8"] || 0 },
      { range: "8-10", count: bucketMap["8-10"] || 0 },
    ];

    // Score dimensions bar chart
    const dimensionData = [
      { dim: "可行性", avg: Number((avgScores.feasibility || 0).toFixed(1)) },
      { dim: "经济性", avg: Number((avgScores.economics || 0).toFixed(1)) },
      { dim: "利润潜力", avg: Number((avgScores.profit || 0).toFixed(1)) },
      { dim: "综合", avg: Number((avgScores.overall || 0).toFixed(1)) },
    ];

    const recent = recentData?.items || [];

    return {
      total: statsData.total,
      statusCounts,
      avgScore: avgScores.overall || 0,
      totalRounds: statsData.total_rounds,
      graduationRate,
      pieData,
      scoreBuckets,
      dimensionData,
      recent,
      voteStats: statsData.vote_stats,
    };
  }, [statsData, recentData?.items]);

  if (statsLoading || !stats) {
    return (
      <div className="flex flex-col items-center justify-center py-20 gap-4">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
        <span className="text-sm font-semibold text-muted-foreground tracking-wider uppercase animate-pulse">加载监控数据...</span>
      </div>
    );
  }

  return (
    <div className="space-y-8 pb-16">
      {/* Header */}
      <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} className="space-y-2">
        <h1 className="text-4xl font-extrabold tracking-tight">
          监控<span className="text-primary">大盘</span>
        </h1>
        <p className="text-muted-foreground font-medium">
          实时概览所有点子的评估状态、评分分布和系统运行情况
        </p>
      </motion.div>

      {/* KPI Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
        <KPICard title="总点子数" value={stats.total} icon={<Zap className="w-5 h-5 text-white" />} color="bg-primary" delay={0} />
        <KPICard title="平均得分" value={stats.avgScore.toFixed(1)} icon={<Trophy className="w-5 h-5 text-white" />} color="bg-amber-500" delay={0.05} />
        <KPICard title="总辩论轮次" value={stats.totalRounds} icon={<Swords className="w-5 h-5 text-white" />} color="bg-violet-500" delay={0.1} />
        <KPICard title="通过率" value={`${stats.graduationRate.toFixed(0)}%`} sub={`${stats.statusCounts.graduated + stats.statusCounts.promising} / ${stats.statusCounts.graduated + stats.statusCounts.promising + stats.statusCounts.failed}`} icon={<CheckCircle2 className="w-5 h-5 text-white" />} color="bg-green-500" delay={0.15} />
        <KPICard title="辩论中" value={stats.statusCounts.debating} icon={<Activity className="w-5 h-5 text-white" />} color="bg-blue-500" delay={0.2} />
        <KPICard title="等待中" value={stats.statusCounts.pending} icon={<Clock className="w-5 h-5 text-white" />} color="bg-yellow-500" delay={0.25} />
      </div>

      {/* Charts Row 1 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Status Pie */}
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.15 }}
          className="rounded-2xl border bg-card p-6">
          <h3 className="text-sm font-bold uppercase tracking-wider text-muted-foreground mb-4">状态分布</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={stats.pieData} dataKey="value" nameKey="name" cx="50%" cy="50%"
                  outerRadius={90} innerRadius={50} paddingAngle={3} strokeWidth={0}>
                  {stats.pieData.map((entry, i) => (
                    <Cell key={i} fill={entry.fill} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ backgroundColor: "hsl(var(--card))", border: "1px solid hsl(var(--border))", borderRadius: "12px", fontSize: "13px" }}
                  itemStyle={{ color: "hsl(var(--foreground))" }}
                />
                <Legend wrapperStyle={{ fontSize: "12px" }} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </motion.div>

        {/* Score Distribution */}
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.2 }}
          className="rounded-2xl border bg-card p-6">
          <h3 className="text-sm font-bold uppercase tracking-wider text-muted-foreground mb-4">评分分布</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={stats.scoreBuckets}>
                <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                <XAxis dataKey="range" tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 12 }} />
                <YAxis allowDecimals={false} tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 12 }} />
                <Tooltip
                  contentStyle={{ backgroundColor: "hsl(var(--card))", border: "1px solid hsl(var(--border))", borderRadius: "12px", fontSize: "13px" }}
                  itemStyle={{ color: "hsl(var(--foreground))" }}
                />
                <Bar dataKey="count" name="数量" fill="hsl(var(--primary))" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </motion.div>
      </div>

      {/* Charts Row 2 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Dimension Avg */}
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.25 }}
          className="rounded-2xl border bg-card p-6">
          <h3 className="text-sm font-bold uppercase tracking-wider text-muted-foreground mb-4">各维度平均分</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={stats.dimensionData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                <XAxis type="number" domain={[0, 10]} tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 12 }} />
                <YAxis type="category" dataKey="dim" tick={{ fill: "hsl(var(--muted-foreground))", fontSize: 12 }} width={60} />
                <Tooltip
                  contentStyle={{ backgroundColor: "hsl(var(--card))", border: "1px solid hsl(var(--border))", borderRadius: "12px", fontSize: "13px" }}
                  itemStyle={{ color: "hsl(var(--foreground))" }}
                />
                <Bar dataKey="avg" name="平均分" fill="rgb(6,182,212)" radius={[0, 6, 6, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </motion.div>

        {/* Voting Stats */}
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3 }}
          className="rounded-2xl border bg-card p-6">
          <h3 className="text-sm font-bold uppercase tracking-wider text-muted-foreground mb-4">投票统计</h3>
          {stats.voteStats && stats.voteStats.total_voted > 0 ? (
            <div className="space-y-6">
              <div className="text-center">
                <div className="text-4xl font-black font-mono">{stats.voteStats.total_voted}</div>
                <div className="text-xs text-muted-foreground mt-1">已投票点子</div>
              </div>
              <div className="space-y-3">
                {[
                  { label: "可行 (YES)", rate: stats.voteStats.yes_rate, color: "bg-green-500" },
                  { label: "不可行 (NO)", rate: stats.voteStats.no_rate, color: "bg-red-500" },
                  { label: "有条件 (COND)", rate: stats.voteStats.cond_rate, color: "bg-amber-500" },
                ].map(item => (
                  <div key={item.label} className="space-y-1">
                    <div className="flex justify-between text-sm">
                      <span className="font-medium">{item.label}</span>
                      <span className="text-muted-foreground font-mono text-xs">{item.rate.toFixed(1)}%</span>
                    </div>
                    <div className="h-2.5 rounded-full bg-muted overflow-hidden">
                      <motion.div
                        initial={{ width: 0 }} animate={{ width: `${item.rate}%` }}
                        transition={{ delay: 0.4, duration: 0.8 }}
                        className={`h-full rounded-full ${item.color}`}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-48 text-muted-foreground text-sm gap-2">
              <Vote className="w-8 h-8 opacity-30" />
              暂无投票数据
            </div>
          )}
        </motion.div>
      </div>

      {/* Recent Debates Table */}
      <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.35 }}
        className="rounded-2xl border bg-card overflow-hidden">
        <div className="px-6 py-4 border-b bg-muted/30">
          <h3 className="text-sm font-bold uppercase tracking-wider text-muted-foreground">近期辩论</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left">
                <th className="px-6 py-3 font-semibold text-muted-foreground text-xs uppercase tracking-wider">项目</th>
                <th className="px-6 py-3 font-semibold text-muted-foreground text-xs uppercase tracking-wider">状态</th>
                <th className="px-6 py-3 font-semibold text-muted-foreground text-xs uppercase tracking-wider text-center">轮次</th>
                <th className="px-6 py-3 font-semibold text-muted-foreground text-xs uppercase tracking-wider text-center">可行</th>
                <th className="px-6 py-3 font-semibold text-muted-foreground text-xs uppercase tracking-wider text-center">经济</th>
                <th className="px-6 py-3 font-semibold text-muted-foreground text-xs uppercase tracking-wider text-center">利润</th>
                <th className="px-6 py-3 font-semibold text-muted-foreground text-xs uppercase tracking-wider text-center">综合</th>
                <th className="px-6 py-3 font-semibold text-muted-foreground text-xs uppercase tracking-wider">时间</th>
                <th className="px-6 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {stats.recent.map((idea) => (
                <tr key={idea.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-6 py-3.5">
                    <div className="font-semibold max-w-[200px] truncate">{idea.product_name || idea.topic}</div>
                  </td>
                  <td className="px-6 py-3.5"><VerdictBadge verdict={idea.status} /></td>
                  <td className="px-6 py-3.5 text-center font-mono text-muted-foreground">{idea.round_count || "-"}</td>
                  <td className="px-6 py-3.5 text-center"><ScoreCell value={idea.score_feasibility} /></td>
                  <td className="px-6 py-3.5 text-center"><ScoreCell value={idea.score_economics} /></td>
                  <td className="px-6 py-3.5 text-center"><ScoreCell value={idea.score_profit} /></td>
                  <td className="px-6 py-3.5 text-center"><ScoreCell value={idea.score_overall} bold /></td>
                  <td className="px-6 py-3.5 text-muted-foreground text-xs font-mono">
                    {new Date(idea.created_at).toLocaleDateString("zh-CN", { month: "short", day: "numeric" })}
                  </td>
                  <td className="px-6 py-3.5">
                    <Link href={`/ideas/${idea.id}`} className="text-primary hover:underline inline-flex items-center gap-1 text-xs font-semibold">
                      详情 <ArrowUpRight className="w-3 h-3" />
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </motion.div>
    </div>
  );
}

/* ───────── Score Cell ───────── */
function ScoreCell({ value, bold }: { value: number; bold?: boolean }) {
  if (!value || value === 0) return <span className="text-muted-foreground">-</span>;
  const color = value >= 8 ? "text-green-500" : value >= 6 ? "text-amber-500" : value >= 4 ? "text-orange-500" : "text-red-500";
  return <span className={`font-mono tabular-nums ${color} ${bold ? "font-black text-base" : "font-semibold"}`}>{value.toFixed(1)}</span>;
}

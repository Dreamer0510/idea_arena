"use client";

import { useState, useEffect } from "react";
import { Activity, Zap, Trophy, TrendingUp, BarChart3, Clock } from "lucide-react";
import apiClient from "@/lib/api-client";
import { StatCard, Card, Loading, PageHeader } from "@/components/admin/shared";

interface DashboardStats {
  total_ideas: number;
  debating_count: number;
  graduated_count: number;
  failed_count: number;
  pending_count: number;
  avg_score: number;
  today_count: number;
  pass_rate: number;
}

export default function AdminDashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    try {
      const res = await apiClient.get("/api/v1/admin/stats/overview");
      setStats(res.data);
    } catch {
      // Fallback: compute from ideas list
      try {
        const res = await apiClient.get("/api/v1/ideas?page_size=1000");
        const items = res.data.items || [];
        const total = items.length;
        const debating = items.filter((i: { status: string }) => i.status === "debating").length;
        const graduated = items.filter((i: { status: string }) => i.status === "graduated").length;
        const failed = items.filter((i: { status: string }) => i.status === "failed").length;
        const pending = items.filter((i: { status: string }) => i.status === "pending").length;
        const scores = items
          .filter((i: { score_overall: number }) => i.score_overall > 0)
          .map((i: { score_overall: number }) => i.score_overall);
        const avgScore = scores.length > 0 ? scores.reduce((a: number, b: number) => a + b, 0) / scores.length : 0;

        const today = new Date().toISOString().split("T")[0];
        const todayCount = items.filter((i: { created_at: string }) => i.created_at?.startsWith(today)).length;
        const completed = graduated + failed;
        const passRate = completed > 0 ? (graduated / completed) * 100 : 0;

        setStats({
          total_ideas: total,
          debating_count: debating,
          graduated_count: graduated,
          failed_count: failed,
          pending_count: pending,
          avg_score: Math.round(avgScore * 10) / 10,
          today_count: todayCount,
          pass_rate: Math.round(passRate),
        });
      } catch {
        setStats({
          total_ideas: 0,
          debating_count: 0,
          graduated_count: 0,
          failed_count: 0,
          pending_count: 0,
          avg_score: 0,
          today_count: 0,
          pass_rate: 0,
        });
      }
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <Loading text="加载仪表盘..." />;

  return (
    <div className="space-y-6 max-w-7xl">
      <PageHeader title="仪表盘" description="Idea Arena 系统总览" />

      {/* Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          icon={<Activity className="h-5 w-5" />}
          label="引擎状态"
          value={stats!.debating_count > 0 ? "运行中" : "空闲"}
          sub={`${stats!.debating_count} 场进行中`}
          color="emerald"
        />
        <StatCard
          icon={<Zap className="h-5 w-5" />}
          label="今日产出"
          value={stats!.today_count}
          sub={`均分 ${stats!.avg_score}`}
          color="amber"
        />
        <StatCard
          icon={<BarChart3 className="h-5 w-5" />}
          label="累计辩论"
          value={stats!.total_ideas}
          sub={`通过率 ${stats!.pass_rate}%`}
          color="blue"
        />
        <StatCard
          icon={<Trophy className="h-5 w-5" />}
          label="毕业方案"
          value={stats!.graduated_count}
          sub={`${stats!.failed_count} 未通过`}
          color="violet"
        />
      </div>

      {/* Status distribution */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card title="状态分布">
          <div className="space-y-3">
            {[
              { label: "待处理", count: stats!.pending_count, color: "bg-yellow-500", total: stats!.total_ideas },
              { label: "辩论中", count: stats!.debating_count, color: "bg-blue-500", total: stats!.total_ideas },
              { label: "已毕业", count: stats!.graduated_count, color: "bg-emerald-500", total: stats!.total_ideas },
              { label: "未通过", count: stats!.failed_count, color: "bg-rose-500", total: stats!.total_ideas },
            ].map((item) => {
              const pct = item.total > 0 ? (item.count / item.total) * 100 : 0;
              return (
                <div key={item.label} className="space-y-1.5">
                  <div className="flex items-center justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <div className={`h-2.5 w-2.5 rounded-full ${item.color}`} />
                      <span className="font-medium">{item.label}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="font-bold">{item.count}</span>
                      <span className="text-xs text-muted-foreground w-10 text-right">{pct.toFixed(0)}%</span>
                    </div>
                  </div>
                  <div className="h-1.5 rounded-full bg-muted overflow-hidden">
                    <div className={`h-full rounded-full ${item.color} transition-all`} style={{ width: `${pct}%` }} />
                  </div>
                </div>
              );
            })}
          </div>
        </Card>

        <Card title="快捷操作">
          <div className="space-y-3">
            {[
              {
                icon: <Zap className="h-5 w-5 text-primary" />,
                title: "提交新点子",
                desc: "手动输入创业点子进行辩论",
                href: "/",
              },
              {
                icon: <TrendingUp className="h-5 w-5 text-emerald-500" />,
                title: "话题发现",
                desc: "从多渠道自动发现热门话题",
                href: "/admin/discovery",
              },
              {
                icon: <Clock className="h-5 w-5 text-amber-500" />,
                title: "辩论监控",
                desc: "查看正在进行的辩论状态",
                href: "/admin/debates",
              },
            ].map((action) => (
              <a
                key={action.href}
                href={action.href}
                className="flex items-center gap-4 rounded-xl border p-4 hover:bg-accent/50 transition-colors group"
              >
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted group-hover:bg-background transition-colors">
                  {action.icon}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold">{action.title}</p>
                  <p className="text-xs text-muted-foreground">{action.desc}</p>
                </div>
                <TrendingUp className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
              </a>
            ))}
          </div>
        </Card>
      </div>
    </div>
  );
}

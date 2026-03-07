"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { Swords, RefreshCw, ExternalLink, Clock, Zap } from "lucide-react";
import apiClient from "@/lib/api-client";
import { PageHeader, Card, StatusBadge, Loading, EmptyState, SecondaryButton } from "@/components/admin/shared";
import type { IdeaInfo } from "@/types/api";

export default function DebatesPage() {
  const [active, setActive] = useState<IdeaInfo[]>([]);
  const [recent, setRecent] = useState<IdeaInfo[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [debatingRes, recentRes] = await Promise.all([
        apiClient.get("/api/v1/ideas?status=debating&page_size=50"),
        apiClient.get("/api/v1/ideas?page_size=10&sort_by=updated_at&sort_order=desc"),
      ]);
      setActive(debatingRes.data.items || []);
      setRecent((recentRes.data.items || []).filter((i: IdeaInfo) => i.status !== "debating"));
    } catch {
      setActive([]);
      setRecent([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  if (loading) return <Loading text="加载辩论状态..." />;

  return (
    <div className="space-y-6 max-w-7xl">
      <PageHeader
        title="辩论监控"
        description="查看正在进行和最近完成的辩论"
        actions={
          <SecondaryButton onClick={load}>
            <RefreshCw className="h-4 w-4" /> 刷新
          </SecondaryButton>
        }
      />

      {/* Active debates */}
      <Card title={`正在进行 (${active.length})`} desc="当前引擎中运行的辩论">
        {active.length === 0 ? (
          <EmptyState
            icon={<Swords className="h-10 w-10" />}
            title="没有正在进行的辩论"
            description="提交新的点子或从话题发现中启动辩论"
          />
        ) : (
          <div className="grid gap-3">
            {active.map((idea) => (
              <div key={idea.id} className="flex items-center gap-4 rounded-xl border border-blue-500/30 bg-blue-500/5 p-4">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10">
                  <Swords className="h-5 w-5 text-blue-500" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold truncate">{idea.product_name || idea.topic}</p>
                  <div className="flex items-center gap-3 mt-0.5">
                    <span className="text-xs text-muted-foreground flex items-center gap-1">
                      <Clock className="h-3 w-3" /> R{idea.round_count}
                    </span>
                    {idea.score_overall > 0 && (
                      <span className="text-xs text-muted-foreground">当前评分: {idea.score_overall.toFixed(1)}</span>
                    )}
                  </div>
                </div>
                <Link
                  href={`/ideas/${idea.id}`}
                  className="inline-flex items-center gap-1 rounded-lg bg-blue-500/10 text-blue-600 px-3 py-1.5 text-xs font-medium hover:bg-blue-500/20 transition-colors"
                >
                  <ExternalLink className="h-3.5 w-3.5" /> 查看
                </Link>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Recently completed */}
      <Card title="最近完成" desc="最近完成的辩论结果">
        {recent.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">暂无最近完成的辩论</p>
        ) : (
          <div className="rounded-xl border overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">产品</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">状态</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">评分</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">轮次</th>
                  <th className="text-left px-4 py-3 font-medium text-muted-foreground">完成时间</th>
                  <th className="text-right px-4 py-3 font-medium text-muted-foreground">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {recent.map((idea) => (
                  <tr key={idea.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3">
                      <p className="font-medium truncate max-w-xs">{idea.product_name || idea.topic}</p>
                    </td>
                    <td className="px-4 py-3"><StatusBadge status={idea.status} /></td>
                    <td className="px-4 py-3">
                      {idea.score_overall > 0 ? (
                        <span className={`font-bold ${
                          idea.score_overall >= 8 ? "text-emerald-600" :
                          idea.score_overall >= 6 ? "text-blue-600" : "text-muted-foreground"
                        }`}>{idea.score_overall.toFixed(1)}</span>
                      ) : <span className="text-muted-foreground">—</span>}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">R{idea.round_count}</td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">
                      {new Date(idea.updated_at).toLocaleString("zh-CN")}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link href={`/ideas/${idea.id}`} className="text-primary hover:underline text-xs">查看 →</Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}

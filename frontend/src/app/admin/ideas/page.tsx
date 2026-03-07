"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { Search, Trash2, ExternalLink, ArrowUpDown } from "lucide-react";
import apiClient from "@/lib/api-client";
import { PageHeader, Card, StatusBadge, Loading, EmptyState } from "@/components/admin/shared";
import type { IdeaInfo } from "@/types/api";

export default function AdminIdeasPage() {
  const [ideas, setIdeas] = useState<IdeaInfo[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("");
  const [sortBy, setSortBy] = useState("created_at");
  const [sortOrder, setSortOrder] = useState("desc");
  const [page, setPage] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(page),
        page_size: "20",
        sort_by: sortBy,
        sort_order: sortOrder,
      });
      if (filter) params.set("status", filter);
      const res = await apiClient.get(`/api/v1/ideas?${params}`);
      let items = res.data.items || [];
      if (searchQuery) {
        const q = searchQuery.toLowerCase();
        items = items.filter(
          (i: IdeaInfo) =>
            i.topic.toLowerCase().includes(q) ||
            i.product_name?.toLowerCase().includes(q)
        );
      }
      setIdeas(items);
      setTotal(res.data.total || 0);
    } catch {
      setIdeas([]);
    } finally {
      setLoading(false);
    }
  }, [page, filter, sortBy, sortOrder, searchQuery]);

  useEffect(() => { load(); }, [load]);

  const deleteIdea = async (id: number) => {
    if (!confirm("确定删除此点子？")) return;
    try {
      await apiClient.delete(`/api/v1/ideas/${id}`);
      load();
    } catch {
      alert("删除失败");
    }
  };

  const toggleSort = (field: string) => {
    if (sortBy === field) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortBy(field);
      setSortOrder("desc");
    }
  };

  const totalPages = Math.ceil(total / 20);

  return (
    <div className="space-y-6 max-w-7xl">
      <PageHeader title="点子管理" description="查看和管理所有创业点子及辩论结果" />

      {/* Filters */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          {["", "pending", "debating", "graduated", "failed"].map((s) => (
            <button
              key={s}
              onClick={() => { setFilter(s); setPage(1); }}
              className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                filter === s ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:bg-accent"
              }`}
            >
              {s === "" ? "全部" : s === "pending" ? "待处理" : s === "debating" ? "辩论中" : s === "graduated" ? "已毕业" : "未通过"}
            </button>
          ))}
          <span className="text-xs text-muted-foreground ml-2">共 {total} 条</span>
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="搜索产品名或话题..."
            className="rounded-lg border bg-background pl-9 pr-3 py-2 text-sm w-64 focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
        </div>
      </div>

      {/* Table */}
      {loading ? (
        <Loading text="加载点子列表..." />
      ) : ideas.length === 0 ? (
        <EmptyState title="暂无点子" description="提交新的创业点子开始辩论" />
      ) : (
        <div className="rounded-xl border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">ID</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">产品</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">状态</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground cursor-pointer" onClick={() => toggleSort("score_overall")}>
                  <span className="inline-flex items-center gap-1">评分 <ArrowUpDown className="h-3 w-3" /></span>
                </th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">轮次</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground cursor-pointer" onClick={() => toggleSort("created_at")}>
                  <span className="inline-flex items-center gap-1">创建时间 <ArrowUpDown className="h-3 w-3" /></span>
                </th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {ideas.map((idea) => (
                <tr key={idea.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-3 text-muted-foreground font-mono text-xs">#{idea.id}</td>
                  <td className="px-4 py-3">
                    <div className="max-w-xs">
                      <p className="font-medium truncate">{idea.product_name || idea.topic}</p>
                      {idea.one_liner && (
                        <p className="text-xs text-muted-foreground truncate mt-0.5">{idea.one_liner}</p>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3"><StatusBadge status={idea.status} /></td>
                  <td className="px-4 py-3">
                    {idea.score_overall > 0 ? (
                      <span className={`font-bold ${
                        idea.score_overall >= 8 ? "text-emerald-600" :
                        idea.score_overall >= 6 ? "text-blue-600" :
                        "text-muted-foreground"
                      }`}>
                        {idea.score_overall.toFixed(1)}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">R{idea.round_count}</td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">
                    {new Date(idea.created_at).toLocaleDateString("zh-CN")}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Link
                        href={`/ideas/${idea.id}`}
                        className="rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                        title="查看详情"
                      >
                        <ExternalLink className="h-4 w-4" />
                      </Link>
                      <button
                        onClick={() => deleteIdea(idea.id)}
                        className="rounded-md p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                        title="删除"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page <= 1}
            className="rounded-lg border px-3 py-1.5 text-sm disabled:opacity-50"
          >
            上一页
          </button>
          <span className="text-sm text-muted-foreground">
            {page} / {totalPages}
          </span>
          <button
            onClick={() => setPage(Math.min(totalPages, page + 1))}
            disabled={page >= totalPages}
            className="rounded-lg border px-3 py-1.5 text-sm disabled:opacity-50"
          >
            下一页
          </button>
        </div>
      )}
    </div>
  );
}

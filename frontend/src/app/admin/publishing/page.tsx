"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Sparkles, Edit3, Send, Trash2, Eye, FileText, CheckCircle,
  XCircle, Clock, Share2, Plus,
} from "lucide-react";
import apiClient from "@/lib/api-client";
import {
  PageHeader, Card, StatusBadge, Loading, EmptyState,
  PrimaryButton, SecondaryButton, Field,
} from "@/components/admin/shared";
import type { IdeaInfo } from "@/types/api";

interface SocialPost {
  id: number;
  idea_id: number;
  platform: string;
  title: string;
  content: string;
  status: string; // draft, review, published, rejected
  published_at: string | null;
  created_at: string;
  idea_name?: string;
}

const PLATFORMS = [
  { id: "xiaohongshu", label: "小红书", emoji: "📕" },
  { id: "wechat", label: "公众号", emoji: "💬" },
  { id: "weibo", label: "微博", emoji: "🔥" },
  { id: "twitter", label: "Twitter/X", emoji: "🐦" },
  { id: "zhihu", label: "知乎", emoji: "💡" },
];

export default function PublishingPage() {
  const [posts, setPosts] = useState<SocialPost[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("");
  const [showGenerate, setShowGenerate] = useState(false);
  const [editingPost, setEditingPost] = useState<SocialPost | null>(null);

  // For generating new posts
  const [ideas, setIdeas] = useState<IdeaInfo[]>([]);
  const [selectedIdeaId, setSelectedIdeaId] = useState<number>(0);
  const [selectedPlatform, setSelectedPlatform] = useState("xiaohongshu");
  const [generating, setGenerating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiClient.get("/api/v1/admin/posts" + (filter ? `?status=${filter}` : ""));
      setPosts(res.data.items || res.data || []);
    } catch {
      // API not implemented yet — show empty state
      setPosts([]);
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => { load(); }, [load]);

  const loadIdeas = async () => {
    try {
      const res = await apiClient.get("/api/v1/ideas?status=graduated&page_size=50&sort_by=score_overall&sort_order=desc");
      setIdeas(res.data.items || []);
    } catch {
      setIdeas([]);
    }
  };

  const generatePost = async () => {
    if (!selectedIdeaId) return;
    setGenerating(true);
    try {
      await apiClient.post("/api/v1/admin/posts/generate", {
        idea_id: selectedIdeaId,
        platform: selectedPlatform,
      });
      setShowGenerate(false);
      load();
    } catch {
      alert("生成失败（后端 API 可能尚未实现）");
    } finally {
      setGenerating(false);
    }
  };

  const updatePost = async (post: SocialPost) => {
    try {
      await apiClient.put(`/api/v1/admin/posts/${post.id}`, {
        title: post.title,
        content: post.content,
      });
      setEditingPost(null);
      load();
    } catch {
      alert("更新失败");
    }
  };

  const publishPost = async (id: number) => {
    try {
      await apiClient.post(`/api/v1/admin/posts/${id}/publish`);
      load();
    } catch {
      alert("发布失败");
    }
  };

  const deletePost = async (id: number) => {
    if (!confirm("确定删除此推文？")) return;
    try {
      await apiClient.delete(`/api/v1/admin/posts/${id}`);
      load();
    } catch {
      alert("删除失败");
    }
  };

  const platformEmoji = (p: string) => PLATFORMS.find((pl) => pl.id === p)?.emoji || "📱";
  const platformLabel = (p: string) => PLATFORMS.find((pl) => pl.id === p)?.label || p;

  return (
    <div className="space-y-6 max-w-5xl">
      <PageHeader
        title="社交推文"
        description="将优秀创业方案生成推文，人工复审后发布到社交平台"
        actions={
          <PrimaryButton
            onClick={() => {
              setShowGenerate(!showGenerate);
              if (!showGenerate) loadIdeas();
            }}
          >
            <Sparkles className="h-4 w-4" /> AI 生成推文
          </PrimaryButton>
        }
      />

      {/* Generate form */}
      {showGenerate && (
        <Card title="生成推文" desc="选择一个毕业方案，AI 将根据平台风格生成推文草稿">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="sm:col-span-2 space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">选择方案</label>
              <select
                value={selectedIdeaId}
                onChange={(e) => setSelectedIdeaId(Number(e.target.value))}
                className="w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
              >
                <option value={0}>请选择一个毕业方案...</option>
                {ideas.map((idea) => (
                  <option key={idea.id} value={idea.id}>
                    [{idea.score_overall.toFixed(1)}分] {idea.product_name || idea.topic}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">目标平台</label>
              <select
                value={selectedPlatform}
                onChange={(e) => setSelectedPlatform(e.target.value)}
                className="w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
              >
                {PLATFORMS.map((p) => (
                  <option key={p.id} value={p.id}>{p.emoji} {p.label}</option>
                ))}
              </select>
            </div>
          </div>
          <div className="flex gap-2 pt-2">
            <PrimaryButton onClick={generatePost} disabled={generating || !selectedIdeaId} loading={generating}>
              <Sparkles className="h-4 w-4" /> 生成草稿
            </PrimaryButton>
            <button onClick={() => setShowGenerate(false)} className="text-sm text-muted-foreground hover:text-foreground">取消</button>
          </div>
        </Card>
      )}

      {/* Filter tabs */}
      <div className="flex items-center gap-2">
        {["", "draft", "review", "published", "rejected"].map((s) => (
          <button
            key={s}
            onClick={() => setFilter(s)}
            className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
              filter === s ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:bg-accent"
            }`}
          >
            {s === "" ? "全部" : s === "draft" ? "草稿" : s === "review" ? "待审核" : s === "published" ? "已发布" : "已拒绝"}
          </button>
        ))}
      </div>

      {/* Edit modal */}
      {editingPost && (
        <Card title="编辑推文" desc={`${platformEmoji(editingPost.platform)} ${platformLabel(editingPost.platform)}`}>
          <Field label="标题" value={editingPost.title} onChange={(v) => setEditingPost({ ...editingPost, title: v })} />
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">内容</label>
            <textarea
              value={editingPost.content}
              onChange={(e) => setEditingPost({ ...editingPost, content: e.target.value })}
              rows={10}
              className="w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 resize-y"
            />
          </div>
          <div className="flex gap-2">
            <PrimaryButton onClick={() => updatePost(editingPost)}>
              <CheckCircle className="h-4 w-4" /> 保存
            </PrimaryButton>
            <SecondaryButton onClick={() => publishPost(editingPost.id)}>
              <Send className="h-4 w-4" /> 保存并发布
            </SecondaryButton>
            <button onClick={() => setEditingPost(null)} className="text-sm text-muted-foreground hover:text-foreground ml-2">取消</button>
          </div>
        </Card>
      )}

      {/* Posts list */}
      {loading ? (
        <Loading text="加载推文..." />
      ) : posts.length === 0 ? (
        <EmptyState
          icon={<Share2 className="h-10 w-10" />}
          title="暂无推文"
          description="选择一个毕业方案，使用 AI 生成推文草稿"
          action={
            <SecondaryButton onClick={() => { setShowGenerate(true); loadIdeas(); }}>
              <Plus className="h-4 w-4" /> 生成第一篇推文
            </SecondaryButton>
          }
        />
      ) : (
        <div className="space-y-3">
          {posts.map((post) => (
            <div key={post.id} className="rounded-xl border bg-card p-4 space-y-3">
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-lg">{platformEmoji(post.platform)}</span>
                    <StatusBadge status={post.status} />
                    {post.idea_name && (
                      <span className="text-xs text-muted-foreground">关联: {post.idea_name}</span>
                    )}
                  </div>
                  <h3 className="font-medium text-sm mt-1">{post.title}</h3>
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  {post.status !== "published" && (
                    <>
                      <button
                        onClick={() => setEditingPost(post)}
                        className="rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                        title="编辑"
                      >
                        <Edit3 className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => publishPost(post.id)}
                        className="rounded-md p-1.5 text-muted-foreground hover:text-emerald-600 hover:bg-emerald-500/10 transition-colors"
                        title="发布"
                      >
                        <Send className="h-4 w-4" />
                      </button>
                    </>
                  )}
                  <button
                    onClick={() => deletePost(post.id)}
                    className="rounded-md p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                    title="删除"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed line-clamp-3 whitespace-pre-line">
                {post.content}
              </p>
              <div className="flex items-center gap-3 text-[10px] text-muted-foreground">
                <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> {new Date(post.created_at).toLocaleString("zh-CN")}</span>
                {post.published_at && (
                  <span className="flex items-center gap-1"><CheckCircle className="h-3 w-3" /> 发布于 {new Date(post.published_at).toLocaleString("zh-CN")}</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Zap, Sparkles, Target, TrendingUp, Send, Loader2, ArrowRight, Search, Shield, BarChart3 } from "lucide-react";
import { motion } from "motion/react";
import apiClient from "@/lib/api-client";
import { useAuthStore } from "@/stores/auth-store";

const EXAMPLES = [
  "一个基于 AI 的个人健康管理助手 App",
  "面向 Z 世代的 Web3 社交平台",
  "企业级 AI 代码审计工具",
  "智能家居能源优化系统",
  "短视频创作者的 AI 剪辑助手",
  "面向独立开发者的 SaaS 模板市场",
];

export default function HomePage() {
  const [topic, setTopic] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState<{ id: number; name: string } | null>(null);
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!topic.trim()) return;

    if (!isAuthenticated) {
      router.push("/login");
      return;
    }

    setLoading(true);
    setError("");
    setSuccess(null);
    try {
      const res = await apiClient.post("/api/v1/ideas", { topic: topic.trim() });
      setSuccess({ id: res.data.id, name: topic.trim() });
      setTopic("");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "提交失败，请重试";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-[calc(100vh-10rem)]">
      <div className="max-w-2xl w-full space-y-10 text-center">

        {/* Hero */}
        <motion.div initial={{ opacity: 0, y: 30 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.8 }} className="space-y-5">
          <motion.div initial={{ scale: 0.8, opacity: 0 }} animate={{ scale: 1, opacity: 1 }} transition={{ delay: 0.2 }}
            className="inline-flex items-center gap-2 rounded-full bg-primary/10 border border-primary/20 px-4 py-1.5 text-sm font-semibold text-primary">
            <Sparkles className="h-4 w-4" />
            灵光一现
          </motion.div>
          <h1 className="text-5xl sm:text-6xl font-extrabold tracking-tight leading-[1.1]">
            你的点子，<br />
            <span className="text-primary">AI 来辩论</span>
          </h1>
          <p className="text-lg text-muted-foreground leading-relaxed max-w-lg mx-auto font-medium">
            输入任何创业灵感，多个 AI Agent 将围绕它展开严格辩论，
            给出可行性、经济性、利润潜力的三维评分。
          </p>
        </motion.div>

        {/* Submit Form */}
        <motion.form onSubmit={handleSubmit} initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3, duration: 0.6 }}
          className="space-y-4">
          <div className="relative rounded-2xl border bg-card shadow-lg shadow-primary/5 focus-within:ring-2 focus-within:ring-primary/40 transition-shadow">
            <textarea
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              placeholder="描述你的创业点子..."
              rows={4}
              className="w-full rounded-2xl bg-transparent p-5 pb-14 text-base placeholder:text-muted-foreground focus:outline-none resize-none"
            />
            <div className="absolute bottom-3 right-3">
              <button
                type="submit"
                disabled={loading || !topic.trim()}
                className="inline-flex items-center gap-2 rounded-xl bg-primary px-6 py-2.5 text-sm font-bold text-primary-foreground hover:bg-primary/90 disabled:opacity-40 disabled:cursor-not-allowed transition-all shadow-lg shadow-primary/25"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                {loading ? "提交中..." : "开始辩论"}
              </button>
            </div>
          </div>

          {error && (
            <motion.p initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm text-destructive font-medium">{error}</motion.p>
          )}

          {success && (
            <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
              className="rounded-2xl border border-green-500/25 bg-green-500/5 p-5 text-left space-y-3">
              <div className="flex items-center gap-2 text-green-600 dark:text-green-400 text-sm font-bold">
                <Sparkles className="w-4 h-4" /> 提交成功！AI 辩论即将开始
              </div>
              <p className="text-sm text-muted-foreground">
                &ldquo;{success.name}&rdquo; 已进入辩论队列，多个 AI Agent 将围绕它展开讨论。
              </p>
              <Link href={`/ideas/${success.id}`}
                className="inline-flex items-center gap-1.5 text-sm font-bold text-primary hover:underline">
                前往查看辩论过程 <ArrowRight className="w-3.5 h-3.5" />
              </Link>
            </motion.div>
          )}
        </motion.form>

        {/* Example Chips */}
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.5 }} className="space-y-3">
          <p className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">灵感示例 — 点击即可填入</p>
          <div className="flex flex-wrap justify-center gap-2">
            {EXAMPLES.map((ex) => (
              <button key={ex} onClick={() => setTopic(ex)}
                className="px-3.5 py-1.5 rounded-full text-xs font-medium bg-secondary text-secondary-foreground hover:bg-secondary/70 transition-colors border border-transparent hover:border-primary/20">
                {ex}
              </button>
            ))}
          </div>
        </motion.div>

        {/* Features */}
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.6 }}
          className="grid grid-cols-1 sm:grid-cols-4 gap-4 pt-4">
          {[
            { icon: <Search className="w-5 h-5" />, title: "真实搜索", desc: "搜索引擎采集真实数据" },
            { icon: <Target className="w-5 h-5" />, title: "多轮辩论", desc: "正反 Agent 深度对局" },
            { icon: <Shield className="w-5 h-5" />, title: "严格评审", desc: "三维评分体系量化评估" },
            { icon: <Sparkles className="w-5 h-5" />, title: "交付物", desc: "报告 + 技术栈 + Prompt" },
          ].map((f, i) => (
            <div key={i} className="rounded-2xl border bg-card p-5 text-center space-y-2 hover:shadow-md transition-shadow">
              <div className="w-10 h-10 rounded-xl bg-primary/10 border border-primary/20 flex items-center justify-center text-primary mx-auto">{f.icon}</div>
              <h3 className="text-sm font-bold">{f.title}</h3>
              <p className="text-xs text-muted-foreground">{f.desc}</p>
            </div>
          ))}
        </motion.div>
      </div>
    </div>
  );
}

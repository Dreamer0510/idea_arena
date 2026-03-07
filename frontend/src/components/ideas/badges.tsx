"use client";

import { Zap, Tag } from "lucide-react";

export function VerdictBadge({ verdict, size = "sm" }: { verdict: string; size?: "sm" | "md" }) {
  const c = {
    graduated: { cls: "bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/25", label: "可行" },
    promising: { cls: "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/25", label: "有潜力" },
    failed: { cls: "bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/25", label: "不可行" },
    pending: { cls: "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border-yellow-500/25", label: "评估中" },
    debating: { cls: "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/25", label: "辩论中" },
  }[verdict] || { cls: "bg-muted text-muted-foreground border-border", label: verdict };
  const padding = size === "md" ? "px-3 py-1.5 text-xs" : "px-2.5 py-1 text-[11px]";
  return (
    <span className={`inline-flex items-center rounded-full font-semibold tracking-wide border ${c.cls} ${padding} transition-all`}>
      {c.label}
    </span>
  );
}

export function StatusBadge({ status, rounds }: { status: string | null; rounds: number }) {
  if (status !== "pending" && status !== "debating") return null;
  return (
    <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/25 animate-pulse tracking-wider uppercase">
      <Zap className="w-3 h-3" />
      {status === "pending" ? "排队中" : `辩论 R${rounds}`}
    </span>
  );
}

export function ScoreBadge({ score }: { score: number }) {
  const colorClass = score >= 8 ? "text-green-600 dark:text-green-400 ring-green-500/20"
    : score >= 6 ? "text-amber-600 dark:text-amber-400 ring-amber-500/20"
      : "text-red-600 dark:text-red-400 ring-red-500/20";
  return (
    <div className={`flex flex-col items-center justify-center w-12 h-12 rounded-xl bg-card border ring-2 ${colorClass} transition-all`}>
      <span className="text-sm font-black font-mono tabular-nums">{score > 0 ? score.toFixed(1) : "-"}</span>
      <span className="text-[9px] font-semibold text-muted-foreground uppercase tracking-widest">得分</span>
    </div>
  );
}

export function TagChip({ tag }: { tag: string }) {
  return (
    <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[11px] font-medium bg-secondary text-secondary-foreground hover:bg-secondary/80 transition-colors">
      <Tag className="w-3 h-3 opacity-50" />{tag}
    </span>
  );
}

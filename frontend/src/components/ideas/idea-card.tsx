"use client";

import Link from "next/link";
import { ArrowUpRight, Zap } from "lucide-react";
import { motion } from "motion/react";
import type { IdeaInfo } from "@/types/api";
import { VerdictBadge, StatusBadge, ScoreBadge } from "./badges";

export function IdeaCard({ idea, index = 0 }: { idea: IdeaInfo; index?: number }) {
  const isActive = idea.status === "pending" || idea.status === "debating";
  const name = idea.product_name || idea.topic;

  const glowClass = idea.score_overall >= 8 ? "hover:shadow-green-500/10 hover:border-green-500/30"
    : idea.score_overall >= 6 ? "hover:shadow-amber-500/10 hover:border-amber-500/30"
      : "";

  return (
    <motion.div
      initial={{ opacity: 0, y: 24 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: Math.min(index * 0.06, 0.6), duration: 0.5, ease: "easeOut" }}
    >
      <Link href={`/ideas/${idea.id}`} className="block h-full">
        <div className={`group h-full flex flex-col rounded-2xl border bg-card p-5 transition-all duration-300 hover:-translate-y-1.5 hover:shadow-xl ${glowClass}`}>

          {/* Header: Badges + Score */}
          <div className="flex justify-between items-start mb-4">
            <div className="flex gap-2 flex-wrap">
              <VerdictBadge verdict={idea.status} />
              {isActive && <StatusBadge status={idea.status} rounds={idea.round_count} />}
            </div>
            {!isActive && idea.score_overall > 0 && <ScoreBadge score={idea.score_overall} />}
          </div>

          {/* Title */}
          <h3 className="font-semibold text-lg leading-tight line-clamp-2 mb-2 group-hover:text-primary transition-colors">
            {name}
          </h3>

          {/* Description */}
          <p className="text-sm text-muted-foreground line-clamp-3 mb-6 flex-1 leading-relaxed">
            {idea.one_liner || idea.topic}
          </p>

          {/* Footer */}
          <div className="flex items-center justify-between text-xs text-muted-foreground mt-auto pt-4 border-t">
            <div className="flex gap-2">
              {idea.tags && idea.tags.slice(0, 2).map(t => (
                <span key={t} className="bg-secondary px-2 py-0.5 rounded-md text-[10px] text-secondary-foreground">#{t}</span>
              ))}
            </div>
            <div className="flex items-center gap-3">
              {idea.round_count > 0 && (
                <span className="font-mono text-muted-foreground opacity-70">R{idea.round_count}</span>
              )}
              <ArrowUpRight className="w-3.5 h-3.5 text-muted-foreground group-hover:text-primary transition-colors opacity-0 group-hover:opacity-100" />
            </div>
          </div>

        </div>
      </Link>
    </motion.div>
  );
}

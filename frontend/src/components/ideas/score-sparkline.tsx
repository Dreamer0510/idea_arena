"use client";

import { motion } from "motion/react";

export function ScoreSparkline({ scores }: { scores: { round: number; score: number }[] }) {
  if (scores.length < 2) return null;
  const w = 240, h = 60;
  const pad = { t: 10, r: 30, b: 16, l: 30 };
  const chartW = w - pad.l - pad.r;
  const chartH = h - pad.t - pad.b;
  const minS = Math.min(...scores.map(s => s.score), 0);
  const maxS = Math.max(...scores.map(s => s.score), 10);
  const range = maxS - minS || 1;
  const points = scores.map((s, i) => ({
    x: pad.l + (i / (scores.length - 1)) * chartW,
    y: pad.t + chartH - ((s.score - minS) / range) * chartH,
    score: s.score, round: s.round,
  }));
  const pathD = points.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ");
  const areaD = pathD + ` L ${points[points.length - 1].x} ${pad.t + chartH} L ${points[0].x} ${pad.t + chartH} Z`;
  const delta = scores[scores.length - 1].score - scores[0].score;

  return (
    <div className="hidden sm:flex items-center gap-4 rounded-2xl border bg-card p-3">
      <svg width={w} height={h} className="shrink-0">
        <line x1={pad.l} y1={pad.t + chartH - ((8 - minS) / range) * chartH}
          x2={w - pad.r} y2={pad.t + chartH - ((8 - minS) / range) * chartH}
          stroke="rgb(34,197,94)" strokeOpacity="0.3" strokeDasharray="4 4" strokeWidth="1" />
        <text x={w - pad.r + 4} y={pad.t + chartH - ((8 - minS) / range) * chartH + 3}
          fill="rgb(34,197,94)" fillOpacity="0.5" fontSize="8" fontWeight="bold">8.0</text>
        <path d={areaD} fill="url(#sparkGrad)" opacity="0.2" />
        <defs>
          <linearGradient id="sparkGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="hsl(var(--primary))" />
            <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity="0" />
          </linearGradient>
        </defs>
        <motion.path d={pathD} fill="none" stroke="hsl(var(--primary))" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
          initial={{ pathLength: 0 }} animate={{ pathLength: 1 }}
          transition={{ duration: 1.5, ease: "easeOut" }} />
        {points.map((p, i) => (
          <g key={i}>
            <circle cx={p.x} cy={p.y} r="3" fill="hsl(var(--background))" stroke="hsl(var(--primary))" strokeWidth="2" />
            <text x={p.x} y={p.y - 8} textAnchor="middle" fill="hsl(var(--foreground))" fontSize="9" fontWeight="bold">{p.score}</text>
          </g>
        ))}
        {points.map((p, i) => (
          <text key={`l-${i}`} x={p.x} y={h - 2} textAnchor="middle" fill="hsl(var(--muted-foreground))" fontSize="8" fontWeight="500">R{scores[i].round}</text>
        ))}
      </svg>
      <div className="text-right shrink-0">
        <div className={`text-sm font-black font-mono tabular-nums tracking-tight ${delta > 0 ? "text-green-500" : delta < 0 ? "text-red-500" : "text-muted-foreground"}`}>
          {delta > 0 ? "+" : ""}{delta.toFixed(1)}
        </div>
        <div className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mt-0.5">总变化</div>
      </div>
    </div>
  );
}

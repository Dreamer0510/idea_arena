"use client";

import { motion } from "motion/react";

export function ScoreRing({ score, label, color, size = "md" }: { 
  score: number; 
  label: string; 
  color: string;
  size?: "sm" | "md";
}) {
  const pct = (score / 10) * 100;
  const radius = size === "sm" ? 22 : 28;
  const viewBox = size === "sm" ? 52 : 64;
  const center = viewBox / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (pct / 100) * circumference;

  const sizeClass = size === "sm" ? "w-12 h-12" : "w-16 h-16 sm:w-20 sm:h-20";
  const textClass = size === "sm" ? "text-sm font-bold" : "text-lg sm:text-xl font-black";

  return (
    <div className="flex flex-col items-center gap-2">
      <div className={`relative ${sizeClass}`} style={{ filter: `drop-shadow(0 0 8px ${color}30)` }}>
        <svg className="w-full h-full -rotate-90" viewBox={`0 0 ${viewBox} ${viewBox}`}>
          <circle cx={center} cy={center} r={radius} fill="none" stroke="hsl(var(--border))" strokeWidth="3" />
          <motion.circle
            cx={center} cy={center} r={radius} fill="none" stroke={color} strokeWidth="3.5"
            strokeLinecap="round" strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }}
            animate={{ strokeDashoffset: offset }}
            transition={{ duration: 1.5, ease: "easeOut" }}
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className={`${textClass} font-mono tabular-nums tracking-tighter`} style={{ color }}>
            {score > 0 ? score.toFixed(1) : "-"}
          </span>
        </div>
      </div>
      <span className="text-[10px] sm:text-xs font-semibold text-muted-foreground uppercase tracking-wider">{label}</span>
    </div>
  );
}

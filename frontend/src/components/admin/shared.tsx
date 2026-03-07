"use client";

import { Loader2 } from "lucide-react";
import type { ReactNode } from "react";

// ============================================================
// StatCard — 统计卡片
// ============================================================
export function StatCard({
  icon,
  label,
  value,
  sub,
  color = "primary",
}: {
  icon: ReactNode;
  label: string;
  value: string | number;
  sub?: string;
  color?: "primary" | "emerald" | "amber" | "rose" | "blue" | "violet";
}) {
  const colorMap: Record<string, string> = {
    primary: "bg-primary/10 text-primary border-primary/20",
    emerald: "bg-emerald-500/10 text-emerald-600 border-emerald-500/20",
    amber: "bg-amber-500/10 text-amber-600 border-amber-500/20",
    rose: "bg-rose-500/10 text-rose-600 border-rose-500/20",
    blue: "bg-blue-500/10 text-blue-600 border-blue-500/20",
    violet: "bg-violet-500/10 text-violet-600 border-violet-500/20",
  };

  return (
    <div className="rounded-xl border bg-card p-5 space-y-3">
      <div className="flex items-center gap-3">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg border ${colorMap[color]}`}>
          {icon}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium text-muted-foreground">{label}</p>
          <p className="text-2xl font-bold tracking-tight">{value}</p>
          {sub && <p className="text-xs text-muted-foreground mt-0.5">{sub}</p>}
        </div>
      </div>
    </div>
  );
}

// ============================================================
// PageHeader — 页面标题
// ============================================================
export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">{title}</h1>
        {description && <p className="text-sm text-muted-foreground">{description}</p>}
      </div>
      {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
    </div>
  );
}

// ============================================================
// Card — 内容卡片
// ============================================================
export function Card({
  title,
  desc,
  actions,
  children,
  className = "",
}: {
  title?: string;
  desc?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={`rounded-xl border bg-card p-5 space-y-4 ${className}`}>
      {(title || actions) && (
        <div className="flex items-start justify-between gap-3">
          <div>
            {title && <h2 className="font-semibold text-base">{title}</h2>}
            {desc && <p className="text-xs text-muted-foreground mt-0.5">{desc}</p>}
          </div>
          {actions && <div className="shrink-0">{actions}</div>}
        </div>
      )}
      {children}
    </div>
  );
}

// ============================================================
// StatusBadge — 状态标签
// ============================================================
const STATUS_COLORS: Record<string, string> = {
  pending: "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400",
  debating: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  graduated: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  promising: "bg-cyan-500/10 text-cyan-600 dark:text-cyan-400",
  failed: "bg-rose-500/10 text-rose-500 dark:text-rose-400",
  draft: "bg-slate-500/10 text-slate-600 dark:text-slate-400",
  review: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  published: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  rejected: "bg-rose-500/10 text-rose-500 dark:text-rose-400",
  recommended: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  submitted: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  dismissed: "bg-slate-500/10 text-slate-500 dark:text-slate-400",
  enabled: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  disabled: "bg-slate-500/10 text-slate-500 dark:text-slate-400",
};

const STATUS_LABELS: Record<string, string> = {
  pending: "待处理",
  debating: "辩论中",
  graduated: "已毕业",
  promising: "有潜力",
  failed: "未通过",
  draft: "草稿",
  review: "待审核",
  published: "已发布",
  rejected: "已拒绝",
  recommended: "已推荐",
  submitted: "已提交",
  dismissed: "已忽略",
  enabled: "已启用",
  disabled: "已禁用",
};

export function StatusBadge({ status, label }: { status: string; label?: string }) {
  const colorClass = STATUS_COLORS[status] || "bg-muted text-muted-foreground";
  const text = label || STATUS_LABELS[status] || status;
  return (
    <span className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium ${colorClass}`}>
      {text}
    </span>
  );
}

// ============================================================
// Loading — 加载状态
// ============================================================
export function Loading({ text }: { text?: string }) {
  return (
    <div className="flex items-center justify-center py-16 gap-2 text-muted-foreground">
      <Loader2 className="h-5 w-5 animate-spin" />
      {text && <span className="text-sm">{text}</span>}
    </div>
  );
}

// ============================================================
// EmptyState — 空状态
// ============================================================
export function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center space-y-3">
      {icon && <div className="text-muted-foreground/40">{icon}</div>}
      <div className="space-y-1">
        <p className="text-sm font-medium">{title}</p>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
      </div>
      {action}
    </div>
  );
}

// ============================================================
// Field — 表单字段
// ============================================================
export function Field({
  label,
  value,
  onChange,
  type = "text",
  placeholder,
  step,
  disabled,
}: {
  label: string;
  value: string | number;
  onChange: (v: string) => void;
  type?: string;
  placeholder?: string;
  step?: string;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-1.5">
      <label className="text-xs font-medium text-muted-foreground">{label}</label>
      <input
        type={type}
        step={step}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        className="w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50 disabled:cursor-not-allowed"
      />
    </div>
  );
}

// ============================================================
// Toggle — 开关
// ============================================================
export function Toggle({
  label,
  checked,
  onChange,
  disabled,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <label className="inline-flex items-center gap-2 cursor-pointer select-none">
      <button
        type="button"
        onClick={() => !disabled && onChange(!checked)}
        disabled={disabled}
        className={`w-9 h-5 rounded-full transition-colors relative ${
          checked ? "bg-primary" : "bg-muted"
        } ${disabled ? "opacity-50 cursor-not-allowed" : ""}`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform ${
            checked ? "left-[18px]" : "left-0.5"
          }`}
        />
      </button>
      <span className="text-sm">{label}</span>
    </label>
  );
}

// ============================================================
// Button variants
// ============================================================
export function PrimaryButton({
  children,
  onClick,
  disabled,
  loading,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  loading?: boolean;
  className?: string;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled || loading}
      className={`inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors ${className}`}
    >
      {loading && <Loader2 className="h-4 w-4 animate-spin" />}
      {children}
    </button>
  );
}

export function SecondaryButton({
  children,
  onClick,
  disabled,
  loading,
  className = "",
}: {
  children: ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  loading?: boolean;
  className?: string;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled || loading}
      className={`inline-flex items-center gap-2 rounded-lg bg-muted px-4 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground disabled:opacity-50 disabled:cursor-not-allowed transition-colors ${className}`}
    >
      {loading && <Loader2 className="h-4 w-4 animate-spin" />}
      {children}
    </button>
  );
}

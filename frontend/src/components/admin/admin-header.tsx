"use client";

import { usePathname } from "next/navigation";
import { Menu } from "lucide-react";

const BREADCRUMB_MAP: Record<string, string> = {
  "/admin": "仪表盘",
  "/admin/ideas": "点子管理",
  "/admin/debates": "辩论监控",
  "/admin/discovery": "话题发现",
  "/admin/models": "模型配置",
  "/admin/agents": "Agent 人设",
  "/admin/publishing": "社交推文",
  "/admin/users": "用户管理",
  "/admin/settings": "系统设置",
};

export function AdminHeader({ onToggleSidebar }: { onToggleSidebar?: () => void }) {
  const pathname = usePathname();

  const title = BREADCRUMB_MAP[pathname] || "管理后台";

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b bg-background/80 backdrop-blur-sm px-6">
      {onToggleSidebar && (
        <button
          onClick={onToggleSidebar}
          className="lg:hidden text-muted-foreground hover:text-foreground transition-colors"
        >
          <Menu className="h-5 w-5" />
        </button>
      )}
      <div className="flex items-center gap-2 text-sm">
        <span className="text-muted-foreground">管理后台</span>
        <span className="text-muted-foreground">/</span>
        <span className="font-semibold">{title}</span>
      </div>
    </header>
  );
}

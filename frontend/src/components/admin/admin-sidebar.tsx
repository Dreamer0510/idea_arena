"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard, Lightbulb, Swords, Compass, Bot, Server,
  Share2, Users, Settings, ChevronLeft, ChevronRight, Zap, LogOut,
  Moon, Sun, Monitor, UserCog,
} from "lucide-react";
import { useTheme } from "next-themes";
import { useAuthStore } from "@/stores/auth-store";
import { useSidebarStore } from "@/stores/sidebar-store";

const NAV_SECTIONS = [
  {
    title: "概览",
    items: [
      { href: "/admin", icon: LayoutDashboard, label: "仪表盘" },
      { href: "/admin/ideas", icon: Lightbulb, label: "点子管理" },
      { href: "/admin/debates", icon: Swords, label: "辩论监控" },
    ],
  },
  {
    title: "话题发现",
    items: [
      { href: "/admin/discovery", icon: Compass, label: "话题发现" },
    ],
  },
  {
    title: "AI 配置",
    items: [
      { href: "/admin/providers", icon: Server, label: "服务商管理" },
      { href: "/admin/agents", icon: Bot, label: "Agent 人设" },
    ],
  },
  {
    title: "运营",
    items: [
      { href: "/admin/publishing", icon: Share2, label: "社交推文" },
    ],
  },
  {
    title: "系统",
    items: [
      { href: "/admin/account", icon: UserCog, label: "账号管理" },
      { href: "/admin/users", icon: Users, label: "用户管理" },
      { href: "/admin/settings", icon: Settings, label: "系统设置" },
    ],
  },
];

export function AdminSidebar() {
  const pathname = usePathname();
  const { theme, setTheme } = useTheme();
  const { user, logout } = useAuthStore();
  const { collapsed, toggle: toggleCollapsed } = useSidebarStore();

  const isActive = (href: string) => {
    if (href === "/admin") return pathname === "/admin";
    return pathname.startsWith(href);
  };

  return (
    <aside
      className={`fixed inset-y-0 left-0 z-40 flex flex-col border-r bg-card transition-all duration-300 ${
        collapsed ? "w-[68px]" : "w-[240px]"
      }`}
    >
      {/* Logo */}
      <div className="flex h-16 items-center gap-2.5 border-b px-4">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 border border-primary/20">
          <Zap className="h-5 w-5 text-primary" />
        </div>
        {!collapsed && (
          <div className="flex flex-col overflow-hidden">
            <span className="text-sm font-bold tracking-tight truncate">Idea Arena</span>
            <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-widest">Admin</span>
          </div>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-6">
        {NAV_SECTIONS.map((section) => (
          <div key={section.title}>
            {!collapsed && (
              <p className="mb-2 px-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
                {section.title}
              </p>
            )}
            <div className="space-y-0.5">
              {section.items.map((item) => {
                const Icon = item.icon;
                const active = isActive(item.href);
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    title={collapsed ? item.label : undefined}
                    className={`flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm font-medium transition-colors ${
                      active
                        ? "bg-primary/10 text-primary border border-primary/20"
                        : "text-muted-foreground hover:bg-accent hover:text-accent-foreground border border-transparent"
                    }`}
                  >
                    <Icon className="h-4 w-4 shrink-0" />
                    {!collapsed && <span className="truncate">{item.label}</span>}
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t px-3 py-3 space-y-2">
        {/* Back to frontend */}
        {!collapsed && (
          <Link
            href="/"
            className="flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
          >
            <Monitor className="h-4 w-4 shrink-0" />
            <span className="truncate">返回前台</span>
          </Link>
        )}

        {/* Theme toggle */}
        <button
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          className="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
          title={collapsed ? "切换主题" : undefined}
        >
          {theme === "dark" ? <Sun className="h-4 w-4 shrink-0" /> : <Moon className="h-4 w-4 shrink-0" />}
          {!collapsed && <span className="truncate">{theme === "dark" ? "浅色模式" : "深色模式"}</span>}
        </button>

        {/* Collapse toggle */}
        <button
          onClick={toggleCollapsed}
          className="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
        >
          {collapsed ? (
            <ChevronRight className="h-4 w-4 shrink-0" />
          ) : (
            <ChevronLeft className="h-4 w-4 shrink-0" />
          )}
          {!collapsed && <span className="truncate">收起</span>}
        </button>

        {/* User info */}
        {user && !collapsed && (
          <div className="flex items-center justify-between rounded-lg bg-muted/50 px-2.5 py-2">
            <div className="flex items-center gap-2 min-w-0">
              <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary/20 text-primary text-xs font-bold uppercase">
                {user.username.charAt(0)}
              </div>
              <div className="min-w-0">
                <p className="text-xs font-semibold truncate">{user.username}</p>
                <p className="text-[10px] text-muted-foreground capitalize">{user.role}</p>
              </div>
            </div>
            <button
              onClick={logout}
              className="text-muted-foreground hover:text-destructive transition-colors"
              title="登出"
            >
              <LogOut className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
      </div>
    </aside>
  );
}

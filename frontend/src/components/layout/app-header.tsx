"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { Zap, LogOut, User, Settings, BarChart3, Home, Trophy, Menu, X } from "lucide-react";
import { ThemeToggle } from "./theme-toggle";
import { useAuthStore } from "@/stores/auth-store";

export function AppHeader() {
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, user, logout } = useAuthStore();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleLogout = () => {
    logout();
    router.push("/login");
  };

  const mobileNavItems = [
    { href: "/", label: "首页", icon: Home },
    { href: "/ideas", label: "排行榜", icon: Trophy },
    { href: "/monitor", label: "监控", icon: BarChart3, auth: true },
    { href: "/admin", label: "管理", icon: Settings, auth: true },
  ];

  return (
    <>
      <header className="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur-sm">
        <div className="container mx-auto flex h-14 md:h-16 items-center justify-between px-4">
          <div className="flex items-center gap-6">
            <Link href="/" className="flex items-center gap-2 font-display text-xl font-bold tracking-tight">
              <Zap className="h-6 w-6 text-primary" />
              <span>Idea Arena</span>
            </Link>
            <nav className="hidden md:flex items-center gap-4">
              <Link
                href="/"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                首页
              </Link>
              <Link
                href="/ideas"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                排行榜
              </Link>
              {isAuthenticated && (
                <>
                  <Link
                    href="/monitor"
                    className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1"
                  >
                    <BarChart3 className="h-3.5 w-3.5" />
                    监控
                  </Link>
                  <Link
                    href="/admin"
                    className="text-sm text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-1"
                  >
                    <Settings className="h-3.5 w-3.5" />
                    管理
                  </Link>
                </>
              )}
            </nav>
          </div>

          <div className="flex items-center gap-2">
            <ThemeToggle />
            {isAuthenticated ? (
              <div className="flex items-center gap-2">
                <span className="hidden md:inline text-sm text-muted-foreground">
                  <User className="inline h-4 w-4 mr-1" />
                  {user?.username}
                </span>
                <button
                  onClick={handleLogout}
                  className="inline-flex items-center gap-1 rounded-md px-3 py-1.5 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
                >
                  <LogOut className="h-4 w-4" />
                  <span className="hidden md:inline">登出</span>
                </button>
              </div>
            ) : (
              <Link
                href="/login"
                className="inline-flex items-center rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                登录
              </Link>
            )}
          </div>
        </div>
      </header>

      {/* Mobile bottom navigation */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 z-50 border-t bg-background/95 backdrop-blur-sm safe-area-bottom">
        <div className="flex items-center justify-around h-14 px-2">
          {mobileNavItems
            .filter((item) => !item.auth || isAuthenticated)
            .map((item) => {
              const Icon = item.icon;
              const isActive = pathname === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex flex-col items-center justify-center gap-0.5 px-3 py-1 rounded-lg transition-colors ${
                    isActive
                      ? "text-primary"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  <span className="text-[10px] font-medium">{item.label}</span>
                </Link>
              );
            })}
        </div>
      </nav>
    </>
  );
}

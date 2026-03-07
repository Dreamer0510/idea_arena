"use client";

import { useState, useEffect, useCallback } from "react";
import { Plus, Trash2, Shield, User, Loader2 } from "lucide-react";
import apiClient from "@/lib/api-client";
import { PageHeader, Card, StatusBadge, Loading, EmptyState, PrimaryButton, Field } from "@/components/admin/shared";

interface UserItem {
  id: number;
  username: string;
  role: string;
  created_at: string;
}

export default function UsersPage() {
  const [users, setUsers] = useState<UserItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newUser, setNewUser] = useState({ username: "", password: "", role: "user" });
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiClient.get("/api/v1/admin/users");
      setUsers(res.data.items || res.data || []);
    } catch {
      setUsers([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const createUser = async () => {
    if (!newUser.username || !newUser.password) return;
    setCreating(true);
    try {
      await apiClient.post("/api/v1/admin/users", newUser);
      setNewUser({ username: "", password: "", role: "user" });
      setShowCreate(false);
      load();
    } catch (e: unknown) {
      alert("创建失败: " + (e instanceof Error ? e.message : "未知错误"));
    } finally {
      setCreating(false);
    }
  };

  const updateRole = async (id: number, role: string) => {
    try {
      await apiClient.put(`/api/v1/admin/users/${id}`, { role });
      load();
    } catch {
      alert("更新失败");
    }
  };

  const deleteUser = async (id: number) => {
    if (!confirm("确定删除此用户？")) return;
    try {
      await apiClient.delete(`/api/v1/admin/users/${id}`);
      load();
    } catch {
      alert("删除失败");
    }
  };

  const roleColors: Record<string, string> = {
    admin: "bg-rose-500/10 text-rose-600",
    editor: "bg-blue-500/10 text-blue-600",
    viewer: "bg-slate-500/10 text-slate-500",
    user: "bg-emerald-500/10 text-emerald-600",
  };

  if (loading) return <Loading text="加载用户列表..." />;

  return (
    <div className="space-y-6 max-w-4xl">
      <PageHeader
        title="用户管理"
        description="管理系统用户和角色权限"
        actions={
          <PrimaryButton onClick={() => setShowCreate(!showCreate)}>
            <Plus className="h-4 w-4" /> 创建用户
          </PrimaryButton>
        }
      />

      {/* Create user form */}
      {showCreate && (
        <Card title="创建新用户">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <Field label="用户名" value={newUser.username} onChange={(v) => setNewUser({ ...newUser, username: v })} placeholder="username" />
            <Field label="密码" value={newUser.password} onChange={(v) => setNewUser({ ...newUser, password: v })} type="password" placeholder="password" />
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">角色</label>
              <select
                value={newUser.role}
                onChange={(e) => setNewUser({ ...newUser, role: e.target.value })}
                className="w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
              >
                <option value="user">用户 (user)</option>
                <option value="editor">编辑 (editor)</option>
                <option value="admin">管理员 (admin)</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2 pt-2">
            <PrimaryButton onClick={createUser} disabled={creating || !newUser.username || !newUser.password} loading={creating}>
              创建
            </PrimaryButton>
            <button onClick={() => setShowCreate(false)} className="text-sm text-muted-foreground hover:text-foreground">取消</button>
          </div>
        </Card>
      )}

      {/* User list */}
      {users.length === 0 ? (
        <EmptyState
          icon={<User className="h-10 w-10" />}
          title="暂无用户"
          description="后端 API 可能尚未实现用户列表功能"
        />
      ) : (
        <div className="rounded-xl border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">ID</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">用户名</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">角色</th>
                <th className="text-left px-4 py-3 font-medium text-muted-foreground">创建时间</th>
                <th className="text-right px-4 py-3 font-medium text-muted-foreground">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {users.map((u) => (
                <tr key={u.id} className="hover:bg-muted/30 transition-colors">
                  <td className="px-4 py-3 text-muted-foreground font-mono text-xs">#{u.id}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/20 text-primary text-xs font-bold uppercase">
                        {u.username.charAt(0)}
                      </div>
                      <span className="font-medium">{u.username}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <select
                      value={u.role}
                      onChange={(e) => updateRole(u.id, e.target.value)}
                      className={`rounded-md px-2 py-0.5 text-xs font-medium border-0 cursor-pointer ${roleColors[u.role] || "bg-muted text-muted-foreground"}`}
                    >
                      <option value="user">user</option>
                      <option value="editor">editor</option>
                      <option value="admin">admin</option>
                    </select>
                  </td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">
                    {u.created_at ? new Date(u.created_at).toLocaleDateString("zh-CN") : "—"}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => deleteUser(u.id)}
                      disabled={u.username === "admin"}
                      className="rounded-md p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                      title={u.username === "admin" ? "无法删除管理员" : "删除"}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

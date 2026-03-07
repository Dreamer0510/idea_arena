"use client";

import { useState } from "react";
import { KeyRound, Save, Eye, EyeOff, UserCog } from "lucide-react";
import apiClient from "@/lib/api-client";
import { useAuthStore } from "@/stores/auth-store";
import { useToast } from "@/components/ui/toast";
import {
  PageHeader, Card, PrimaryButton,
} from "@/components/admin/shared";

export default function AccountPage() {
  const { user } = useAuthStore();
  const { toast } = useToast();

  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [saving, setSaving] = useState(false);
  const [showOld, setShowOld] = useState(false);
  const [showNew, setShowNew] = useState(false);

  const handleChangePassword = async () => {
    if (!oldPassword || !newPassword) {
      toast("error", "请填写旧密码和新密码");
      return;
    }
    if (newPassword.length < 6) {
      toast("error", "新密码至少 6 个字符");
      return;
    }
    if (newPassword !== confirmPassword) {
      toast("error", "两次输入的新密码不一致");
      return;
    }

    setSaving(true);
    try {
      await apiClient.put("/api/v1/auth/password", {
        old_password: oldPassword,
        new_password: newPassword,
      });
      toast("success", "密码修改成功", "下次登录请使用新密码");
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error || "修改失败";
      toast("error", "密码修改失败", msg);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6 max-w-2xl">
      <PageHeader
        title="账号管理"
        description="管理个人账号信息与安全设置"
      />

      {/* User Info */}
      <Card title="账号信息">
        <div className="flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10 border-2 border-primary/20">
            <UserCog className="h-8 w-8 text-primary" />
          </div>
          <div>
            <p className="text-lg font-semibold">{user?.username || "—"}</p>
            <div className="flex items-center gap-2 mt-1">
              <span className={`inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium ${
                user?.role === "admin"
                  ? "bg-amber-500/10 text-amber-600 border border-amber-500/20"
                  : "bg-blue-500/10 text-blue-600 border border-blue-500/20"
              }`}>
                {user?.role === "admin" ? "管理员" : "普通用户"}
              </span>
            </div>
          </div>
        </div>
      </Card>

      {/* Change Password */}
      <Card title="修改密码" desc="修改后下次登录需使用新密码">
        <div className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">当前密码</label>
            <div className="flex gap-2">
              <input
                type={showOld ? "text" : "password"}
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder="输入当前密码"
                className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
              />
              <button onClick={() => setShowOld(!showOld)}
                className="rounded-lg border px-3 py-2 text-muted-foreground hover:text-foreground transition-colors">
                {showOld ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">新密码</label>
            <div className="flex gap-2">
              <input
                type={showNew ? "text" : "password"}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="至少 6 个字符"
                className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
              />
              <button onClick={() => setShowNew(!showNew)}
                className="rounded-lg border px-3 py-2 text-muted-foreground hover:text-foreground transition-colors">
                {showNew ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>

          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">确认新密码</label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="再次输入新密码"
              className="w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
            {confirmPassword && newPassword !== confirmPassword && (
              <p className="text-xs text-rose-500">两次输入的密码不一致</p>
            )}
          </div>
        </div>

        <div className="flex justify-end pt-2">
          <PrimaryButton
            onClick={handleChangePassword}
            disabled={saving || !oldPassword || !newPassword || newPassword !== confirmPassword}
            loading={saving}
          >
            <KeyRound className="h-4 w-4" /> 修改密码
          </PrimaryButton>
        </div>
      </Card>
    </div>
  );
}

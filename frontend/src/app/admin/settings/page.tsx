"use client";

import { useState, useEffect } from "react";
import { Save, Loader2 } from "lucide-react";
import apiClient from "@/lib/api-client";
import { PageHeader, Card, Loading, Field, Toggle, PrimaryButton } from "@/components/admin/shared";
import type { SystemSettings } from "@/types/api";

export default function SettingsPage() {
  const [settings, setSettings] = useState<SystemSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");

  useEffect(() => {
    apiClient.get("/api/v1/admin/settings").then((r) => setSettings(r.data));
  }, []);

  const save = async () => {
    if (!settings) return;
    setSaving(true);
    setMsg("");
    try {
      await apiClient.put("/api/v1/admin/settings", settings);
      setMsg("✅ 保存成功");
    } catch {
      setMsg("❌ 保存失败");
    } finally {
      setSaving(false);
    }
  };

  if (!settings) return <Loading text="加载设置..." />;

  return (
    <div className="space-y-6 max-w-4xl">
      <PageHeader title="系统设置" description="配置辩论引擎和话题发现的全局参数" />

      <Card title="辩论引擎设置" desc="控制辩论过程的核心参数">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Field label="最大辩论轮次" value={settings.debate_max_rounds}
            onChange={(v) => setSettings({ ...settings, debate_max_rounds: Number(v) })} type="number" />
          <Field label="毕业分数线" value={settings.debate_graduation_score}
            onChange={(v) => setSettings({ ...settings, debate_graduation_score: Number(v) })} type="number" step="0.1" />
          <Field label="辩论超时" value={settings.debate_timeout}
            onChange={(v) => setSettings({ ...settings, debate_timeout: v })} placeholder="如 1800s" />
        </div>
      </Card>

      <Card title="话题发现设置" desc="控制自动话题发现的频率和行为">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Field label="发现间隔" value={settings.discovery_interval}
            onChange={(v) => setSettings({ ...settings, discovery_interval: v })} placeholder="如 3h" />
          <Field label="每渠道话题数" value={settings.topics_per_source}
            onChange={(v) => setSettings({ ...settings, topics_per_source: Number(v) })} type="number" />
          <div />
        </div>
        <div className="flex gap-6 mt-4">
          <Toggle label="启用自动发现" checked={settings.discovery_enabled}
            onChange={(v) => setSettings({ ...settings, discovery_enabled: v })} />
          <Toggle label="自动提交辩论" checked={settings.auto_submit_debate}
            onChange={(v) => setSettings({ ...settings, auto_submit_debate: v })} />
        </div>
      </Card>

      <div className="flex items-center gap-3">
        <PrimaryButton onClick={save} disabled={saving} loading={saving}>
          <Save className="h-4 w-4" /> 保存设置
        </PrimaryButton>
        {msg && <span className="text-sm text-muted-foreground">{msg}</span>}
      </div>
    </div>
  );
}

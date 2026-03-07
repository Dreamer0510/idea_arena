"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Plus, Save, Trash2, TestTube, RefreshCw, Eye, EyeOff,
  CheckCircle, XCircle, Loader2, Server, Star, StarOff, Settings,
} from "lucide-react";
import apiClient from "@/lib/api-client";
import {
  PageHeader, Card, Loading, Field, PrimaryButton, SecondaryButton, Toggle,
} from "@/components/admin/shared";
import { useToast } from "@/components/ui/toast";

interface Provider {
  id: number;
  name: string;
  base_url: string;
  api_key: string;
  models_url: string;
  is_default: boolean;
  enabled: boolean;
}

interface ProviderModel {
  model_id: string;
  owned_by: string;
}

interface GlobalDefaults {
  max_tokens: number;
  timeout_sec: number;
  max_retries: number;
}

export default function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [showNewForm, setShowNewForm] = useState(false);
  const [globals, setGlobals] = useState<GlobalDefaults>({ max_tokens: 4096, timeout_sec: 300, max_retries: 4 });
  const [savingGlobals, setSavingGlobals] = useState(false);
  const { toast } = useToast();

  // New provider form
  const [newProvider, setNewProvider] = useState({
    name: "", base_url: "", api_key: "", models_url: "", is_default: false,
  });

  const loadProviders = useCallback(async () => {
    try {
      const res = await apiClient.get("/api/v1/admin/providers");
      setProviders(res.data.items || []);
    } catch {
      setProviders([]);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadGlobals = useCallback(async () => {
    try {
      const res = await apiClient.get("/api/v1/admin/models/config");
      setGlobals({
        max_tokens: res.data.max_tokens || 4096,
        timeout_sec: res.data.timeout_sec || 300,
        max_retries: res.data.max_retries || 4,
      });
    } catch { /* ignore */ }
  }, []);

  const saveGlobals = async () => {
    setSavingGlobals(true);
    try {
      await apiClient.put("/api/v1/admin/models/config", globals);
      toast("success", "全局设置已保存");
    } catch {
      toast("error", "保存全局设置失败");
    } finally {
      setSavingGlobals(false);
    }
  };

  useEffect(() => { loadProviders(); loadGlobals(); }, [loadProviders, loadGlobals]);

  const createProvider = async () => {
    if (!newProvider.name || !newProvider.base_url || !newProvider.api_key) {
      toast("error", "请填写名称、API地址和Key");
      return;
    }
    try {
      await apiClient.post("/api/v1/admin/providers", newProvider);
      toast("success", "服务商创建成功");
      setShowNewForm(false);
      setNewProvider({ name: "", base_url: "", api_key: "", models_url: "", is_default: false });
      loadProviders();
    } catch {
      toast("error", "创建失败");
    }
  };

  const deleteProvider = async (id: number) => {
    if (!confirm("确定删除该服务商？")) return;
    try {
      await apiClient.delete(`/api/v1/admin/providers/${id}`);
      toast("success", "已删除");
      loadProviders();
    } catch {
      toast("error", "删除失败");
    }
  };

  if (loading) return <Loading text="加载服务商..." />;

  return (
    <div className="space-y-6 max-w-4xl">
      <PageHeader
        title="服务商管理"
        description="管理 LLM API 服务商，支持多服务商并行使用"
        actions={
          <PrimaryButton onClick={() => setShowNewForm(!showNewForm)}>
            <Plus className="h-4 w-4" /> 添加服务商
          </PrimaryButton>
        }
      />

      {/* Global Defaults */}
      <Card title="全局默认设置" desc="所有服务商通用的调用参数">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <Field label="最大 Tokens" value={globals.max_tokens} type="number"
            onChange={(v) => setGlobals({ ...globals, max_tokens: Number(v) })} />
          <Field label="超时时间 (秒)" value={globals.timeout_sec} type="number"
            onChange={(v) => setGlobals({ ...globals, timeout_sec: Number(v) })} />
          <Field label="最大重试次数" value={globals.max_retries} type="number"
            onChange={(v) => setGlobals({ ...globals, max_retries: Number(v) })} />
        </div>
        <div className="flex justify-end">
          <PrimaryButton onClick={saveGlobals} disabled={savingGlobals} loading={savingGlobals}>
            <Save className="h-4 w-4" /> 保存
          </PrimaryButton>
        </div>
      </Card>

      {/* New Provider Form */}
      {showNewForm && (
        <NewProviderForm
          provider={newProvider}
          onChange={setNewProvider}
          onCreate={createProvider}
          onCancel={() => setShowNewForm(false)}
        />
      )}

      {/* Provider List */}
      {providers.length === 0 && !showNewForm ? (
        <Card>
          <div className="flex flex-col items-center py-12 text-center space-y-3">
            <Server className="h-12 w-12 text-muted-foreground/30" />
            <p className="text-sm font-medium">尚未添加服务商</p>
            <p className="text-xs text-muted-foreground">添加一个 LLM API 服务商来开始使用</p>
          </div>
        </Card>
      ) : (
        <div className="space-y-4">
          {providers.map((p) => (
            <ProviderCard
              key={p.id}
              provider={p}
              isEditing={editingId === p.id}
              onEdit={() => setEditingId(editingId === p.id ? null : p.id)}
              onDelete={() => deleteProvider(p.id)}
              onUpdate={() => { loadProviders(); toast("success", "更新成功"); }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ========== New Provider Form ==========

function NewProviderForm({
  provider,
  onChange,
  onCreate,
  onCancel,
}: {
  provider: { name: string; base_url: string; api_key: string; models_url: string; is_default: boolean };
  onChange: (v: typeof provider) => void;
  onCreate: () => void;
  onCancel: () => void;
}) {
  const [testing, setTesting] = useState(false);
  const [showKey, setShowKey] = useState(false);
  const { toast } = useToast();

  const testConnection = async () => {
    setTesting(true);
    try {
      const res = await apiClient.post("/api/v1/admin/providers/test", {
        base_url: provider.base_url,
        api_key: provider.api_key,
      });
      if (res.data.success && res.data.working_url && res.data.working_url !== provider.base_url) {
        onChange({ ...provider, base_url: res.data.working_url });
      }
      if (res.data.success) {
        toast("success", "连通成功", res.data.response);
      } else {
        toast("error", "连通失败", res.data.error);
      }
    } catch (e: unknown) {
      toast("error", "测试失败", e instanceof Error ? e.message : "网络错误");
    } finally {
      setTesting(false);
    }
  };

  return (
    <Card title="添加新服务商" desc="支持所有 OpenAI 兼容 API">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <Field label="服务商名称" value={provider.name}
          onChange={(v) => onChange({ ...provider, name: v })}
          placeholder="如：云雾 AI / OpenAI / DeepSeek" />
        <div className="sm:col-span-2">
          <Field label="API Base URL" value={provider.base_url}
            onChange={(v) => onChange({ ...provider, base_url: v })}
            placeholder="https://yunwu.ai 或 https://yunwu.ai/v1" />
        </div>
        <div className="sm:col-span-2 space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground">API Key</label>
          <div className="flex gap-2">
            <input
              type={showKey ? "text" : "password"}
              value={provider.api_key}
              onChange={(e) => onChange({ ...provider, api_key: e.target.value })}
              placeholder="sk-..."
              className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
            <button onClick={() => setShowKey(!showKey)}
              className="rounded-lg border px-3 py-2 text-muted-foreground hover:text-foreground transition-colors">
              {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
        </div>
        <div className="sm:col-span-2">
          <Field label="模型查询接口 (可选)" value={provider.models_url}
            onChange={(v) => onChange({ ...provider, models_url: v })}
            placeholder="留空则自动探测，如 https://api.example.com/v1/models" />
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Toggle label="设为默认服务商" checked={provider.is_default}
          onChange={(v) => onChange({ ...provider, is_default: v })} />
      </div>
      <div className="flex items-center gap-3 pt-2">
        <button onClick={testConnection} disabled={testing || !provider.base_url || !provider.api_key}
          className="inline-flex items-center gap-1.5 rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-1.5 text-xs font-medium text-emerald-600 hover:bg-emerald-500/10 disabled:opacity-50 transition-colors">
          {testing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <TestTube className="h-3.5 w-3.5" />} 测试连通性
        </button>
        <PrimaryButton onClick={onCreate}>
          <Plus className="h-4 w-4" /> 创建
        </PrimaryButton>
        <SecondaryButton onClick={onCancel}>取消</SecondaryButton>
      </div>
    </Card>
  );
}

// ========== Provider Card ==========

function ProviderCard({
  provider,
  isEditing,
  onEdit,
  onDelete,
  onUpdate,
}: {
  provider: Provider;
  isEditing: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onUpdate: () => void;
}) {
  const [form, setForm] = useState(provider);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [fetchingModels, setFetchingModels] = useState(false);
  const [models, setModels] = useState<ProviderModel[]>([]);
  const [showModels, setShowModels] = useState(false);
  const [showKey, setShowKey] = useState(false);
  const { toast } = useToast();

  useEffect(() => { setForm(provider); }, [provider]);

  const save = async () => {
    setSaving(true);
    try {
      await apiClient.put(`/api/v1/admin/providers/${provider.id}`, form);
      onUpdate();
    } catch { toast("error", "保存失败"); }
    finally { setSaving(false); }
  };

  const testConnection = async () => {
    setTesting(true);
    try {
      const res = await apiClient.post(`/api/v1/admin/providers/${provider.id}/test`);
      if (res.data.success && res.data.working_url) {
        setForm((prev) => ({ ...prev, base_url: res.data.working_url }));
      }
      if (res.data.success) {
        toast("success", `${provider.name} 连通成功`, res.data.response);
        onUpdate();
      } else {
        toast("error", `${provider.name} 连通失败`, res.data.error);
      }
    } catch (e: unknown) {
      toast("error", "测试失败", e instanceof Error ? e.message : "网络错误");
    } finally {
      setTesting(false);
    }
  };

  const fetchModels = async () => {
    setFetchingModels(true);
    try {
      const res = await apiClient.post(`/api/v1/admin/providers/${provider.id}/models`);
      if (res.data.success) {
        setModels(res.data.items || []);
        setShowModels(true);
        toast("success", `同步完成`, `共 ${res.data.chat_models} 个聊天模型（总计 ${res.data.total}）`);
      } else {
        toast("error", "同步失败", res.data.error);
      }
    } catch (e: unknown) {
      toast("error", "同步失败", e instanceof Error ? e.message : "网络错误");
    } finally {
      setFetchingModels(false);
    }
  };

  return (
    <Card>
      {/* Header */}
      <div className="flex items-center gap-4">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg border ${
          provider.enabled ? "bg-emerald-500/10 text-emerald-600 border-emerald-500/20" : "bg-muted text-muted-foreground"
        }`}>
          <Server className="h-5 w-5" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className="font-semibold">{provider.name}</p>
            {provider.is_default && (
              <span className="inline-flex items-center gap-1 rounded-md bg-amber-500/10 px-2 py-0.5 text-[10px] font-medium text-amber-600">
                <Star className="h-3 w-3" /> 默认
              </span>
            )}
            {!provider.enabled && (
              <span className="inline-flex items-center rounded-md bg-rose-500/10 px-2 py-0.5 text-[10px] font-medium text-rose-500">
                已禁用
              </span>
            )}
          </div>
          <p className="text-xs text-muted-foreground truncate">{provider.base_url}</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={testConnection} disabled={testing}
            className="inline-flex items-center gap-1.5 rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-1.5 text-xs font-medium text-emerald-600 hover:bg-emerald-500/10 disabled:opacity-50 transition-colors">
            {testing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <TestTube className="h-3.5 w-3.5" />} 测试
          </button>
          <button onClick={fetchModels} disabled={fetchingModels}
            className="inline-flex items-center gap-1.5 rounded-lg border border-blue-500/30 bg-blue-500/5 px-3 py-1.5 text-xs font-medium text-blue-600 hover:bg-blue-500/10 disabled:opacity-50 transition-colors">
            {fetchingModels ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />} 同步模型
          </button>
          <button onClick={onEdit}
            className="inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
            {isEditing ? "收起" : "编辑"}
          </button>
        </div>
      </div>

      {/* Edit form */}
      {isEditing && (
        <div className="border-t pt-4 space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Field label="名称" value={form.name} onChange={(v) => setForm({ ...form, name: v })} />
            <div className="sm:col-span-2">
              <Field label="API Base URL" value={form.base_url} onChange={(v) => setForm({ ...form, base_url: v })} />
            </div>
            <div className="sm:col-span-2 space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">API Key</label>
              <div className="flex gap-2">
                <input
                  type={showKey ? "text" : "password"}
                  value={form.api_key}
                  onChange={(e) => setForm({ ...form, api_key: e.target.value })}
                  placeholder="不修改留空"
                  className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/50"
                />
                <button onClick={() => setShowKey(!showKey)}
                  className="rounded-lg border px-3 py-2 text-muted-foreground hover:text-foreground transition-colors">
                  {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>
            <div className="sm:col-span-2">
              <Field label="模型查询接口 (可选)" value={form.models_url || ""}
                onChange={(v) => setForm({ ...form, models_url: v })}
                placeholder="留空则自动探测，如 https://api.example.com/v1/models" />
            </div>
          </div>
          <div className="flex items-center gap-4">
            <Toggle label="启用" checked={form.enabled} onChange={(v) => setForm({ ...form, enabled: v })} />
            <Toggle label="默认服务商" checked={form.is_default} onChange={(v) => setForm({ ...form, is_default: v })} />
          </div>
          <div className="flex items-center gap-3">
            <PrimaryButton onClick={save} disabled={saving} loading={saving}>
              <Save className="h-4 w-4" /> 保存
            </PrimaryButton>
            <button onClick={onDelete} className="inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-rose-500 hover:bg-rose-500/10 transition-colors">
              <Trash2 className="h-4 w-4" /> 删除
            </button>
          </div>
        </div>
      )}

      {/* Models list */}
      {showModels && (
        <div className="border-t pt-4 space-y-3">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">支持的模型 <span className="text-muted-foreground">({models.length})</span></p>
            <button onClick={() => setShowModels(false)} className="text-xs text-muted-foreground hover:text-foreground transition-colors">
              收起
            </button>
          </div>
          {models.length > 0 ? (
            <div className="max-h-60 overflow-y-auto rounded-lg border bg-muted/30 p-2">
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-1">
                {models.map((m) => (
                  <div key={m.model_id} className="rounded px-2 py-1 text-xs font-mono text-muted-foreground hover:bg-accent hover:text-foreground transition-colors truncate" title={m.model_id}>
                    {m.model_id}
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">点击「同步模型」按钮从服务商获取最新模型列表</p>
          )}
        </div>
      )}
    </Card>
  );
}

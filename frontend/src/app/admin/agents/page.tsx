"use client";

import { useState, useEffect, useCallback } from "react";
import { Save, RotateCcw, Bot, ChevronDown, ChevronUp, Server } from "lucide-react";
import apiClient from "@/lib/api-client";
import { PageHeader, Card, Loading, Field, PrimaryButton, SecondaryButton, Toggle } from "@/components/admin/shared";
import { SearchableSelect } from "@/components/ui/searchable-select";

interface AgentConfig {
  id: string;
  name: string;
  emoji: string;
  role: string;
  system_prompt: string;
  temperature: number;
  provider_id: number;
  model_name: string;
  enabled: boolean;
  default_prompt?: string;
}

interface ProviderOption { id: number; name: string; enabled: boolean; }
interface ProviderModel { model_id: string; owned_by: string; }

const DEFAULT_AGENTS: AgentConfig[] = [
  { id: "proposer", name: "创想者", emoji: "💥", role: "proposer", system_prompt: "", temperature: 0.8, provider_id: 0, model_name: "", enabled: true },
  { id: "opponent", name: "毒舌审判官", emoji: "⚖️", role: "opponent", system_prompt: "", temperature: 0.8, provider_id: 0, model_name: "", enabled: true },
  { id: "referee", name: "裁判", emoji: "🎯", role: "judge", system_prompt: "", temperature: 0.3, provider_id: 0, model_name: "", enabled: true },
  { id: "judge", name: "终极仲裁", emoji: "⚖️", role: "judge", system_prompt: "", temperature: 0.5, provider_id: 0, model_name: "", enabled: true },
  { id: "meta_extractor", name: "元数据提取", emoji: "📋", role: "utility", system_prompt: "", temperature: 0.2, provider_id: 0, model_name: "", enabled: true },
  { id: "debate_summarizer", name: "辩论摘要", emoji: "📝", role: "utility", system_prompt: "", temperature: 0.3, provider_id: 0, model_name: "", enabled: true },
];

export default function AgentsPage() {
  const [agents, setAgents] = useState<AgentConfig[]>([]);
  const [providers, setProviders] = useState<ProviderOption[]>([]);
  const [providerModels, setProviderModels] = useState<Record<number, ProviderModel[]>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [defaultPrompts, setDefaultPrompts] = useState<Record<string, string>>({});

  const loadAgents = useCallback(async () => {
    try {
      const res = await apiClient.get("/api/v1/admin/agents");
      const items = res.data.items || res.data || DEFAULT_AGENTS;
      // 提取默认提示词
      const prompts: Record<string, string> = {};
      items.forEach((a: AgentConfig & { default_prompt?: string }) => {
        if (a.default_prompt) prompts[a.id] = a.default_prompt;
      });
      setDefaultPrompts(prompts);
      setAgents(items.map((a: AgentConfig) => ({ ...a, provider_id: a.provider_id || 0, model_name: a.model_name || "" })));
    } catch {
      setAgents(DEFAULT_AGENTS);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadProviders = useCallback(async () => {
    try {
      const res = await apiClient.get("/api/v1/admin/providers");
      setProviders(res.data.items || []);
    } catch { /* ignore */ }
  }, []);

  const loadModelsForProvider = async (pid: number) => {
    if (providerModels[pid]) return;
    try {
      const res = await apiClient.get(`/api/v1/admin/providers/${pid}/models`);
      setProviderModels((prev) => ({ ...prev, [pid]: res.data.items || [] }));
    } catch { /* ignore */ }
  };

  useEffect(() => { loadAgents(); loadProviders(); }, [loadAgents, loadProviders]);

  const updateAgent = (id: string, patch: Partial<AgentConfig>) => {
    setAgents((prev) => prev.map((a) => (a.id === id ? { ...a, ...patch } : a)));
  };

  const saveAll = async () => {
    setSaving(true);
    setMsg("");
    try {
      await apiClient.put("/api/v1/admin/agents", { items: agents });
      setMsg("Agent 配置已保存");
    } catch {
      setMsg("保存失败");
    } finally {
      setSaving(false);
    }
  };

  const resetAgent = async (id: string) => {
    try {
      await apiClient.post(`/api/v1/admin/agents/${id}/reset`);
      loadAgents();
    } catch {
      // Reset to default locally
      const def = DEFAULT_AGENTS.find((a) => a.id === id);
      if (def) updateAgent(id, def);
    }
  };

  if (loading) return <Loading text="加载 Agent 配置..." />;

  const roleColors: Record<string, string> = {
    proposer: "border-blue-500/30 bg-blue-500/5",
    opponent: "border-rose-500/30 bg-rose-500/5",
    judge: "border-amber-500/30 bg-amber-500/5",
    utility: "border-slate-500/30 bg-slate-500/5",
  };

  const roleLabels: Record<string, string> = {
    proposer: "提案者",
    opponent: "审查者",
    judge: "裁判",
    utility: "工具",
  };

  return (
    <div className="space-y-6 max-w-4xl">
      <PageHeader
        title="Agent 人设管理"
        description="配置辩论中各 AI Agent 的名称、人设、服务商和模型"
        actions={
          <PrimaryButton onClick={saveAll} disabled={saving} loading={saving}>
            <Save className="h-4 w-4" /> 保存全部
          </PrimaryButton>
        }
      />

      {msg && <p className="text-sm text-muted-foreground">{msg}</p>}

      <div className="space-y-3">
        {agents.map((agent) => {
          const expanded = expandedId === agent.id;
          const providerName = providers.find((p) => p.id === agent.provider_id)?.name;
          return (
            <div
              key={agent.id}
              className={`rounded-xl border transition-colors ${roleColors[agent.role] || ""}`}
            >
              {/* Header */}
              <button
                onClick={() => {
                  setExpandedId(expanded ? null : agent.id);
                  if (!expanded && agent.provider_id > 0) loadModelsForProvider(agent.provider_id);
                }}
                className="flex w-full items-center gap-4 p-4 text-left"
              >
                <span className="text-2xl">{agent.emoji}</span>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <p className="text-sm font-semibold">{agent.name}</p>
                    <span className="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground uppercase">
                      {roleLabels[agent.role] || agent.role}
                    </span>
                    {agent.provider_id > 0 && agent.model_name && (
                      <span className="inline-flex items-center gap-1 rounded-md bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
                        <Server className="h-3 w-3" />
                        {providerName} / {agent.model_name}
                      </span>
                    )}
                    {!agent.enabled && (
                      <span className="inline-flex items-center rounded-md bg-rose-500/10 px-2 py-0.5 text-[10px] font-medium text-rose-500">
                        已禁用
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5 truncate">
                    {(agent.system_prompt || defaultPrompts[agent.id] || "").slice(0, 100) + "..."}
                  </p>
                </div>
                {expanded ? <ChevronUp className="h-4 w-4 text-muted-foreground" /> : <ChevronDown className="h-4 w-4 text-muted-foreground" />}
              </button>

              {/* Expanded content */}
              {expanded && (
                <div className="border-t px-4 pb-4 pt-4 space-y-4">
                  {/* Provider & Model */}
                  <div className="rounded-lg border bg-muted/30 p-4 space-y-3">
                    <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">服务商 & 模型</p>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                      <div className="space-y-1.5">
                        <label className="text-xs font-medium text-muted-foreground">服务商</label>
                        <select
                          value={agent.provider_id}
                          onChange={(e) => {
                            const pid = Number(e.target.value);
                            updateAgent(agent.id, { provider_id: pid, model_name: "" });
                            if (pid > 0) loadModelsForProvider(pid);
                          }}
                          className="w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                        >
                          <option value={0}>使用全局默认</option>
                          {providers.filter((p) => p.enabled).map((p) => (
                            <option key={p.id} value={p.id}>{p.name}</option>
                          ))}
                        </select>
                      </div>
                      <div className="space-y-1.5">
                        <label className="text-xs font-medium text-muted-foreground">模型</label>
                        {agent.provider_id > 0 && (providerModels[agent.provider_id]?.length ?? 0) > 0 ? (
                          <SearchableSelect
                            value={agent.model_name}
                            onChange={(v) => updateAgent(agent.id, { model_name: v })}
                            options={[
                              { value: "", label: "选择模型..." },
                              ...providerModels[agent.provider_id].map((m) => ({ value: m.model_id, label: m.model_id })),
                            ]}
                            placeholder="选择模型..."
                          />
                        ) : (
                          <input
                            value={agent.model_name}
                            onChange={(e) => updateAgent(agent.id, { model_name: e.target.value })}
                            placeholder={agent.provider_id > 0 ? "输入模型名或去服务商页面刷新列表" : "使用全局默认模型"}
                            className="w-full rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
                          />
                        )}
                      </div>
                    </div>
                    {agent.provider_id === 0 && (
                      <p className="text-xs text-muted-foreground">使用全局默认时，将回退到“模型配置”页面中设定的角色模型</p>
                    )}
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <Field label="名称" value={agent.name} onChange={(v) => updateAgent(agent.id, { name: v })} />
                    <Field label="Emoji" value={agent.emoji} onChange={(v) => updateAgent(agent.id, { emoji: v })} />
                    <div className="space-y-1.5">
                      <label className="text-xs font-medium text-muted-foreground">温度 (Temperature)</label>
                      <div className="flex items-center gap-3">
                        <input
                          type="range"
                          min="0"
                          max="2"
                          step="0.1"
                          value={agent.temperature}
                          onChange={(e) => updateAgent(agent.id, { temperature: Number(e.target.value) })}
                          className="flex-1 accent-primary"
                        />
                        <span className="text-sm font-mono w-10 text-right">{agent.temperature}</span>
                      </div>
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <div className="flex items-center justify-between">
                      <label className="text-xs font-medium text-muted-foreground">系统提示词 (System Prompt)</label>
                      {!agent.system_prompt && defaultPrompts[agent.id] && (
                        <span className="text-[10px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded">默认提示词</span>
                      )}
                    </div>
                    <textarea
                      value={agent.system_prompt || defaultPrompts[agent.id] || ""}
                      onChange={(e) => updateAgent(agent.id, { system_prompt: e.target.value })}
                      rows={12}
                      className="w-full rounded-lg border bg-background px-3 py-2 text-sm font-mono leading-relaxed focus:outline-none focus:ring-2 focus:ring-primary/50 resize-y"
                    />
                  </div>

                  <div className="flex items-center justify-between">
                    <Toggle
                      label="启用此 Agent"
                      checked={agent.enabled}
                      onChange={(v) => updateAgent(agent.id, { enabled: v })}
                    />
                    <SecondaryButton onClick={() => resetAgent(agent.id)}>
                      <RotateCcw className="h-4 w-4" /> 重置为默认
                    </SecondaryButton>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

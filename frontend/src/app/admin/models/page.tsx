"use client";

import { useState, useEffect } from "react";
import { Save, Brain, TestTube, Loader2, Eye, EyeOff, CheckCircle, XCircle } from "lucide-react";
import apiClient from "@/lib/api-client";
import { PageHeader, Card, Loading, Field, PrimaryButton, SecondaryButton } from "@/components/admin/shared";

interface ModelConfig {
  base_url: string;
  api_key: string;
  default_model: string;
  max_tokens: number;
  timeout_sec: number;
  max_retries: number;
  models: {
    proposer: string;
    opponent: string;
    judge: string;
    utility: string;
  };
}

const ROLE_INFO = [
  { key: "proposer", label: "创想者", emoji: "💥", desc: "负责提出和改进创业方案" },
  { key: "opponent", label: "审判官", emoji: "⚖️", desc: "负责审查和质疑方案" },
  { key: "judge", label: "裁判/仲裁", emoji: "🎯", desc: "评分和最终裁决" },
  { key: "utility", label: "工具", emoji: "📋", desc: "元数据提取和摘要生成" },
];

export default function ModelsPage() {
  const [config, setConfig] = useState<ModelConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ ok: boolean; msg: string } | null>(null);
  const [msg, setMsg] = useState("");
  const [showKey, setShowKey] = useState(false);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      const res = await apiClient.get("/api/v1/admin/models/config");
      setConfig(res.data);
    } catch {
      // Fallback default
      setConfig({
        base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1",
        api_key: "",
        default_model: "qwen-plus",
        max_tokens: 4096,
        timeout_sec: 180,
        max_retries: 4,
        models: { proposer: "kimi-k2.5", opponent: "glm-5", judge: "qwen3.5-plus", utility: "MiniMax-M2.5" },
      });
    }
  };

  const save = async () => {
    if (!config) return;
    setSaving(true);
    setMsg("");
    try {
      await apiClient.put("/api/v1/admin/models/config", config);
      setMsg("✅ 模型配置已保存");
    } catch {
      setMsg("❌ 保存失败");
    } finally {
      setSaving(false);
    }
  };

  const testConnection = async () => {
    if (!config) return;
    setTesting(true);
    setTestResult(null);
    try {
      const res = await apiClient.post("/api/v1/admin/models/test", {
        base_url: config.base_url,
        api_key: config.api_key,
        model: config.default_model,
      });
      setTestResult({ ok: true, msg: res.data.message || "连接成功" });
    } catch (e: unknown) {
      const message = e instanceof Error ? e.message : "连接失败";
      setTestResult({ ok: false, msg: message });
    } finally {
      setTesting(false);
    }
  };

  if (!config) return <Loading text="加载模型配置..." />;

  const maskedKey = config.api_key
    ? config.api_key.slice(0, 8) + "••••••••" + config.api_key.slice(-4)
    : "";

  return (
    <div className="space-y-6 max-w-4xl">
      <PageHeader
        title="模型配置"
        description="配置 LLM API 提供商和各角色使用的模型"
      />

      {/* Provider Config */}
      <Card title="API 提供商" desc="配置 OpenAI 兼容的 LLM API 端点">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="sm:col-span-2">
            <Field label="API Base URL" value={config.base_url}
              onChange={(v) => setConfig({ ...config, base_url: v })}
              placeholder="https://api.openai.com/v1" />
          </div>
          <div className="sm:col-span-2 space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">API Key</label>
            <div className="flex gap-2">
              <input
                type={showKey ? "text" : "password"}
                value={showKey ? config.api_key : maskedKey}
                onChange={(e) => setConfig({ ...config, api_key: e.target.value })}
                placeholder="sk-..."
                className="flex-1 rounded-lg border bg-background px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary/50"
              />
              <button
                onClick={() => setShowKey(!showKey)}
                className="rounded-lg border px-3 py-2 text-muted-foreground hover:text-foreground transition-colors"
                title={showKey ? "隐藏" : "显示"}
              >
                {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          <Field label="默认模型" value={config.default_model}
            onChange={(v) => setConfig({ ...config, default_model: v })}
            placeholder="qwen-plus" />
          <Field label="最大 Tokens" value={config.max_tokens}
            onChange={(v) => setConfig({ ...config, max_tokens: Number(v) })} type="number" />
          <Field label="超时时间 (秒)" value={config.timeout_sec}
            onChange={(v) => setConfig({ ...config, timeout_sec: Number(v) })} type="number" />
          <Field label="最大重试次数" value={config.max_retries}
            onChange={(v) => setConfig({ ...config, max_retries: Number(v) })} type="number" />
        </div>

        {/* Test connection */}
        <div className="flex items-center gap-3 pt-2">
          <SecondaryButton onClick={testConnection} disabled={testing} loading={testing}>
            <TestTube className="h-4 w-4" /> 测试连通性
          </SecondaryButton>
          {testResult && (
            <div className={`flex items-center gap-1.5 text-sm ${testResult.ok ? "text-emerald-600" : "text-rose-500"}`}>
              {testResult.ok ? <CheckCircle className="h-4 w-4" /> : <XCircle className="h-4 w-4" />}
              {testResult.msg}
            </div>
          )}
        </div>
      </Card>

      {/* Role Model Assignment */}
      <Card title="角色模型分配" desc="为辩论中的不同角色指定使用的模型">
        <div className="grid gap-3">
          {ROLE_INFO.map((role) => (
            <div key={role.key} className="flex items-center gap-4 rounded-xl border p-4">
              <span className="text-2xl">{role.emoji}</span>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold">{role.label}</p>
                <p className="text-xs text-muted-foreground">{role.desc}</p>
              </div>
              <input
                value={config.models[role.key as keyof typeof config.models]}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    models: { ...config.models, [role.key]: e.target.value },
                  })
                }
                placeholder={config.default_model}
                className="w-48 rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50"
              />
            </div>
          ))}
        </div>
        <p className="text-xs text-muted-foreground mt-2">
          留空将使用默认模型 ({config.default_model})
        </p>
      </Card>

      <div className="flex items-center gap-3">
        <PrimaryButton onClick={save} disabled={saving} loading={saving}>
          <Save className="h-4 w-4" /> 保存配置
        </PrimaryButton>
        {msg && <span className="text-sm text-muted-foreground">{msg}</span>}
      </div>
    </div>
  );
}

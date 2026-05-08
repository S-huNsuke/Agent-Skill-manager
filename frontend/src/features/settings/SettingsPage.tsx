import { useEffect, useState } from "react";
import type { DiagnosticItemViewModel, GeneralSettingsViewModel, AutomationSettingsViewModel, AISettingsViewModel, AppInfoViewModel } from "../../lib/mocks";
import { isRunningInWails, waitForApi } from "../../lib/api";
import { StatusBadge } from "../../components/StatusBadge";
import { aiProviderPresets, applyPreset, defaultAISettings, getModelOptions, getPresetFromSettings, type AiPreset } from "../ai/aiSettings";

interface SettingsPageProps {
  diagnostics: DiagnosticItemViewModel[];
  onRefresh?: () => void;
}

type SettingsTab = "general" | "ai" | "automation" | "diagnostics" | "about";

const toneCopy = { stable: "正常", attention: "待检查", critical: "异常", muted: "已忽略" } as const;

/** 诊断 Wails 绑定状态 */
function getWailsBindingDiagnostics(): { label: string; value: string; tone: string }[] {
  const results: { label: string; value: string; tone: string }[] = [];
  const w = window as unknown as Record<string, unknown>;
  const hasGo = typeof w.go === "object" && w.go !== null;
  results.push({ label: "window.go 存在", value: hasGo ? "是" : "否", tone: hasGo ? "stable" : "critical" });
  if (hasGo) {
    const go = w.go as Record<string, unknown>;
    const pkgKeys = Object.keys(go);
    results.push({ label: "go 顶层键", value: pkgKeys.length > 0 ? pkgKeys.join(", ") : "（空）", tone: pkgKeys.length > 0 ? "stable" : "critical" });
    for (const pkgKey of pkgKeys) {
      const pkg = go[pkgKey];
      if (pkg && typeof pkg === "object") {
        const structKeys = Object.keys(pkg as Record<string, unknown>);
        results.push({ label: `go["${pkgKey}"] 子键`, value: structKeys.join(", ") || "（空）", tone: "stable" });
        for (const structKey of structKeys) {
          const structObj = (pkg as Record<string, unknown>)[structKey];
          if (structObj && typeof structObj === "object") {
            const methodKeys = Object.keys(structObj as Record<string, unknown>);
            results.push({ label: `go["${pkgKey}"]["${structKey}"] 方法`, value: methodKeys.join(", ") || "（空）", tone: methodKeys.length > 0 ? "stable" : "critical" });
          }
        }
      }
    }
  }
  return results;
}

/** 系统设置与诊断页面，展示通用设置、自动化设置、诊断信息和关于页面 */
export function SettingsPage({ diagnostics, onRefresh }: SettingsPageProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>("general");
  const [bindingDiag, setBindingDiag] = useState<{ label: string; value: string; tone: string }[]>([]);
  const [generalSettings, setGeneralSettings] = useState<GeneralSettingsViewModel>({ theme: "light", fontSize: "medium", notificationsEnabled: true, language: "zh-CN" });
  const [automationSettings, setAutomationSettings] = useState<AutomationSettingsViewModel>({ autoSyncCatalog: true, autoCheckUpdates: true, autoApplySkillGroups: false, healthCheckSchedule: "daily", autoRepair: false });
  const [aiSettings, setAiSettings] = useState<AISettingsViewModel>(defaultAISettings);
  const [aiPreset, setAiPreset] = useState<AiPreset>("none");
  const [appInfo, setAppInfo] = useState<AppInfoViewModel | null>(null);
  const [saving, setSaving] = useState(false);
  const [settingsError, setSettingsError] = useState<string | null>(null);

  useEffect(() => {
    setBindingDiag(getWailsBindingDiagnostics());
    let cancelled = false;
    async function loadSettings() {
      try {
        const api = await waitForApi();
        const [generalRes, automationRes, aiRes, infoRes] = await Promise.allSettled([
          api.getGeneralSettings(),
          api.getAutomationSettings(),
          api.getAISettings(),
          api.getAppInfoFull(),
        ]);
        if (cancelled) return;

        const errors: string[] = [];
        if (generalRes.status === "fulfilled") {
          setGeneralSettings(generalRes.value);
        } else {
          errors.push("通用设置");
        }
        if (automationRes.status === "fulfilled") {
          setAutomationSettings(automationRes.value);
        } else {
          errors.push("自动化设置");
        }
        if (aiRes.status === "fulfilled") {
          setAiSettings(aiRes.value);
          setAiPreset(getPresetFromSettings(aiRes.value));
        } else {
          setAiSettings(defaultAISettings);
          setAiPreset("none");
          errors.push("AI 设置");
        }
        if (infoRes.status === "fulfilled") {
          setAppInfo(infoRes.value);
        } else {
          errors.push("应用信息");
        }
        setSettingsError(errors.length > 0 ? `部分设置加载失败：${errors.join("、")}` : null);
      } catch (err) {
        if (!cancelled) {
          setSettingsError(err instanceof Error ? err.message : String(err));
        }
      }
    }
    void loadSettings();
    return () => {
      cancelled = true;
    };
  }, []);

  const inWails = isRunningInWails();

  /** 保存通用设置 */
  const handleSaveGeneral = async () => {
    if (!generalSettings) return;
    setSaving(true);
    try {
      const api = await waitForApi();
      await api.saveGeneralSettings(generalSettings);
    } catch {
      // 静默处理
    } finally {
      setSaving(false);
    }
  };

  /** 保存自动化设置 */
  const handleSaveAutomation = async () => {
    if (!automationSettings) return;
    setSaving(true);
    try {
      const api = await waitForApi();
      await api.saveAutomationSettings(automationSettings);
    } catch {
      // 静默处理
    } finally {
      setSaving(false);
    }
  };

  /** 保存 AI 设置 */
  const handleSaveAI = async () => {
    setSaving(true);
    try {
      const api = await waitForApi();
      await api.saveAISettings(aiSettings);
    } catch {
      // 静默处理
    } finally {
      setSaving(false);
    }
  };

  const tabs: { key: SettingsTab; label: string }[] = [
    { key: "general", label: "通用" },
    { key: "ai", label: "AI 配置" },
    { key: "automation", label: "自动化" },
    { key: "diagnostics", label: "诊断" },
    { key: "about", label: "关于" },
  ];

  return (
    <section className="animate-page-in">
      {/* Header */}
      <div className="bg-surface rounded-panel shadow-panel p-8 mb-8">
        <p className="uppercase tracking-widest text-xs text-ink-muted font-body">设置</p>
        <h1 className="font-display text-3xl font-semibold text-ink tracking-tight">系统设置</h1>
        <p className="text-lg text-ink-soft leading-relaxed">管理应用设置、自动化配置和系统诊断。</p>
      </div>

      {/* Tab Navigation */}
      <div className="flex gap-2 mb-6">
        {tabs.map((tab) => (
          <button key={tab.key} type="button" onClick={() => setActiveTab(tab.key)} className={`rounded-chip px-4 py-2 text-sm font-medium transition-colors ${activeTab === tab.key ? "bg-accent-glow text-accent" : "text-ink-soft hover:bg-surface-hover"}`}>
            {tab.label}
          </button>
        ))}
      </div>

      {/* General Settings */}
      {activeTab === "general" && (
        <article className="bg-surface rounded-panel shadow-panel p-6">
          <h2 className="font-display text-xl font-semibold text-ink mb-4">通用设置</h2>
          <div className="flex flex-col gap-4">
            <div>
              <label className="block text-sm font-medium text-ink mb-1">主题</label>
              <select value={generalSettings.theme} onChange={(e) => setGeneralSettings({ ...generalSettings, theme: e.target.value })} className="bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent">
                <option value="light">浅色</option>
                <option value="dark">深色</option>
                <option value="system">跟随系统</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-ink mb-1">字体大小</label>
              <select value={generalSettings.fontSize} onChange={(e) => setGeneralSettings({ ...generalSettings, fontSize: e.target.value })} className="bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent">
                <option value="small">小</option>
                <option value="medium">中</option>
                <option value="large">大</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-ink mb-1">语言</label>
              <select value={generalSettings.language} onChange={(e) => setGeneralSettings({ ...generalSettings, language: e.target.value })} className="bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent">
                <option value="zh-CN">简体中文</option>
                <option value="en">English</option>
              </select>
            </div>
            <div className="flex items-center justify-between p-3 rounded-card bg-surface-warm">
              <span className="text-sm text-ink">启用通知</span>
              <button type="button" onClick={() => setGeneralSettings({ ...generalSettings, notificationsEnabled: !generalSettings.notificationsEnabled })} className={`w-10 h-6 rounded-full transition-colors ${generalSettings.notificationsEnabled ? "bg-accent" : "bg-border-soft"}`}>
                <span className={`block w-4 h-4 rounded-full bg-white transition-transform ${generalSettings.notificationsEnabled ? "translate-x-5" : "translate-x-1"}`} />
              </button>
            </div>
            <div className="flex justify-end">
              <button type="button" onClick={handleSaveGeneral} disabled={saving} className="rounded-pill px-5 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40">
                {saving ? "保存中..." : "保存设置"}
              </button>
            </div>
          </div>
        </article>
      )}

      {/* AI Settings */}
      {activeTab === "ai" && (
        <article className="bg-surface rounded-panel shadow-panel p-6">
          <div className="flex items-center justify-between gap-4 mb-4">
            <div>
              <h2 className="font-display text-xl font-semibold text-ink">AI 配置</h2>
              <p className="text-sm text-ink-soft">配置 AI 助手使用的 LLM 供应商、API Key 和模型。</p>
            </div>
            <StatusBadge tone={inWails ? "stable" : "attention"} label={inWails ? "Wails 已连接" : "示例模式"} />
          </div>
          {settingsError && (
            <div className="mb-4 rounded-card border border-attention-ink/20 bg-attention-bg/40 px-4 py-3 text-sm text-ink">
              {settingsError}
            </div>
          )}
          <div className="flex flex-col gap-4">
            <div>
              <label htmlFor="ai-provider-preset" className="block text-sm font-medium text-ink mb-1">供应商预设</label>
              <select
                id="ai-provider-preset"
                value={aiPreset}
                onChange={(e) => {
                  const nextPreset = e.target.value as AiPreset;
                  setAiPreset(nextPreset);
                  setAiSettings((current) => applyPreset(nextPreset, current));
                }}
                className="bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent w-full"
              >
                {aiProviderPresets.map((preset) => (
                  <option key={preset.value} value={preset.value}>{preset.label}</option>
                ))}
              </select>
              <p className="text-xs text-ink-muted mt-1">常见兼容网关会自动填入可用的 Base URL 和模型名。</p>
            </div>
            <div>
              <label htmlFor="ai-api-key" className="block text-sm font-medium text-ink mb-1">API Key</label>
              <input
                id="ai-api-key"
                type="password"
                value={aiSettings.apiKey}
                onChange={(e) => setAiSettings({ ...aiSettings, apiKey: e.target.value })}
                placeholder={
                  aiSettings.provider === "openai"
                    ? "sk-..."
                    : aiSettings.provider === "anthropic"
                      ? "sk-ant-..."
                      : aiSettings.provider === "gemini"
                        ? "AIza..."
                        : aiSettings.provider === "openai-compatible"
                          ? "兼容网关 API Key"
                          : "选择供应商后输入 API Key"
                }
                className="bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent w-full"
                disabled={aiSettings.provider === "none"}
              />
              <p className="text-xs text-ink-muted mt-1">API Key 仅存储在本地，不会上传到任何服务器。</p>
            </div>
            <div>
              <label htmlFor="ai-model-preset" className="block text-sm font-medium text-ink mb-1">模型预设</label>
              <select
                id="ai-model-preset"
                value={getModelOptions(aiPreset).includes(aiSettings.model) ? aiSettings.model : "__custom__"}
                onChange={(e) => {
                  if (e.target.value !== "__custom__") {
                    setAiSettings({ ...aiSettings, model: e.target.value });
                  }
                }}
                className="bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent w-full"
                disabled={aiPreset === "none"}
              >
                <option value="__custom__">自定义模型</option>
                {getModelOptions(aiPreset).map((model) => (
                  <option key={model} value={model}>{model}</option>
                ))}
              </select>
              <input
                id="ai-model-custom"
                type="text"
                value={aiSettings.model}
                onChange={(e) => setAiSettings({ ...aiSettings, model: e.target.value })}
                placeholder={
                  aiSettings.provider === "openai"
                    ? "gpt-4o"
                    : aiSettings.provider === "anthropic"
                      ? "claude-sonnet-4-20250514"
                      : aiSettings.provider === "gemini"
                        ? "gemini-2.0-flash"
                        : aiSettings.provider === "openai-compatible"
                          ? "gpt-4o-mini"
                          : "选择供应商后输入模型名称"
                }
                className="mt-2 bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent w-full"
                disabled={aiSettings.provider === "none"}
              />
            </div>
            <div>
              <label htmlFor="ai-base-url" className="block text-sm font-medium text-ink mb-1">自定义 Base URL（可选）</label>
              <input
                id="ai-base-url"
                type="text"
                value={aiSettings.baseUrl}
                onChange={(e) => setAiSettings({ ...aiSettings, baseUrl: e.target.value })}
                placeholder={aiSettings.provider === "openai-compatible" ? "https://openrouter.ai/api/v1" : "留空使用官方 API 地址"}
                className="bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent w-full"
                disabled={aiSettings.provider === "none"}
              />
              <p className="text-xs text-ink-muted mt-1">如使用代理、网关或 OpenAI-Compatible 服务，可填写自定义 API 地址。</p>
            </div>
            <div className="flex justify-end">
              <button type="button" onClick={handleSaveAI} disabled={saving} className="rounded-pill px-5 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40">
                {saving ? "保存中..." : "保存设置"}
              </button>
            </div>
          </div>
        </article>
      )}

      {/* Automation Settings */}
      {activeTab === "automation" && automationSettings && (
        <article className="bg-surface rounded-panel shadow-panel p-6">
          <h2 className="font-display text-xl font-semibold text-ink mb-4">自动化设置</h2>
          <div className="flex flex-col gap-4">
            {[
              { key: "autoSyncCatalog" as const, label: "自动同步商店目录", desc: "定期从商店源同步最新技能列表" },
              { key: "autoCheckUpdates" as const, label: "自动检查更新", desc: "定期检查已安装技能的更新" },
              { key: "autoApplySkillGroups" as const, label: "自动应用技能组", desc: "项目激活时自动应用绑定的技能组" },
              { key: "autoRepair" as const, label: "自动修复", desc: "检测到异常时自动尝试修复" },
            ].map((item) => (
              <div key={item.key} className="flex items-center justify-between p-3 rounded-card bg-surface-warm">
                <div>
                  <p className="text-sm font-medium text-ink">{item.label}</p>
                  <p className="text-xs text-ink-muted">{item.desc}</p>
                </div>
                <button type="button" onClick={() => setAutomationSettings({ ...automationSettings, [item.key]: !automationSettings[item.key] })} className={`w-10 h-6 rounded-full transition-colors ${automationSettings[item.key] ? "bg-accent" : "bg-border-soft"}`}>
                  <span className={`block w-4 h-4 rounded-full bg-white transition-transform ${automationSettings[item.key] ? "translate-x-5" : "translate-x-1"}`} />
                </button>
              </div>
            ))}
            <div>
              <label className="block text-sm font-medium text-ink mb-1">健康检查频率</label>
              <select value={automationSettings.healthCheckSchedule} onChange={(e) => setAutomationSettings({ ...automationSettings, healthCheckSchedule: e.target.value })} className="bg-surface-warm rounded-card px-4 py-2 text-sm text-ink border border-border focus:outline-none focus:ring-1 focus:ring-accent">
                <option value="never">从不</option>
                <option value="daily">每天</option>
                <option value="weekly">每周</option>
              </select>
            </div>
            <div className="flex justify-end">
              <button type="button" onClick={handleSaveAutomation} disabled={saving} className="rounded-pill px-5 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40">
                {saving ? "保存中..." : "保存设置"}
              </button>
            </div>
          </div>
        </article>
      )}

      {/* Diagnostics */}
      {activeTab === "diagnostics" && (
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-4">
            {diagnostics?.map((item) => (
              <article key={item.id} className="bg-surface rounded-card shadow-panel p-6 hover:shadow-panel-hover transition-shadow">
                <div className="flex items-center justify-between mb-2">
                  <h2 className="font-display text-lg font-semibold text-ink">{item.label}</h2>
                  <StatusBadge tone={item.tone} label={toneCopy[item.tone as keyof typeof toneCopy] || item.tone} />
                </div>
                <p className="text-ink-soft">{item.value}</p>
              </article>
            ))}
          </div>
          <div className="mt-4">
            <p className="uppercase tracking-widest text-xs text-ink-muted font-body mb-2">Wails 绑定诊断</p>
            <h2 className="font-display text-xl font-semibold text-ink mb-4">前端绑定状态</h2>
            <p className="text-sm text-ink-soft mb-4">运行环境: {inWails ? "Wails 桌面" : "浏览器（未连接后端）"}</p>
            <div className="flex flex-col gap-3">
              {bindingDiag?.map((item, idx) => (
                <article key={idx} className="bg-surface rounded-card shadow-panel p-4">
                  <div className="flex items-center justify-between mb-1">
                    <h3 className="font-medium text-ink">{item.label}</h3>
                    <StatusBadge tone={(item.tone as "stable" | "attention" | "critical" | "muted")} label={item.tone === "stable" ? "正常" : item.tone === "critical" ? "异常" : "待检查"} />
                  </div>
                  <p className="text-sm text-ink-soft break-all">{item.value}</p>
                </article>
              ))}
            </div>
          </div>
          <div className="mt-4">
            <button type="button" onClick={() => { setBindingDiag(getWailsBindingDiagnostics()); onRefresh?.(); }} className="bg-accent text-white rounded-pill px-6 py-2.5 font-medium shadow-accent hover:bg-accent-warm transition-colors">
              刷新诊断数据
            </button>
          </div>
        </div>
      )}

      {/* About */}
      {activeTab === "about" && appInfo && (
        <article className="bg-surface rounded-panel shadow-panel p-6">
          <h2 className="font-display text-xl font-semibold text-ink mb-4">关于</h2>
          <div className="flex flex-col gap-3">
            {[
              { label: "应用名称", value: appInfo.name },
              { label: "版本", value: appInfo.version },
              { label: "构建时间", value: appInfo.buildTime || "未知" },
              { label: "Go 版本", value: appInfo.goVersion || "未知" },
              { label: "操作系统", value: `${appInfo.os || "未知"} / ${appInfo.arch || "未知"}` },
            ].map((item) => (
              <div key={item.label} className="flex items-center justify-between p-3 rounded-card bg-surface-warm">
                <span className="text-sm text-ink-muted">{item.label}</span>
                <span className="text-sm font-medium text-ink">{item.value}</span>
              </div>
            ))}
          </div>
          <div className="mt-6 p-4 rounded-card bg-surface-warm text-center">
            <p className="text-sm text-ink-soft">Agent Skills Manager</p>
            <p className="text-xs text-ink-muted mt-1">管理本机 AI 代理的技能安装、更新和配置</p>
          </div>
        </article>
      )}
    </section>
  );
}

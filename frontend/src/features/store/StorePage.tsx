import { useMemo, useState, useEffect, useCallback } from "react";
import type { StoreItemViewModel, CatalogSourceViewModel, SyncResultViewModel, AgentViewModel, SkillExplanationViewModel } from "../../lib/mocks";
import { selectApi } from "../../lib/api";
import { SearchBar } from "../../components/SearchBar";
import { FilterPanel } from "../../components/FilterPanel";
import { EmptyState } from "../../components/EmptyState";
import { Modal } from "../../components/Modal";
import { StatusBadge } from "../../components/StatusBadge";

interface StorePageProps {
  items: StoreItemViewModel[];
  agents: AgentViewModel[];
  onRefresh?: () => void;
  onAction?: (action: string, agentID: string, skillName: string, cachePath?: string) => void;
}

/** 商店页面组件，展示远程技能市场，支持源管理、搜索筛选与安装操作 */
export function StorePage({ items, agents, onRefresh, onAction }: StorePageProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [compatFilter, setCompatFilter] = useState("");
  const [sources, setSources] = useState<CatalogSourceViewModel[]>([]);
  const [showSources, setShowSources] = useState(false);
  const [syncingSourceId, setSyncingSourceId] = useState<string | null>(null);
  const [syncResult, setSyncResult] = useState<SyncResultViewModel | null>(null);
  const [addSourceUrl, setAddSourceUrl] = useState("");
  const [addSourceName, setAddSourceName] = useState("");
  const [installTarget, setInstallTarget] = useState<StoreItemViewModel | null>(null);
  const [selectedAgent, setSelectedAgent] = useState("");
  const [installing, setInstalling] = useState(false);
  const [syncingAll, setSyncingAll] = useState(false);
  const [detailItem, setDetailItem] = useState<StoreItemViewModel | null>(null);
  const [explainItem, setExplainItem] = useState<StoreItemViewModel | null>(null);
  const [explainResult, setExplainResult] = useState<SkillExplanationViewModel | null>(null);
  const [explainLoading, setExplainLoading] = useState(false);

  const allSources = useMemo(() => Array.from(new Set(items.map((i) => i.source))).sort(), [items]);
  const allAgents = useMemo(() => {
    const agentSet = new Set<string>();
    items.forEach((item) => item.compatibility.forEach((agent) => agentSet.add(agent)));
    return Array.from(agentSet).sort();
  }, [items]);

  const availableAgents = useMemo(() => {
    if (!installTarget) return [];
    return agents.filter((a) =>
      installTarget.compatibility.length === 0 || installTarget.compatibility.includes(a.name)
    );
  }, [agents, installTarget]);

  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      const matchesSearch = searchQuery === "" || item.name.toLowerCase().includes(searchQuery.toLowerCase()) || item.summary.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesSource = sourceFilter === "" || item.source === sourceFilter;
      const matchesCompat = compatFilter === "" || item.compatibility.includes(compatFilter);
      return matchesSearch && matchesSource && matchesCompat;
    });
  }, [items, searchQuery, sourceFilter, compatFilter]);

  const sourceCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const item of items) {
      counts[item.source] = (counts[item.source] || 0) + 1;
    }
    return counts;
  }, [items]);

  /** 加载商店源列表 */
  const handleLoadSources = useCallback(async () => {
    try {
      const api = selectApi();
      const result = await api.getCatalogSources();
      setSources(result);
      setShowSources(true);
    } catch {
      setSources([]);
    }
  }, []);

  /** 同步单个商店源 */
  const handleSync = useCallback(async (sourceID: string) => {
    setSyncingSourceId(sourceID);
    try {
      const api = selectApi();
      const result = await api.syncCatalogSource(sourceID);
      setSyncResult(result);
      const updated = await api.getCatalogSources();
      setSources(updated);
      onRefresh?.();
    } catch {
      setSyncResult({ sourceId: sourceID, success: false, newSkills: 0, updatedSkills: 0, errors: ["同步失败"] });
    } finally {
      setSyncingSourceId(null);
    }
  }, [onRefresh]);

  /** 同步所有源 */
  const handleSyncAll = useCallback(async () => {
    setSyncingAll(true);
    try {
      const api = selectApi();
      const results = await api.syncAllSources();
      const updated = await api.getCatalogSources();
      setSources(updated);
      const totalNew = results.reduce((sum, r) => sum + r.newSkills, 0);
      const hasErrors = results.some((r) => !r.success);
      setSyncResult({
        sourceId: "all",
        success: !hasErrors,
        newSkills: totalNew,
        updatedSkills: 0,
        errors: results.flatMap((r) => r.errors),
      });
      onRefresh?.();
    } catch {
      setSyncResult({ sourceId: "all", success: false, newSkills: 0, updatedSkills: 0, errors: ["同步失败"] });
    } finally {
      setSyncingAll(false);
    }
  }, [onRefresh]);

  /** 添加自定义源 */
  const handleAddSource = useCallback(async () => {
    if (!addSourceName.trim() || !addSourceUrl.trim()) return;
    try {
      const api = selectApi();
      await api.addCatalogSource(addSourceName.trim(), addSourceUrl.trim());
      const updated = await api.getCatalogSources();
      setSources(updated);
      setAddSourceName("");
      setAddSourceUrl("");
    } catch (err) {
      alert(`添加失败: ${err instanceof Error ? err.message : String(err)}`);
    }
  }, [addSourceName, addSourceUrl]);

  /** 移除商店源 */
  const handleRemoveSource = useCallback(async (sourceID: string) => {
    try {
      const api = selectApi();
      const result = await api.removeCatalogSource(sourceID);
      if (result !== "ok") {
        alert(result);
        return;
      }
      const updated = await api.getCatalogSources();
      setSources(updated);
      onRefresh?.();
    } catch (err) {
      alert(`移除失败: ${err instanceof Error ? err.message : String(err)}`);
    }
  }, [onRefresh]);

  /** 打开安装对话框 */
  const handleInstallClick = useCallback((item: StoreItemViewModel) => {
    setInstallTarget(item);
    setSelectedAgent("");
  }, []);

  /** 查看技能详情 */
  const handleViewDetail = useCallback((item: StoreItemViewModel) => {
    setDetailItem(item);
  }, []);

  /** 问 AI 技能作用 */
  const handleAskAI = useCallback(async (item: StoreItemViewModel) => {
    setExplainItem(item);
    setExplainLoading(true);
    setExplainResult(null);
    try {
      const api = selectApi();
      const result = await api.explainStoreSkill(item.source, item.name);
      setExplainResult(result);
    } catch {
      setExplainResult({
        agentId: "store",
        agentName: item.source,
        skillName: item.name,
        found: false,
        skillPath: "",
        readmeFile: "",
        readmeContent: "",
        files: [],
      });
    } finally {
      setExplainLoading(false);
    }
  }, []);

  /** 执行安装 */
  const handleConfirmInstall = useCallback(async () => {
    if (!installTarget || !selectedAgent) return;
    setInstalling(true);
    try {
      onAction?.("install", selectedAgent, installTarget.name, installTarget.localCachePath);
      setInstallTarget(null);
      setSelectedAgent("");
      onRefresh?.();
    } finally {
      setInstalling(false);
    }
  }, [installTarget, selectedAgent, onAction, onRefresh]);

  /** 首次加载时自动获取源列表 */
  useEffect(() => {
    handleLoadSources();
  }, [handleLoadSources]);

  const filters = useMemo(() => [
    {
      key: "source",
      label: "全部来源",
      options: allSources.map((s) => ({ value: s, label: `${s} (${sourceCounts[s] || 0})` })),
      value: sourceFilter,
    },
    {
      key: "compat",
      label: "全部代理",
      options: allAgents.map((a) => ({ value: a, label: a })),
      value: compatFilter,
    },
  ], [allSources, sourceCounts, allAgents, sourceFilter, compatFilter]);

  const handleFilterChange = (key: string, value: string) => {
    if (key === "source") setSourceFilter(value);
    if (key === "compat") setCompatFilter(value);
  };

  return (
    <section className="animate-page-in">
      {/* Header */}
      <div className="bg-surface rounded-panel shadow-panel p-8 mb-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="uppercase tracking-widest text-xs text-ink-muted font-body">技能商店</p>
            <h1 className="font-display text-3xl font-semibold text-ink tracking-tight">浏览与安装技能</h1>
            <p className="text-lg text-ink-soft leading-relaxed">从远程市场浏览可用技能，选择目标代理后安装。</p>
          </div>
          <div className="flex gap-3 items-center">
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-ink">{items.length}</p>
              <p className="text-xs text-ink-muted">可用技能</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-stable-ink">{items.filter((i) => i.status === "installed").length}</p>
              <p className="text-xs text-ink-muted">已安装</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-accent">{sources.length}</p>
              <p className="text-xs text-ink-muted">市场源</p>
            </div>
          </div>
        </div>
        <div className="flex gap-3 mt-6">
          <button
            type="button"
            className="bg-accent text-white rounded-pill px-6 py-2.5 font-medium shadow-accent hover:bg-accent-warm transition-colors"
            onClick={handleSyncAll}
            disabled={syncingAll}
          >
            {syncingAll ? "同步中..." : "同步全部来源"}
          </button>
          <button
            type="button"
            className="bg-surface text-ink rounded-pill px-6 py-2.5 border border-border hover:bg-surface-hover transition-colors"
            onClick={() => setShowSources(true)}
          >
            管理来源
          </button>
        </div>
      </div>

      {/* Sync Result */}
      {syncResult && (
        <div className={`mb-6 p-4 rounded-card ${syncResult.success ? "bg-stable-bg" : "bg-critical/10"}`}>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-ink">{syncResult.success ? "同步成功" : "同步失败"}</p>
              {syncResult.success && <p className="text-xs text-ink-soft">新增 {syncResult.newSkills} 个技能</p>}
              {syncResult.errors?.length > 0 && <p className="text-xs text-critical">{syncResult.errors.join(", ")}</p>}
            </div>
            <button type="button" onClick={() => setSyncResult(null)} className="text-ink-soft hover:text-ink text-sm">✕</button>
          </div>
        </div>
      )}

      {/* Search & Filter */}
      <div className="flex flex-wrap gap-3 mb-6">
        <SearchBar value={searchQuery} onChange={setSearchQuery} placeholder="搜索技能名称或描述..." />
        <FilterPanel filters={filters} onChange={handleFilterChange} />
        {onRefresh && (
          <button onClick={onRefresh} className="bg-surface rounded-card shadow-panel px-4 py-2 text-sm text-ink hover:shadow-panel-hover transition-shadow">
            刷新
          </button>
        )}
      </div>

      {/* Skills Grid */}
      {filteredItems.length > 0 ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(320px,1fr))] gap-4 mb-8">
          {filteredItems.map((item) => (
            <article
              key={item.id}
              className="bg-surface rounded-panel shadow-panel p-5 hover:shadow-panel-hover transition-all flex flex-col gap-3"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <h3 className="font-display text-base font-semibold text-ink truncate">{item.name}</h3>
                  <p className="text-xs text-ink-muted mt-0.5">来自 {item.source}</p>
                </div>
                <StatusBadge
                  tone={item.status === "installed" ? "stable" : item.status === "failed" ? "critical" : "muted"}
                  label={item.status === "installed" ? "已安装" : item.status === "failed" ? "失败" : "可用"}
                  size="sm"
                />
              </div>

              <p className="text-sm text-ink-soft line-clamp-2 flex-1">{item.summary}</p>

              {item.compatibility.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {item.compatibility.slice(0, 4).map((agent) => (
                    <span key={agent} className="rounded-chip px-2 py-0.5 text-[11px] font-medium bg-chip text-chip-ink">{agent}</span>
                  ))}
                  {item.compatibility.length > 4 && (
                    <span className="rounded-chip px-2 py-0.5 text-[11px] font-medium bg-chip text-chip-ink">+{item.compatibility.length - 4}</span>
                  )}
                </div>
              )}

              <div className="flex items-center justify-between pt-1 border-t border-border-soft">
                <span className="text-xs text-ink-muted">{item.author}</span>
                <div className="flex gap-1.5">
                  <button
                    type="button"
                    className="rounded-chip px-2.5 py-1 text-xs font-medium bg-surface-warm text-ink hover:bg-surface-hover transition-colors"
                    onClick={() => handleViewDetail(item)}
                  >
                    详情
                  </button>
                  <button
                    type="button"
                    className="rounded-chip px-2.5 py-1 text-xs font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors"
                    onClick={() => handleAskAI(item)}
                  >
                    AI 解读
                  </button>
                  {item.status === "installed" ? (
                    <span className="text-xs font-medium text-stable-ink px-2 py-1">✓ 已安装</span>
                  ) : (
                    <button
                      type="button"
                      className="bg-accent text-white rounded-pill px-4 py-1.5 text-xs font-medium hover:bg-accent-warm transition-colors"
                      onClick={() => handleInstallClick(item)}
                    >
                      安装
                    </button>
                  )}
                </div>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <EmptyState
          title="暂无可用技能"
          description="点击「同步全部来源」从远程市场获取技能列表"
          action={{ label: "同步来源", onClick: handleSyncAll }}
        />
      )}

      {/* Install Agent Selection Modal */}
      <Modal
        open={installTarget !== null}
        onClose={() => { setInstallTarget(null); setSelectedAgent(""); }}
        title={`安装「${installTarget?.name ?? ""}」`}
        subtitle="选择目标代理"
      >
        <div className="flex flex-col gap-4">
          <p className="text-ink-soft text-sm">{installTarget?.summary}</p>

          {installTarget?.compatibility && installTarget.compatibility.length > 0 && (
            <div className="flex flex-wrap gap-1">
              <span className="text-xs text-ink-muted">兼容代理：</span>
              {installTarget.compatibility.map((agent) => (
                <span key={agent} className="rounded-chip px-2 py-0.5 text-[11px] font-medium bg-chip text-chip-ink">{agent}</span>
              ))}
            </div>
          )}

          <div>
            <p className="text-sm font-medium text-ink mb-2">选择安装至哪个代理</p>
            {availableAgents.length > 0 ? (
              <div className="flex flex-col gap-2">
                {availableAgents.map((agent) => (
                  <label
                    key={agent.id}
                    className={`flex items-center gap-3 p-3 rounded-card cursor-pointer transition-colors ${selectedAgent === agent.id ? "bg-accent/8 ring-1 ring-accent/30" : "bg-surface hover:bg-surface-hover"}`}
                  >
                    <input
                      type="radio"
                      name="install-agent"
                      value={agent.id}
                      checked={selectedAgent === agent.id}
                      onChange={() => setSelectedAgent(agent.id)}
                      className="accent-accent"
                    />
                    <div className="flex-1">
                      <p className="text-sm font-medium text-ink">{agent.name}</p>
                      <p className="text-xs text-ink-muted">{agent.installPath}</p>
                    </div>
                    <StatusBadge tone={agent.status === "healthy" ? "stable" : "attention"} label={agent.status === "healthy" ? "正常" : "异常"} size="sm" />
                  </label>
                ))}
              </div>
            ) : (
              <p className="text-sm text-ink-soft py-2">未找到兼容的代理，请先安装对应的 AI 代理软件。</p>
            )}

            {availableAgents.length === 0 && agents.length > 0 && (
              <div className="mt-2">
                <p className="text-xs text-ink-muted mb-2">本机所有代理：</p>
                <div className="flex flex-col gap-2">
                  {agents.map((agent) => (
                    <label
                      key={agent.id}
                      className={`flex items-center gap-3 p-3 rounded-card cursor-pointer transition-colors ${selectedAgent === agent.id ? "bg-accent/8 ring-1 ring-accent/30" : "bg-surface hover:bg-surface-hover"}`}
                    >
                      <input
                        type="radio"
                        name="install-agent"
                        value={agent.id}
                        checked={selectedAgent === agent.id}
                        onChange={() => setSelectedAgent(agent.id)}
                        className="accent-accent"
                      />
                      <div className="flex-1">
                        <p className="text-sm font-medium text-ink">{agent.name}</p>
                        <p className="text-xs text-ink-muted">{agent.installPath}</p>
                      </div>
                      <span className="text-xs text-attention-ink">未在兼容列表</span>
                    </label>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={() => { setInstallTarget(null); setSelectedAgent(""); }}
              className="rounded-pill px-5 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors"
            >
              取消
            </button>
            <button
              type="button"
              onClick={handleConfirmInstall}
              disabled={!selectedAgent || installing}
              className="rounded-pill px-5 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {installing ? "安装中..." : "确认安装"}
            </button>
          </div>
        </div>
      </Modal>

      {/* Sources Management Modal */}
      <Modal open={showSources} onClose={() => setShowSources(false)} title="管理来源" subtitle="商店源管理">
        <div className="flex flex-col gap-4">
          {sources.length > 0 ? (
            sources.map((source) => (
              <div key={source.id} className="bg-surface rounded-card p-4">
                <div className="flex items-center justify-between">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <p className="font-medium text-ink">{source.name}</p>
                      {source.isBuiltin && (
                        <span className="rounded-chip px-2 py-0.5 text-[10px] font-medium bg-accent-glow text-accent">内置</span>
                      )}
                    </div>
                    <p className="text-xs text-ink-muted break-all mt-0.5">{source.url}</p>
                    <div className="flex gap-3 mt-1 text-xs text-ink-soft">
                      <span>{source.skillCount} 个技能</span>
                      <span>上次同步: {source.lastSyncedAt || "从未"}</span>
                      {source.lastSyncStatus && (
                        <StatusBadge
                          tone={source.lastSyncStatus === "success" ? "stable" : "critical"}
                          label={source.lastSyncStatus === "success" ? "成功" : "失败"}
                          size="sm"
                        />
                      )}
                    </div>
                  </div>
                  <div className="flex gap-2 items-center ml-3">
                    <StatusBadge tone={source.enabled ? "stable" : "muted"} label={source.enabled ? "已启用" : "已禁用"} />
                    <button
                      type="button"
                      onClick={() => handleSync(source.id)}
                      disabled={syncingSourceId === source.id}
                      className="rounded-chip px-3 py-1 text-xs font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors disabled:opacity-50"
                    >
                      {syncingSourceId === source.id ? "同步中" : "同步"}
                    </button>
                    {!source.isBuiltin && (
                      <button
                        type="button"
                        onClick={() => handleRemoveSource(source.id)}
                        className="rounded-chip px-3 py-1 text-xs font-medium bg-red-500/10 text-red-600 hover:opacity-80 transition-opacity"
                      >
                        移除
                      </button>
                    )}
                  </div>
                </div>
              </div>
            ))
          ) : (
            <p className="text-ink-soft text-center py-4">暂无商店源</p>
          )}
          <div className="border-t border-border-soft pt-4">
            <p className="text-sm font-medium text-ink mb-2">添加自定义来源</p>
            <div className="flex gap-2">
              <input
                type="text"
                placeholder="来源名称"
                value={addSourceName}
                onChange={(e) => setAddSourceName(e.target.value)}
                className="flex-1 bg-surface rounded-card px-3 py-2 text-sm text-ink placeholder:text-ink-muted border border-border focus:outline-none focus:ring-1 focus:ring-accent"
              />
              <input
                type="text"
                placeholder="GitHub URL"
                value={addSourceUrl}
                onChange={(e) => setAddSourceUrl(e.target.value)}
                className="flex-1 bg-surface rounded-card px-3 py-2 text-sm text-ink placeholder:text-ink-muted border border-border focus:outline-none focus:ring-1 focus:ring-accent"
              />
              <button
                type="button"
                onClick={handleAddSource}
                disabled={!addSourceName.trim() || !addSourceUrl.trim()}
                className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                添加
              </button>
            </div>
          </div>
        </div>
      </Modal>

      {/* Skill Detail Modal */}
      <Modal
        open={detailItem !== null}
        onClose={() => setDetailItem(null)}
        title={detailItem?.name ?? "技能详情"}
        subtitle="技能详情"
      >
        {detailItem && (
          <div className="flex flex-col gap-4">
            <div className="flex items-start justify-between">
              <div>
                <h2 className="font-display text-lg font-semibold text-ink">{detailItem.name}</h2>
                <p className="text-sm text-ink-muted">来自 {detailItem.source}</p>
              </div>
              <StatusBadge
                tone={detailItem.status === "installed" ? "stable" : "muted"}
                label={detailItem.status === "installed" ? "已安装" : "可用"}
              />
            </div>

            <p className="text-ink-soft">{detailItem.summary}</p>

            <div className="grid grid-cols-2 gap-4">
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">来源</p>
                <p className="text-sm text-ink">{detailItem.source}</p>
              </div>
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">作者</p>
                <p className="text-sm text-ink">{detailItem.author}</p>
              </div>
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">影响范围</p>
                <p className="text-sm text-ink">{detailItem.impact}</p>
              </div>
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">安装记录</p>
                <p className="text-sm text-ink">{detailItem.installs}</p>
              </div>
            </div>

            {detailItem.compatibility.length > 0 && (
              <div>
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">兼容代理</p>
                <div className="flex flex-wrap gap-1">
                  {detailItem.compatibility.map((agent) => (
                    <span key={agent} className="rounded-chip px-2.5 py-0.5 text-xs font-medium bg-chip text-chip-ink">{agent}</span>
                  ))}
                </div>
              </div>
            )}

            {detailItem.homepage && (
              <div>
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">主页</p>
                <a href={detailItem.homepage} target="_blank" rel="noopener noreferrer" className="text-sm text-accent hover:underline break-all">{detailItem.homepage}</a>
              </div>
            )}

            <div className="flex gap-2 pt-2">
              <button
                type="button"
                onClick={() => { setDetailItem(null); handleAskAI(detailItem); }}
                className="rounded-pill px-4 py-2 text-sm font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors"
              >
                AI 解读
              </button>
              {detailItem.status !== "installed" && (
                <button
                  type="button"
                  onClick={() => { setDetailItem(null); handleInstallClick(detailItem); }}
                  className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors"
                >
                  安装
                </button>
              )}
            </div>
          </div>
        )}
      </Modal>

      {/* AI Explain Modal */}
      <Modal
        open={explainItem !== null}
        onClose={() => { setExplainItem(null); setExplainResult(null); }}
        title={`AI 解读：${explainItem?.name ?? ""}`}
        subtitle="技能用途分析"
      >
        <div className="flex flex-col gap-4">
          {explainLoading ? (
            <div className="text-center py-8">
              <div className="inline-block w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin mb-3" />
              <p className="text-ink-soft">正在分析技能用途...</p>
            </div>
          ) : explainResult?.found ? (
            <>
              <div className="bg-accent-glow/30 rounded-card p-4">
                <p className="text-sm font-medium text-accent mb-1">💡 技能解读</p>
                <p className="text-sm text-ink">{explainItem?.summary}</p>
              </div>

              {explainResult.readmeContent && (
                <div>
                  <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">技能文档 ({explainResult.readmeFile})</p>
                  <div className="bg-surface rounded-card p-4 max-h-64 overflow-y-auto">
                    <pre className="text-xs text-ink-soft whitespace-pre-wrap font-mono">{explainResult.readmeContent}</pre>
                  </div>
                </div>
              )}

              {explainResult.skillPath && (
                <div>
                  <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">技能路径</p>
                  <a href={explainResult.skillPath} target="_blank" rel="noopener noreferrer" className="text-sm text-accent hover:underline break-all">{explainResult.skillPath}</a>
                </div>
              )}

              {explainResult.files?.length > 0 && (
                <div>
                  <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">包含文件</p>
                  <div className="flex flex-wrap gap-1">
                    {explainResult.files.map((file) => (
                      <span key={file} className="rounded-chip bg-surface px-2 py-0.5 text-xs text-ink-soft">{file}</span>
                    ))}
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="text-center py-8">
              <p className="text-ink-soft">未找到技能详细信息</p>
              <p className="text-xs text-ink-muted mt-1">该技能可能尚未安装或来源不可用</p>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            {explainItem && !explainLoading && (
              <>
                {explainItem.status !== "installed" && (
                  <button
                    type="button"
                    onClick={() => { setExplainItem(null); setExplainResult(null); handleInstallClick(explainItem); }}
                    className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors"
                  >
                    安装此技能
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => { setExplainItem(null); setExplainResult(null); }}
                  className="rounded-pill px-4 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors"
                >
                  关闭
                </button>
              </>
            )}
          </div>
        </div>
      </Modal>
    </section>
  );
}

import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { SkillViewModel, SkillDetailViewModel } from "../../lib/mocks";
import { selectApi } from "../../lib/api";
import { SearchBar } from "../../components/SearchBar";
import { FilterPanel } from "../../components/FilterPanel";
import { EmptyState } from "../../components/EmptyState";
import { Modal } from "../../components/Modal";
import { StatusBadge } from "../../components/StatusBadge";

interface SkillsPageProps {
  skills: SkillViewModel[];
  onRefresh?: () => void;
  onAction?: (action: string, agentID: string, skillName: string) => void;
}

type SortField = "name" | "installedAt" | "agent";
type SortOrder = "asc" | "desc";
type ViewMode = "list" | "grouped";

/** 技能页面组件：展示本机已安装的技能列表，支持搜索、筛选、批量操作和分组视图 */
export function SkillsPage({ skills, onRefresh, onAction }: SkillsPageProps) {
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [agentFilter, setAgentFilter] = useState("");
  const [healthFilter, setHealthFilter] = useState("");
  const [sortField, setSortField] = useState<SortField>("name");
  const [sortOrder, setSortOrder] = useState<SortOrder>("asc");
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [detailSkill, setDetailSkill] = useState<SkillDetailViewModel | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [confirmUninstall, setConfirmUninstall] = useState<SkillViewModel | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [showBatchConfirm, setShowBatchConfirm] = useState(false);
  const [batchAction, setBatchAction] = useState<"update" | "uninstall">("update");
  const [batchRunning, setBatchRunning] = useState(false);

  const agentNames = useMemo(() => Array.from(new Set(skills.map((s) => s.agent))).sort(), [skills]);

  const healthCounts = useMemo(() => ({
    ok: skills.filter((s) => s.healthStatus === "ok").length,
    warning: skills.filter((s) => s.healthStatus === "warning").length,
    error: skills.filter((s) => s.healthStatus === "error").length,
  }), [skills]);

  const filters = useMemo(() => [
    {
      key: "agent",
      label: "全部代理",
      options: agentNames.map((n) => ({ value: n, label: n })),
      value: agentFilter,
    },
    {
      key: "health",
      label: "全部状态",
      options: [
        { value: "ok", label: `正常 (${healthCounts.ok})` },
        { value: "warning", label: `异常 (${healthCounts.warning})` },
        { value: "error", label: `错误 (${healthCounts.error})` },
      ],
      value: healthFilter,
    },
  ], [agentNames, healthCounts, agentFilter, healthFilter]);

  const filtered = useMemo(() => {
    const result = skills.filter((s) => {
      const matchesSearch = s.name.toLowerCase().includes(search.toLowerCase()) || s.summary.toLowerCase().includes(search.toLowerCase());
      const matchesAgent = agentFilter === "" || s.agent === agentFilter;
      const matchesHealth = healthFilter === "" || s.healthStatus === healthFilter;
      return matchesSearch && matchesAgent && matchesHealth;
    });

    result.sort((a, b) => {
      let cmp = 0;
      switch (sortField) {
        case "name": cmp = a.name.localeCompare(b.name); break;
        case "installedAt": cmp = a.installedAt.localeCompare(b.installedAt); break;
        case "agent": cmp = a.agent.localeCompare(b.agent); break;
      }
      return sortOrder === "asc" ? cmp : -cmp;
    });

    return result;
  }, [skills, search, agentFilter, healthFilter, sortField, sortOrder]);

  /** 按代理分组 */
  const grouped = useMemo(() => {
    const groups: Record<string, SkillViewModel[]> = {};
    for (const skill of filtered) {
      if (!groups[skill.agent]) groups[skill.agent] = [];
      groups[skill.agent].push(skill);
    }
    return groups;
  }, [filtered]);

  const handleFilterChange = (key: string, value: string) => {
    if (key === "agent") setAgentFilter(value);
    if (key === "health") setHealthFilter(value);
  };

  /** 打开技能详情模态框 */
  const handleViewDetail = async (skill: SkillViewModel) => {
    setDetailLoading(true);
    try {
      const api = selectApi();
      const detail = await api.getSkillDetail(skill.agent, skill.name);
      setDetailSkill(detail);
    } catch {
      setDetailSkill({ id: skill.id, name: skill.name, agentId: skill.agent, agentName: skill.agent, version: "", author: "", description: skill.summary, tags: [], installPath: "", installedAt: skill.installedAt, source: "", files: [], projectCount: 0, found: false });
    } finally {
      setDetailLoading(false);
    }
  };

  /** 确认卸载技能 */
  const handleConfirmUninstall = () => {
    if (confirmUninstall) {
      onAction?.("uninstall", confirmUninstall.agent, confirmUninstall.name);
      setConfirmUninstall(null);
    }
  };

  /** 切换单个技能的选中状态 */
  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  /** 全选/取消全选 */
  const toggleSelectAll = () => {
    if (selectedIds.size === filtered.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(filtered.map((s) => s.id)));
    }
  };

  /** 执行批量操作 */
  const handleBatchAction = async () => {
    setBatchRunning(true);
    try {
      const selected = filtered.filter((s) => selectedIds.has(s.id));
      for (const skill of selected) {
        onAction?.(batchAction, skill.agent, skill.name);
      }
      setSelectedIds(new Set());
      setShowBatchConfirm(false);
      onRefresh?.();
    } finally {
      setBatchRunning(false);
    }
  };

  /** 一键更新所有技能 */
  const handleUpdateAll = async () => {
    setBatchRunning(true);
    try {
      for (const skill of skills.filter((s) => s.statusLabel === "已管理")) {
        onAction?.("update", skill.agent, skill.name);
      }
      onRefresh?.();
    } finally {
      setBatchRunning(false);
    }
  };

  /** 在 Finder 中打开技能目录 */
  const handleOpenInFinder = async (skill: SkillViewModel) => {
    try {
      const api = selectApi();
      const detail = await api.getSkillDetail(skill.agent, skill.name);
      if (detail.installPath) {
        await api.openInFinder(detail.installPath);
      }
    } catch (err) {
      alert(`打开失败: ${err instanceof Error ? err.message : String(err)}`);
    }
  };

  const allSelected = filtered.length > 0 && selectedIds.size === filtered.length;

  /** 渲染单个技能卡片 */
  function renderSkillCard(skill: SkillViewModel) {
    const isSelected = selectedIds.has(skill.id);
    const isAbnormal = skill.healthStatus === "warning" || skill.healthStatus === "error";
    return (
      <article
        key={skill.id}
        className={`bg-surface rounded-card shadow-panel p-5 hover:shadow-panel-hover transition-all ${isSelected ? "ring-1 ring-accent/30 bg-accent/4" : ""} ${skill.healthStatus === "error" ? "border-l-4 border-critical" : skill.healthStatus === "warning" ? "border-l-4 border-attention-ink" : ""}`}
      >
        <div className="flex items-start gap-3">
          <button
            type="button"
            onClick={() => toggleSelect(skill.id)}
            className={`shrink-0 w-5 h-5 mt-0.5 rounded border-2 flex items-center justify-center transition-colors ${
              isSelected ? "bg-accent border-accent text-white" : "border-border hover:border-accent/50"
            }`}
          >
            {isSelected && <span className="text-[10px]">✓</span>}
          </button>
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between mb-1">
              <div className="flex items-center gap-2">
                <h2 className="font-display text-base font-semibold text-ink">{skill.name}</h2>
                <span className={`w-2 h-2 rounded-full shrink-0 ${skill.healthStatus === "ok" ? "bg-stable-ink" : skill.healthStatus === "warning" ? "bg-attention-ink" : "bg-critical-ink"}`} />
              </div>
              <div className="flex items-center gap-2">
                <StatusBadge
                  tone={skill.healthStatus === "ok" ? "stable" : skill.healthStatus === "warning" ? "attention" : "critical"}
                  label={skill.healthStatus === "ok" ? "正常" : skill.healthStatus === "warning" ? "异常" : "错误"}
                  size="sm"
                />
              </div>
            </div>
            <p className="text-sm text-ink-soft mb-2 line-clamp-2">{skill.summary}</p>
            {isAbnormal && skill.healthMessage && (
              <div className={`text-xs px-3 py-2 rounded-card mb-2 ${skill.healthStatus === "error" ? "bg-critical/5 text-critical" : "bg-attention-bg/50 text-attention-ink"}`}>
                {skill.healthMessage}
              </div>
            )}
            <div className="flex items-center justify-between">
              <div className="flex gap-3 text-xs text-ink-muted">
                <span className="rounded-chip bg-surface-warm px-2 py-0.5">{skill.agent}</span>
                <span>{skill.group}</span>
                <span>{skill.projects} 个项目</span>
                <span>{skill.installedAt}</span>
              </div>
              <div className="flex gap-1.5">
                <button
                  onClick={() => handleViewDetail(skill)}
                  className="rounded-chip px-2.5 py-1 text-xs font-medium bg-surface-warm text-ink hover:bg-surface-hover transition-colors"
                >
                  详情
                </button>
                <button
                  onClick={() => handleOpenInFinder(skill)}
                  className="rounded-chip px-2.5 py-1 text-xs font-medium bg-surface-warm text-ink hover:bg-surface-hover transition-colors"
                  title="在 Finder 中打开"
                >
                  打开
                </button>
                {skill.healthStatus === "ok" && (
                  <>
                    <button
                      onClick={() => onAction?.("update", skill.agent, skill.name)}
                      className="rounded-chip px-2.5 py-1 text-xs font-medium bg-badge-bg text-badge-ink hover:opacity-80 transition-opacity"
                    >
                      更新
                    </button>
                    <button
                      onClick={() => setConfirmUninstall(skill)}
                      className="rounded-chip px-2.5 py-1 text-xs font-medium bg-red-500/10 text-red-600 hover:opacity-80 transition-opacity"
                    >
                      卸载
                    </button>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
      </article>
    );
  }

  return (
    <section className="animate-page-in">
      {/* Header */}
      <div className="bg-surface rounded-panel shadow-panel p-8 mb-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="uppercase tracking-widest text-xs text-ink-muted font-body">技能管理</p>
            <h1 className="font-display text-3xl font-semibold text-ink tracking-tight">已安装技能</h1>
            <p className="text-lg text-ink-soft leading-relaxed">查看和管理本机已安装的 AI 技能，支持批量操作和健康状态监控。</p>
          </div>
          <div className="flex gap-4 items-center">
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-ink">{skills.length}</p>
              <p className="text-xs text-ink-muted">总数</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-stable-ink">{healthCounts.ok}</p>
              <p className="text-xs text-ink-muted">正常</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-attention-ink">{healthCounts.warning}</p>
              <p className="text-xs text-ink-muted">异常</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-critical-ink">{healthCounts.error}</p>
              <p className="text-xs text-ink-muted">错误</p>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions Bar */}
      <div className="bg-surface rounded-panel shadow-panel p-4 mb-6 flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={handleUpdateAll}
            disabled={healthCounts.ok === 0 || batchRunning}
            className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {batchRunning ? "更新中..." : "一键更新全部"}
          </button>
          <div className="flex items-center gap-2 text-xs text-ink-muted">
            <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-stable-ink" />{healthCounts.ok} 正常</span>
            <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-attention-ink" />{healthCounts.warning} 异常</span>
            <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-critical-ink" />{healthCounts.error} 错误</span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {selectedIds.size > 0 && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-ink-muted">已选 {selectedIds.size} 项</span>
              <button
                type="button"
                onClick={() => { setBatchAction("update"); setShowBatchConfirm(true); }}
                className="rounded-chip px-3 py-1 text-xs font-medium bg-badge-bg text-badge-ink hover:opacity-80 transition-opacity"
              >
                批量更新
              </button>
              <button
                type="button"
                onClick={() => { setBatchAction("uninstall"); setShowBatchConfirm(true); }}
                className="rounded-chip px-3 py-1 text-xs font-medium bg-red-500/10 text-red-600 hover:opacity-80 transition-opacity"
              >
                批量卸载
              </button>
              <button
                type="button"
                onClick={() => setSelectedIds(new Set())}
                className="rounded-chip px-3 py-1 text-xs font-medium bg-surface-warm text-ink-muted hover:text-ink transition-colors"
              >
                取消选择
              </button>
            </div>
          )}
          <div className="flex rounded-card border border-border overflow-hidden">
            <button
              type="button"
              onClick={() => setViewMode("list")}
              className={`px-3 py-1.5 text-xs font-medium transition-colors ${viewMode === "list" ? "bg-accent text-white" : "bg-surface text-ink-soft hover:bg-surface-hover"}`}
            >
              列表
            </button>
            <button
              type="button"
              onClick={() => setViewMode("grouped")}
              className={`px-3 py-1.5 text-xs font-medium transition-colors ${viewMode === "grouped" ? "bg-accent text-white" : "bg-surface text-ink-soft hover:bg-surface-hover"}`}
            >
              分组
            </button>
          </div>
        </div>
      </div>

      {/* Search, Filter, Sort */}
      <div className="flex flex-wrap items-center gap-3 mb-6">
        <button
          type="button"
          onClick={toggleSelectAll}
          className={`shrink-0 w-5 h-5 rounded border-2 flex items-center justify-center transition-colors ${allSelected ? "bg-accent border-accent text-white" : "border-border hover:border-accent/50"}`}
        >
          {allSelected && <span className="text-[10px]">✓</span>}
        </button>
        <SearchBar value={search} onChange={setSearch} placeholder="搜索技能名称或描述…" />
        <FilterPanel filters={filters} onChange={handleFilterChange} />
        <select
          value={sortField}
          onChange={(e) => setSortField(e.target.value as SortField)}
          className="bg-surface rounded-card shadow-panel px-4 py-2 text-sm text-ink focus:outline-none focus:ring-2 focus:ring-accent/40"
        >
          <option value="name">按名称排序</option>
          <option value="installedAt">按安装时间排序</option>
          <option value="agent">按代理排序</option>
        </select>
        <button
          type="button"
          onClick={() => setSortOrder((o) => o === "asc" ? "desc" : "asc")}
          className="bg-surface rounded-card shadow-panel px-3 py-2 text-sm text-ink hover:bg-surface-hover transition-colors"
          title={sortOrder === "asc" ? "升序" : "降序"}
        >
          {sortOrder === "asc" ? "↑" : "↓"}
        </button>
        {onRefresh && (
          <button onClick={onRefresh} className="rounded-pill px-4 py-2 text-sm font-medium bg-surface-warm shadow-panel hover:shadow-panel-hover transition-shadow text-ink">
            刷新
          </button>
        )}
      </div>

      {/* Skills Content */}
      {filtered.length > 0 ? (
        viewMode === "list" ? (
          <div className="flex flex-col gap-3">
            {filtered.map(renderSkillCard)}
          </div>
        ) : (
          <div className="flex flex-col gap-6">
            {Object.entries(grouped).sort(([a], [b]) => a.localeCompare(b)).map(([agent, agentSkills]) => (
              <div key={agent}>
                <div className="flex items-center gap-3 mb-3">
                  <h2 className="font-display text-lg font-semibold text-ink">{agent}</h2>
                  <span className="rounded-chip bg-stable-bg text-stable-ink px-2.5 py-0.5 text-xs font-medium">{agentSkills.length} 个技能</span>
                </div>
                <div className="flex flex-col gap-3">
                  {agentSkills.map(renderSkillCard)}
                </div>
              </div>
            ))}
          </div>
        )
      ) : (
        <EmptyState
          title="暂无已安装的技能"
          description="前往商店浏览和安装技能"
          action={{ label: "前往商店", onClick: () => navigate("/store") }}
        />
      )}

      {/* Skill Detail Modal */}
      <Modal
        open={detailSkill !== null || detailLoading}
        onClose={() => setDetailSkill(null)}
        title={detailSkill?.name ?? "技能详情"}
        subtitle="技能详情"
      >
        {detailLoading ? (
          <div className="text-center py-8"><p className="text-ink-soft">正在加载技能详情...</p></div>
        ) : detailSkill?.found ? (
          <div className="flex flex-col gap-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">所属代理</p>
                <p className="text-sm text-ink">{detailSkill.agentName}</p>
              </div>
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">来源</p>
                <p className="text-sm text-ink">{detailSkill.source || "未知"}</p>
              </div>
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">安装路径</p>
                <p className="text-sm text-ink break-all">{detailSkill.installPath || "未知"}</p>
              </div>
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">安装时间</p>
                <p className="text-sm text-ink">{detailSkill.installedAt || "未知"}</p>
              </div>
            </div>
            {detailSkill.description && (
              <div className="bg-surface rounded-card p-4">
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">描述</p>
                <p className="text-sm text-ink leading-relaxed">{detailSkill.description}</p>
              </div>
            )}
            {detailSkill.tags?.length > 0 && (
              <div>
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">标签</p>
                <div className="flex flex-wrap gap-1">
                  {detailSkill.tags.map((tag) => (
                    <span key={tag} className="rounded-chip bg-accent-glow text-accent px-2 py-0.5 text-xs font-medium">{tag}</span>
                  ))}
                </div>
              </div>
            )}
            {detailSkill.files?.length > 0 && (
              <div>
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">目录文件</p>
                <div className="flex flex-wrap gap-1">
                  {detailSkill.files.map((file) => (
                    <span key={file} className="rounded-chip bg-surface px-2 py-0.5 text-xs text-ink-soft">{file}</span>
                  ))}
                </div>
              </div>
            )}
            <div className="flex gap-2 mt-2">
              {detailSkill.installPath && (
                <button
                  type="button"
                  onClick={async () => {
                    try { await selectApi().openInFinder(detailSkill.installPath); } catch { /* ignore */ }
                  }}
                  className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors"
                >
                  在 Finder 中打开
                </button>
              )}
              <button
                type="button"
                onClick={() => onAction?.("update", detailSkill.agentId, detailSkill.name)}
                className="rounded-pill px-4 py-2 text-sm font-medium bg-badge-bg text-badge-ink hover:opacity-80 transition-opacity"
              >
                更新
              </button>
              <button
                type="button"
                onClick={() => { setDetailSkill(null); setConfirmUninstall({ id: detailSkill.id, name: detailSkill.name, group: detailSkill.agentName, installedAt: detailSkill.installedAt, summary: detailSkill.description, statusLabel: "已管理", projects: detailSkill.projectCount, agent: detailSkill.agentName, healthStatus: "ok" as const, healthMessage: "" }); }}
                className="rounded-pill px-4 py-2 text-sm font-medium bg-red-500/10 text-red-600 hover:opacity-80 transition-opacity"
              >
                卸载
              </button>
            </div>
          </div>
        ) : (
          <div className="text-center py-8"><p className="text-ink-soft">未找到技能详情</p></div>
        )}
      </Modal>

      {/* Confirm Uninstall Modal */}
      {confirmUninstall && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setConfirmUninstall(null)}>
          <article className="bg-surface-warm rounded-panel shadow-panel max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="p-6">
              <h2 className="font-display text-xl font-semibold text-ink mb-2">确认卸载</h2>
              <p className="text-ink-soft">确定要卸载技能「{confirmUninstall.name}」吗？此操作不可撤销。</p>
            </div>
            <div className="flex justify-end gap-3 px-6 py-4 border-t border-border-soft">
              <button type="button" onClick={() => setConfirmUninstall(null)} className="rounded-pill px-5 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors">取消</button>
              <button type="button" onClick={handleConfirmUninstall} className="rounded-pill px-5 py-2 text-sm font-medium bg-red-500 text-white hover:bg-red-600 transition-colors">确认卸载</button>
            </div>
          </article>
        </div>
      )}

      {/* Batch Action Confirm Modal */}
      <Modal
        open={showBatchConfirm}
        onClose={() => setShowBatchConfirm(false)}
        title={batchAction === "update" ? "批量更新" : "批量卸载"}
        subtitle="确认操作"
      >
        <div className="flex flex-col gap-4">
          <p className="text-ink-soft">
            确定要{batchAction === "update" ? "更新" : "卸载"}以下 <strong className="text-ink">{selectedIds.size}</strong> 个技能吗？
            {batchAction === "uninstall" && <span className="text-red-500"> 此操作不可撤销。</span>}
          </p>
          <div className="bg-surface rounded-card p-4 max-h-40 overflow-y-auto">
            <div className="flex flex-wrap gap-1">
              {filtered.filter((s) => selectedIds.has(s.id)).map((skill) => (
                <span key={skill.id} className="rounded-chip bg-surface-warm px-2 py-0.5 text-xs text-ink-soft">{skill.name}</span>
              ))}
            </div>
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => setShowBatchConfirm(false)} className="rounded-pill px-5 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors">取消</button>
            <button
              type="button"
              onClick={handleBatchAction}
              disabled={batchRunning}
              className={`rounded-pill px-5 py-2 text-sm font-medium text-white transition-colors disabled:opacity-40 ${batchAction === "uninstall" ? "bg-red-500 hover:bg-red-600" : "bg-accent hover:bg-accent-warm"}`}
            >
              {batchRunning ? "执行中..." : batchAction === "update" ? "确认更新" : "确认卸载"}
            </button>
          </div>
        </div>
      </Modal>
    </section>
  );
}

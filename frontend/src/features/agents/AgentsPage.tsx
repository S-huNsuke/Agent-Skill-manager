import { useState } from "react";
import type { AgentViewModel, AgentDetailViewModel, SkillExplanationViewModel, SkillViewModel } from "../../lib/types";
import { selectApi } from "../../lib/api";
import { StatusBadge } from "../../components/StatusBadge";
import { Modal } from "../../components/Modal";

interface AgentsPageProps {
  agents: AgentViewModel[];
  onRefresh?: () => void;
  onAction?: (action: string, agentID: string, skillName?: string) => void;
  skills?: SkillViewModel[];
}

/** 代理列表页面：展示本机已发现的 AI Agent，支持查看详情、批量操作、快速修复 */
export function AgentsPage({ agents, onRefresh, onAction, skills }: AgentsPageProps) {
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null);
  const [agentDetail, setAgentDetail] = useState<AgentDetailViewModel | null>(null);
  const [loading, setLoading] = useState(false);
  const [skillSearch, setSkillSearch] = useState("");
  const [explainingSkill, setExplainingSkill] = useState<SkillExplanationViewModel | null>(null);
  const [explainLoading, setExplainLoading] = useState(false);
  const [aiExplanation, setAiExplanation] = useState<string | null>(null);
  const [aiLoading, setAiLoading] = useState(false);
  const [showBatchUpdate, setShowBatchUpdate] = useState(false);
  const [batchUpdating, setBatchUpdating] = useState(false);

  const getAgentSkills = (agentName: string) =>
    (skills ?? []).filter((s) => s.agent === agentName);

  /** 点击代理卡片，展开详情面板 */
  const handleSelectAgent = async (agent: AgentViewModel) => {
    if (selectedAgentId === agent.id) {
      setSelectedAgentId(null);
      setAgentDetail(null);
      return;
    }
    setSelectedAgentId(agent.id);
    setLoading(true);
    setSkillSearch("");
    try {
      const api = selectApi();
      const detail = await api.getAgentDetail(agent.id);
      setAgentDetail(detail);
    } catch {
      setAgentDetail(null);
    } finally {
      setLoading(false);
    }
  };

  /** 在 Finder 中打开路径 */
  const handleOpenInFinder = async (path: string) => {
    try {
      const api = selectApi();
      const result = await api.openInFinder(path);
      if (result !== "ok" && !result.startsWith("ok")) {
        alert(`打开失败: ${result}`);
      }
    } catch (err) {
      alert(`打开出错: ${err instanceof Error ? err.message : String(err)}`);
    }
  };

  /** 修复代理（创建缺失的 skills 目录） */
  const handleRepair = async (agentID: string) => {
    try {
      const api = selectApi();
      const result = await api.repairAgent(agentID);
      if (result === "ok" || result.startsWith("ok")) {
        onRefresh?.();
      } else {
        alert(`修复失败: ${result}`);
      }
    } catch (err) {
      alert(`修复出错: ${err instanceof Error ? err.message : String(err)}`);
    }
  };

  /** 复制路径到剪贴板 */
  const handleCopyPath = async (path: string) => {
    try {
      await navigator.clipboard.writeText(path);
    } catch {
      const textarea = document.createElement("textarea");
      textarea.value = path;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
    }
  };

  /** 询问 AI 技能的作用，先快速展示基本信息，再异步加载 AI 解读 */
  const handleExplainSkill = async (agentID: string, skillName: string) => {
    setExplainLoading(true);
    setAiExplanation(null);
    setAiLoading(false);
    try {
      const api = selectApi();
      const explanation = await api.explainSkill(agentID, skillName);
      setExplainingSkill(explanation);
      setExplainLoading(false);

      if (explanation.found) {
        setAiLoading(true);
        try {
          const aiResult = await api.generateSkillExplanation(agentID, skillName);
          setAiExplanation(aiResult || null);
        } catch {
          setAiExplanation(null);
        } finally {
          setAiLoading(false);
        }
      }
    } catch (err) {
      setExplainingSkill({
        agentId: agentID,
        agentName: "",
        skillName: skillName,
        found: false,
        skillPath: "",
        readmeFile: "",
        readmeContent: `获取技能信息失败: ${err instanceof Error ? err.message : String(err)}`,
        files: [],
        aiExplanation: "",
      });
      setExplainLoading(false);
    }
  };

  /** 批量更新代理下所有技能 */
  const handleBatchUpdate = async () => {
    if (!agentDetail?.id) return;
    setBatchUpdating(true);
    try {
      const api = selectApi();
      const names = agentDetail.skillNames?.join(",") ?? "";
      const result = await api.batchUpdateSkills(agentDetail.id, names);
      if (result === "ok") {
        setShowBatchUpdate(false);
        onRefresh?.();
      } else {
        alert(`批量更新失败: ${result}`);
      }
    } catch (err) {
      alert(`批量更新出错: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setBatchUpdating(false);
    }
  };

  const healthLabel: Record<string, string> = {
    ready: "运行正常",
    not_installed: "未安装",
    installed_but_unreadable: "读取异常",
    installed_but_skill_path_missing: "缺少技能目录",
    installed_but_skill_path_empty: "技能为空",
  };

  const filteredSkillNames = agentDetail?.skillNames
    ? agentDetail.skillNames.filter((name) =>
        skillSearch ? name.toLowerCase().includes(skillSearch.toLowerCase()) : true,
      )
    : [];

  const totalSkills = agents.reduce((sum, a) => sum + a.skills, 0);
  const healthyAgents = agents.filter((a) => a.status === "healthy").length;

  return (
    <section className="animate-page-in">
      <div className="bg-surface rounded-panel shadow-panel p-8 mb-6 flex items-start justify-between gap-8">
        <div>
          <p className="uppercase tracking-widest text-xs text-ink-muted font-body">代理管理</p>
          <h1 className="font-display text-3xl font-semibold text-ink tracking-tight">本地 AI 代理</h1>
          <p className="text-lg text-ink-soft leading-relaxed">查看本机 AI 代理的运行状态、技能目录和详细信息。</p>
        </div>
        <div className="flex gap-3 items-center">
          <div className="flex gap-4 mr-4">
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-ink">{agents.length}</p>
              <p className="text-xs text-ink-muted">代理总数</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-stable-ink">{healthyAgents}</p>
              <p className="text-xs text-ink-muted">运行正常</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-accent">{totalSkills}</p>
              <p className="text-xs text-ink-muted">技能总数</p>
            </div>
          </div>
          {onRefresh && (
            <button
              onClick={onRefresh}
              className="rounded-pill px-4 py-2 text-sm font-medium bg-surface-warm shadow-panel hover:shadow-panel-hover transition-shadow text-ink"
            >
              刷新
            </button>
          )}
        </div>
      </div>

      {agents.length > 0 ? (
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-3">
            {agents.map((agent) => {
              const isSelected = selectedAgentId === agent.id;
              return (
                <button
                  key={agent.id}
                  type="button"
                  onClick={() => handleSelectAgent(agent)}
                  className={`text-left w-full rounded-card p-5 transition-all duration-200 relative overflow-hidden ${
                    isSelected
                      ? "bg-accent/8 shadow-panel ring-1 ring-accent/30"
                      : "bg-surface shadow-panel hover:shadow-panel-hover"
                  }`}
                >
                  {isSelected && (
                    <span className="absolute left-0 top-0 bottom-0 w-1 bg-accent rounded-r-full" />
                  )}
                  <div className="flex items-center justify-between mb-2">
                    <h2 className={`font-display text-lg font-semibold ${isSelected ? "text-accent" : "text-ink"}`}>{agent.name}</h2>
                    <span
                      className={`rounded-chip px-2.5 py-0.5 text-xs font-medium shrink-0 ${
                        agent.status === "healthy"
                          ? "bg-stable-bg text-stable-ink"
                          : "bg-attention-bg text-attention-ink"
                      }`}
                    >
                      {agent.status === "healthy" ? "正常" : "异常"}
                    </span>
                  </div>
                  <p className="text-sm text-ink-soft mb-2">{agent.mode}</p>
                  <div className="flex items-center gap-3">
                    <span className="rounded-chip bg-stable-bg text-stable-ink px-2 py-0.5 text-xs font-medium">
                      {agent.skills} 个技能
                    </span>
                  </div>
                </button>
              );
            })}
          </div>

          <div className="min-w-0">
            {selectedAgentId && agentDetail && agentDetail.found ? (
              <div className="flex flex-col gap-5">
                <article className="bg-surface rounded-panel shadow-panel p-6">
                  <div className="flex items-center justify-between mb-5">
                    <div>
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">代理详情</p>
                      <h2 className="font-display text-2xl font-semibold text-ink">{agentDetail.displayName}</h2>
                    </div>
                    <StatusBadge
                      tone={agentDetail.health === "ready" ? "stable" : agentDetail.health === "not_installed" ? "muted" : "attention"}
                      label={healthLabel[agentDetail.health] ?? agentDetail.health}
                      size="md"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-4 mb-5">
                    <div className="bg-surface-warm rounded-card p-4">
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">安装路径</p>
                      {agentDetail.installPaths && agentDetail.installPaths.length > 1 ? (
                        <div className="flex flex-col gap-1">
                          {agentDetail.installPaths.map((path, idx) => (
                            <p key={idx} className="text-sm text-ink break-all">{path}</p>
                          ))}
                        </div>
                      ) : (
                        <p className="text-sm text-ink break-all">{agentDetail.installPath || "未知"}</p>
                      )}
                    </div>
                    <div className="bg-surface-warm rounded-card p-4">
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">技能目录</p>
                      {agentDetail.skillsPaths && agentDetail.skillsPaths.length > 1 ? (
                        <div className="flex flex-col gap-1">
                          {agentDetail.skillsPaths.map((path, idx) => (
                            <p key={idx} className="text-sm text-ink break-all">{path}</p>
                          ))}
                        </div>
                      ) : (
                        <p className="text-sm text-ink break-all">{agentDetail.skillsPath || "未配置"}</p>
                      )}
                    </div>
                    <div className="bg-surface-warm rounded-card p-4">
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">技能数量</p>
                      <p className="text-sm text-ink">{agentDetail.skillCount} 个</p>
                    </div>
                    <div className="bg-surface-warm rounded-card p-4">
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">上次扫描</p>
                      <p className="text-sm text-ink">{agentDetail.lastScannedAt || "未扫描"}</p>
                    </div>
                  </div>

                  {agentDetail.lastErrorMessage && (
                    <div className="bg-attention-bg/30 border border-attention-ink/20 rounded-card p-4 mb-5">
                      <p className="text-sm font-medium text-attention-ink mb-1">错误信息</p>
                      <p className="text-sm text-ink-soft">{agentDetail.lastErrorMessage}</p>
                      {agentDetail.lastErrorCode && (
                        <p className="text-xs text-ink-muted mt-1">错误码: {agentDetail.lastErrorCode}</p>
                      )}
                    </div>
                  )}

                  <div className="flex flex-wrap gap-3">
                    {agentDetail.installPath && (
                      <button
                        type="button"
                        onClick={() => handleOpenInFinder(agentDetail.installPath)}
                        className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors"
                      >
                        在 Finder 中打开
                      </button>
                    )}
                    {agentDetail.skillsPath && agentDetail.health === "ready" && (
                      <button
                        type="button"
                        onClick={() => handleOpenInFinder(agentDetail.skillsPath)}
                        className="rounded-pill px-4 py-2 text-sm font-medium bg-surface-warm text-ink border border-border hover:bg-surface-hover transition-colors"
                      >
                        打开技能目录
                      </button>
                    )}
                    {agentDetail.installPath && (
                      <button
                        type="button"
                        onClick={() => handleCopyPath(agentDetail.installPath)}
                        className="rounded-pill px-4 py-2 text-sm font-medium bg-surface-warm text-ink border border-border hover:bg-surface-hover transition-colors"
                      >
                        复制路径
                      </button>
                    )}
                    {(agentDetail.health === "installed_but_skill_path_missing" ||
                      agentDetail.health === "installed_but_skill_path_empty") && (
                      <button
                        type="button"
                        onClick={() => handleRepair(agentDetail.id)}
                        className="rounded-pill px-4 py-2 text-sm font-medium bg-attention-bg text-attention-ink hover:opacity-80 transition-opacity"
                      >
                        修复技能目录
                      </button>
                    )}
                    {agentDetail.health === "ready" && agentDetail.skillCount > 0 && (
                      <button
                        type="button"
                        onClick={() => setShowBatchUpdate(true)}
                        className="rounded-pill px-4 py-2 text-sm font-medium bg-badge-bg text-badge-ink hover:opacity-80 transition-opacity"
                      >
                        批量更新技能
                      </button>
                    )}
                  </div>
                </article>

                {agentDetail.health === "ready" && agentDetail.skillCount > 0 && (
                  <article className="bg-surface rounded-panel shadow-panel p-6">
                    <div className="flex items-center justify-between mb-4">
                      <div>
                        <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">已安装技能</p>
                        <h3 className="font-display text-lg font-semibold text-ink">
                          {agentDetail.skillCount} 个技能
                        </h3>
                      </div>
                      <div className="flex gap-2 items-center">
                        <input
                          type="text"
                          placeholder="搜索技能..."
                          value={skillSearch}
                          onChange={(e) => setSkillSearch(e.target.value)}
                          className="rounded-pill px-3 py-1.5 text-sm bg-surface-warm border border-border focus:outline-none focus:ring-1 focus:ring-accent"
                        />
                        {agentDetail.skillsPath && (
                          <button
                            type="button"
                            onClick={() => handleOpenInFinder(agentDetail.skillsPath)}
                            className="rounded-chip px-3 py-1.5 text-xs font-medium bg-surface shadow-panel hover:shadow-panel-hover transition-shadow text-ink"
                          >
                            在 Finder 中查看
                          </button>
                        )}
                      </div>
                    </div>
                    <div className="grid grid-cols-3 gap-2">
                      {filteredSkillNames.map((name) => {
                        const skillInfo = getAgentSkills(agentDetail.displayName).find(
                          (s) => s.name === name,
                        );
                        return (
                          <div
                            key={name}
                            className="bg-surface-warm rounded-card px-3 py-2 flex items-center justify-between group"
                          >
                            <div className="flex items-center gap-2 min-w-0 flex-1">
                              <span className="text-sm text-ink font-medium truncate">{name}</span>
                              {skillInfo && (
                                <span
                                  className={`rounded-chip px-2 py-0.5 text-[10px] font-medium shrink-0 ${
                                    skillInfo.statusLabel === "已管理"
                                      ? "bg-stable-bg text-stable-ink"
                                      : "bg-badge-bg text-badge-ink"
                                  }`}
                                >
                                  {skillInfo.statusLabel}
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                              <button
                                type="button"
                                onClick={() => handleExplainSkill(agentDetail.id, name)}
                                className="shrink-0 rounded-chip px-2 py-0.5 text-[10px] font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white"
                                title="询问 AI 这个技能的作用"
                              >
                                AI
                              </button>
                              <button
                                type="button"
                                onClick={() => onAction?.("uninstall", agentDetail.id, name)}
                                className="shrink-0 rounded-chip px-2 py-0.5 text-[10px] font-medium bg-red-500/10 text-red-600 hover:opacity-80"
                                title="卸载此技能"
                              >
                                ✕
                              </button>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                    {skillSearch && filteredSkillNames.length === 0 && (
                      <p className="text-sm text-ink-muted mt-3 text-center">没有匹配「{skillSearch}」的技能</p>
                    )}
                  </article>
                )}

                {agentDetail.health === "ready" && agentDetail.skillCount === 0 && (
                  <article className="bg-surface rounded-panel shadow-panel p-8 text-center">
                    <p className="text-ink-soft">技能目录为空</p>
                    <p className="text-ink-muted text-sm mt-2">
                      前往商店页面浏览并安装技能，或使用 AI 助手自动推荐。
                    </p>
                  </article>
                )}
              </div>
            ) : selectedAgentId && loading ? (
              <article className="bg-surface rounded-panel shadow-panel p-12 text-center">
                <p className="text-ink-soft">正在加载代理详情...</p>
              </article>
            ) : (
              <article className="bg-surface rounded-panel shadow-panel p-12 text-center">
                <p className="text-ink-soft text-lg">点击左侧代理查看详情</p>
                <p className="text-ink-muted text-sm mt-2">
                  选择一个代理以查看安装路径、技能列表和运行状态。
                </p>
              </article>
            )}
          </div>
        </div>
      ) : (
        <div className="bg-surface rounded-panel shadow-panel p-12 text-center">
          <p className="text-ink-soft text-lg">未检测到本地 AI 代理</p>
          <p className="text-ink-muted text-sm mt-2">请确保已安装至少一个 AI 代理（如 Codex、Claude Code、Gemini CLI）</p>
        </div>
      )}

      {/* 批量更新确认弹窗 */}
      <Modal
        open={showBatchUpdate}
        onClose={() => setShowBatchUpdate(false)}
        title="批量更新技能"
        subtitle="确认操作"
      >
        <div className="flex flex-col gap-4">
          <p className="text-ink-soft">确定要更新 <strong className="text-ink">{agentDetail?.displayName}</strong> 下的所有技能吗？</p>
          {agentDetail?.skillNames && agentDetail.skillNames.length > 0 && (
            <div className="bg-surface rounded-card p-4 max-h-40 overflow-y-auto">
              <p className="text-xs text-ink-muted mb-2">将更新以下 {agentDetail.skillNames.length} 个技能：</p>
              <div className="flex flex-wrap gap-1">
                {agentDetail.skillNames.map((name) => (
                  <span key={name} className="rounded-chip bg-surface-warm px-2 py-0.5 text-xs text-ink-soft">{name}</span>
                ))}
              </div>
            </div>
          )}
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => setShowBatchUpdate(false)} className="rounded-pill px-5 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors">取消</button>
            <button type="button" onClick={handleBatchUpdate} disabled={batchUpdating} className="rounded-pill px-5 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40">
              {batchUpdating ? "更新中..." : "确认更新"}
            </button>
          </div>
        </div>
      </Modal>

      {/* AI 技能分析弹窗 */}
      {explainingSkill && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => { setExplainingSkill(null); setAiExplanation(null); setAiLoading(false); }}>
          <article
            className="bg-surface-warm rounded-panel shadow-panel max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-6 border-b border-border-soft">
              <div>
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-1">AI 技能分析</p>
                <h2 className="font-display text-xl font-semibold text-ink">
                  {explainingSkill.skillName}
                </h2>
                <p className="text-sm text-ink-soft">
                  来自 {explainingSkill.agentName || explainingSkill.agentId}
                </p>
              </div>
              <button
                type="button"
                onClick={() => { setExplainingSkill(null); setAiExplanation(null); setAiLoading(false); }}
                className="w-8 h-8 rounded-full bg-surface hover:bg-surface-hover flex items-center justify-center text-ink-soft hover:text-ink transition-colors"
              >
                ✕
              </button>
            </div>
            <div className="p-6 overflow-y-auto flex-1">
              {explainLoading ? (
                <div className="text-center py-8">
                  <div className="inline-block w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin mb-3" />
                  <p className="text-ink-soft">正在分析技能用途...</p>
                </div>
              ) : explainingSkill.found ? (
                <div className="flex flex-col gap-4">
                  {aiLoading ? (
                    <div className="bg-accent-glow/30 rounded-card p-4">
                      <div className="flex items-center gap-2">
                        <div className="inline-block w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin" />
                        <p className="text-sm font-medium text-accent">AI 正在解读...</p>
                      </div>
                    </div>
                  ) : aiExplanation ? (
                    <div className="bg-accent-glow/30 rounded-card p-4">
                      <p className="text-sm font-medium text-accent mb-1">💡 AI 解读</p>
                      <p className="text-sm text-ink">{aiExplanation}</p>
                    </div>
                  ) : (
                    <div className="bg-accent-glow/30 rounded-card p-4">
                      <p className="text-sm font-medium text-accent mb-1">💡 技能简介</p>
                      <p className="text-sm text-ink">{explainingSkill.readmeContent?.split("\n")[0] || "暂无简介"}</p>
                    </div>
                  )}

                  {explainingSkill.skillPath && (
                    <div className="bg-surface rounded-card p-3 flex items-center justify-between">
                      <span className="text-sm text-ink-soft break-all">{explainingSkill.skillPath}</span>
                      <button
                        type="button"
                        onClick={() => handleCopyPath(explainingSkill.skillPath)}
                        className="rounded-chip px-2 py-0.5 text-xs font-medium bg-surface-warm hover:bg-surface-hover text-ink shrink-0 ml-2"
                      >
                        复制
                      </button>
                    </div>
                  )}
                  {explainingSkill.readmeFile && (
                    <div className="text-xs text-ink-muted">
                      描述来源: {explainingSkill.readmeFile}
                    </div>
                  )}
                  {explainingSkill.readmeContent && (
                    <div>
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">技能文档 ({explainingSkill.readmeFile})</p>
                      <div className="bg-surface rounded-card p-4 max-h-64 overflow-y-auto">
                        <pre className="text-xs text-ink-soft whitespace-pre-wrap font-mono">{explainingSkill.readmeContent}</pre>
                      </div>
                    </div>
                  )}
                  {explainingSkill.files?.length > 0 && (
                    <div>
                      <p className="text-xs text-ink-muted mb-2">目录文件:</p>
                      <div className="flex flex-wrap gap-1">
                        {explainingSkill.files.map((file) => (
                          <span key={file} className="rounded-chip bg-surface px-2 py-0.5 text-xs text-ink-soft">
                            {file}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                  {explainingSkill.skillPath && (
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => handleOpenInFinder(explainingSkill.skillPath)}
                        className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors"
                      >
                        在 Finder 中打开
                      </button>
                      <button
                        type="button"
                        onClick={() => onAction?.("uninstall", explainingSkill.agentId, explainingSkill.skillName)}
                        className="rounded-pill px-4 py-2 text-sm font-medium bg-red-500/10 text-red-600 hover:opacity-80 transition-opacity"
                      >
                        卸载此技能
                      </button>
                    </div>
                  )}
                </div>
              ) : (
                <div className="text-center py-8">
                  <p className="text-ink-soft">未找到技能信息</p>
                  <p className="text-sm text-ink-muted mt-2">{explainingSkill.readmeContent}</p>
                </div>
              )}
            </div>
          </article>
        </div>
      )}
    </section>
  );
}

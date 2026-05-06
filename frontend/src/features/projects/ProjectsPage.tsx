import { useEffect, useMemo, useState, useCallback } from "react";
import type { ProjectViewModel, SkillGroupViewModel, AgentViewModel, SkillViewModel, StoreItemViewModel, ReconcilePlanViewModel } from "../../lib/mocks";
import { selectApi } from "../../lib/api";
import { EmptyState } from "../../components/EmptyState";
import { Modal } from "../../components/Modal";
import { StatusBadge } from "../../components/StatusBadge";

interface ProjectsPageProps {
  projects: ProjectViewModel[];
  agents: AgentViewModel[];
  skills?: SkillViewModel[];
  storeItems?: StoreItemViewModel[];
  onRefresh?: () => void;
}

/** 项目管理页面：新建/选择项目，绑定代理和技能组 */
export function ProjectsPage({ projects, agents, skills, storeItems, onRefresh }: ProjectsPageProps) {
  const [activeProjectId, setActiveProjectId] = useState(projects[0]?.id ?? "");
  const [skillGroups, setSkillGroups] = useState<SkillGroupViewModel[]>([]);
  const [showCreateProject, setShowCreateProject] = useState(false);
  const [newProjectName, setNewProjectName] = useState("");
  const [newProjectPath, setNewProjectPath] = useState("");
  const [showCreateGroup, setShowCreateGroup] = useState(false);
  const [newGroupName, setNewGroupName] = useState("");
  const [newGroupDesc, setNewGroupDesc] = useState("");
  const [newGroupSelectedSkills, setNewGroupSelectedSkills] = useState<string[]>([]);
  const [newGroupSelectedAgent, setNewGroupSelectedAgent] = useState("");
  const [showBindAgent, setShowBindAgent] = useState(false);
  const [showBindGroup, setShowBindGroup] = useState(false);
  const [showGroupDetail, setShowGroupDetail] = useState<SkillGroupViewModel | null>(null);
  const [addSkillName, setAddSkillName] = useState("");
  const [skillSearchQuery, setSkillSearchQuery] = useState("");
  const [reconcilePlan, setReconcilePlan] = useState<ReconcilePlanViewModel | null>(null);
  const [reconcileExecuting, setReconcileExecuting] = useState(false);

  const activeProject = useMemo(
    () => projects.find((p) => p.id === activeProjectId) ?? projects[0],
    [activeProjectId, projects],
  );

  /** 合并已安装技能和商店技能，去重后作为可选技能列表 */
  const availableSkills = useMemo(() => {
    const seen = new Set<string>();
    const result: { name: string; source: string }[] = [];
    if (skills) {
      for (const s of skills) {
        if (!seen.has(s.name)) {
          seen.add(s.name);
          result.push({ name: s.name, source: "已安装" });
        }
      }
    }
    if (storeItems) {
      for (const s of storeItems) {
        if (!seen.has(s.name)) {
          seen.add(s.name);
          result.push({ name: s.name, source: s.source || "商店" });
        }
      }
    }
    return result;
  }, [skills, storeItems]);

  /** 根据搜索词过滤技能 */
  const filteredAvailableSkills = useMemo(() => {
    if (!skillSearchQuery.trim()) return availableSkills;
    const q = skillSearchQuery.toLowerCase();
    return availableSkills.filter((s) => s.name.toLowerCase().includes(q));
  }, [availableSkills, skillSearchQuery]);

  /** 加载技能组列表 */
  const loadSkillGroups = useCallback(async () => {
    try {
      const api = selectApi();
      const groups = await api.getSkillGroups();
      setSkillGroups(groups);
    } catch {
      setSkillGroups([]);
    }
  }, []);

  useEffect(() => {
    loadSkillGroups();
  }, [loadSkillGroups]);

  /** 打开文件管理器选择目录 */
  const handleSelectDirectory = async () => {
    try {
      const api = selectApi();
      const dir = await api.selectDirectory("选择项目目录");
      if (dir) {
        setNewProjectPath(dir);
        if (!newProjectName.trim()) {
          const parts = dir.split("/");
          const dirName = parts[parts.length - 1] || "新项目";
          setNewProjectName(dirName);
        }
      }
    } catch {
      // 静默处理
    }
  };

  /** 创建新项目 */
  const handleCreateProject = async () => {
    if (!newProjectName.trim()) return;
    try {
      const api = selectApi();
      await api.createProject(newProjectName.trim(), newProjectPath.trim());
      setNewProjectName("");
      setNewProjectPath("");
      setShowCreateProject(false);
      onRefresh?.();
    } catch {
      // 静默处理
    }
  };

  /** 删除项目 */
  const handleDeleteProject = async (projectID: string) => {
    try {
      const api = selectApi();
      await api.deleteProject(projectID);
      if (activeProjectId === projectID) {
        setActiveProjectId(projects[0]?.id ?? "");
      }
      onRefresh?.();
    } catch {
      // 静默处理
    }
  };

  /** 创建技能组 */
  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) return;
    try {
      const api = selectApi();
      const skillNamesStr = newGroupSelectedSkills.join(",");
      await api.createSkillGroup(newGroupName.trim(), newGroupDesc.trim(), skillNamesStr, newGroupSelectedAgent);
      setNewGroupName("");
      setNewGroupDesc("");
      setNewGroupSelectedSkills([]);
      setNewGroupSelectedAgent("");
      setShowCreateGroup(false);
      await loadSkillGroups();
    } catch {
      // 静默处理
    }
  };

  /** 删除技能组 */
  const handleDeleteGroup = async (groupID: string) => {
    try {
      const api = selectApi();
      await api.deleteSkillGroup(groupID);
      await loadSkillGroups();
    } catch {
      // 静默处理
    }
  };

  /** 绑定代理到项目 */
  const handleBindAgent = async (agentID: string) => {
    if (!activeProject) return;
    try {
      const api = selectApi();
      await api.bindAgentToProject(activeProject.id, agentID);
      setShowBindAgent(false);
      onRefresh?.();
    } catch {
      // 静默处理
    }
  };

  /** 绑定技能组到项目 */
  const handleBindGroup = async (groupName: string) => {
    if (!activeProject) return;
    try {
      const api = selectApi();
      await api.bindSkillGroupToProject(activeProject.id, groupName);
      setShowBindGroup(false);
      onRefresh?.();
    } catch {
      // 静默处理
    }
  };

  /** 为技能组添加技能 */
  const handleAddSkillToGroup = async () => {
    if (!showGroupDetail || !addSkillName.trim()) return;
    try {
      const api = selectApi();
      await api.addSkillToGroup(showGroupDetail.id, addSkillName.trim());
      setAddSkillName("");
      await loadSkillGroups();
      const updated = await selectApi().getSkillGroups();
      const target = updated.find((g) => g.id === showGroupDetail.id);
      if (target) setShowGroupDetail(target);
    } catch {
      // 静默处理
    }
  };

  /** 从技能组移除技能 */
  const handleRemoveSkillFromGroup = async (skillName: string) => {
    if (!showGroupDetail) return;
    try {
      const api = selectApi();
      await api.removeSkillFromGroup(showGroupDetail.id, skillName);
      await loadSkillGroups();
      const updated = await selectApi().getSkillGroups();
      const target = updated.find((g) => g.id === showGroupDetail.id);
      if (target) setShowGroupDetail(target);
    } catch {
      // 静默处理
    }
  };

  /** 刷新项目列表 */
  const handleRefreshProjects = async () => {
    try {
      const api = selectApi();
      await api.refreshProjects();
      onRefresh?.();
    } catch {
      // 静默处理
    }
  };

  /** 切换技能选择 */
  const toggleSkillSelection = (skillName: string) => {
    setNewGroupSelectedSkills((prev) =>
      prev.includes(skillName)
        ? prev.filter((s) => s !== skillName)
        : [...prev, skillName],
    );
  };

  /** 协调项目技能 */
  const handleReconcileProject = async () => {
    if (!activeProject) return;
    try {
      const api = selectApi();
      const plan = await api.reconcileProject(activeProject.id);
      setReconcilePlan(plan);
    } catch {
      // 静默处理
    }
  };

  /** 执行协调计划 */
  const handleExecuteReconcile = async () => {
    if (!activeProject || !reconcilePlan) return;
    setReconcileExecuting(true);
    try {
      const api = selectApi();
      await api.executeReconcilePlan(activeProject.id, JSON.stringify(reconcilePlan));
      setReconcilePlan(null);
      onRefresh?.();
    } catch {
      // 静默处理
    } finally {
      setReconcileExecuting(false);
    }
  };

  const boundAgent = useMemo(() => {
    if (!activeProject?.boundAgentId) return null;
    return agents.find((a) => a.id === activeProject.boundAgentId || a.name === activeProject.boundAgentName);
  }, [activeProject, agents]);

  return (
    <section className="animate-page-in">
      {/* Header */}
      <div className="bg-surface rounded-panel shadow-panel p-8 mb-6">
        <div className="flex items-start justify-between">
          <div>
            <p className="uppercase tracking-widest text-xs text-ink-muted font-body">项目管理</p>
            <h1 className="font-display text-3xl font-semibold text-ink tracking-tight">项目与技能绑定</h1>
            <p className="text-lg text-ink-soft leading-relaxed">管理项目、绑定代理和技能组，为不同项目配置专属工作环境。</p>
          </div>
          <div className="flex gap-4 items-center">
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-ink">{projects.length}</p>
              <p className="text-xs text-ink-muted">项目</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-accent">{skillGroups.length}</p>
              <p className="text-xs text-ink-muted">技能组</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-display font-semibold text-stable-ink">{agents.length}</p>
              <p className="text-xs text-ink-muted">代理</p>
            </div>
          </div>
        </div>
        <div className="flex gap-3 mt-6">
          <button type="button" className="rounded-pill px-5 py-2 text-sm font-medium bg-accent text-white shadow-accent hover:bg-accent-warm transition-colors" onClick={() => setShowCreateProject(true)}>
            新建项目
          </button>
          <button type="button" className="rounded-pill px-5 py-2 text-sm font-medium bg-surface-warm text-ink border border-border hover:bg-surface-hover transition-colors" onClick={() => setShowCreateGroup(true)}>
            创建技能组
          </button>
          <button type="button" className="rounded-pill px-5 py-2 text-sm font-medium bg-surface-warm text-ink border border-border hover:bg-surface-hover transition-colors" onClick={handleRefreshProjects}>
            刷新项目
          </button>
        </div>
      </div>

      {/* Main Content */}
      {projects.length > 0 ? (
        <div className="flex gap-6">
          {/* Project List */}
          <aside className="w-72 shrink-0 bg-surface rounded-panel shadow-panel p-5 flex flex-col gap-3 max-h-[calc(100vh-220px)] overflow-y-auto">
            <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">项目列表 ({projects.length})</p>
            {projects.map((project) => (
              <button
                key={project.id}
                type="button"
                className={`text-left w-full px-4 py-3 rounded-card transition-colors ${project.id === activeProject?.id ? "bg-accent-glow border border-accent/30" : "bg-surface-hover hover:bg-surface-cream border border-transparent"}`}
                onClick={() => setActiveProjectId(project.id)}
              >
                <strong className="block text-sm text-ink">{project.name}</strong>
                <span className="block text-xs text-ink-muted mt-0.5">{project.stage}</span>
                <div className="flex gap-2 mt-1">
                  {project.boundAgentName && (
                    <span className="rounded-chip px-1.5 py-0.5 text-[10px] font-medium bg-accent-glow text-accent">{project.boundAgentName}</span>
                  )}
                  {project.boundSkillGroup && (
                    <span className="rounded-chip px-1.5 py-0.5 text-[10px] font-medium bg-chip text-chip-ink">{project.boundSkillGroup}</span>
                  )}
                </div>
              </button>
            ))}
          </aside>

          {/* Project Detail */}
          <div className="flex-1 flex flex-col gap-5 min-w-0">
            {activeProject ? (
              <>
                <article className="bg-surface rounded-panel shadow-panel p-6">
                  <div className="flex items-center justify-between mb-4">
                    <div>
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">项目详情</p>
                      <h2 className="font-display text-lg font-semibold text-ink">{activeProject.name}</h2>
                      <p className="text-sm text-ink-soft">{activeProject.path || activeProject.summary}</p>
                    </div>
                    <div className="flex gap-2">
                      <StatusBadge tone={activeProject.boundAgentName ? "stable" : "attention"} label={activeProject.boundAgentName ? `代理: ${activeProject.boundAgentName}` : "未绑定代理"} />
                      <StatusBadge tone={activeProject.boundSkillGroup ? "stable" : "attention"} label={activeProject.boundSkillGroup || "未绑定技能组"} />
                    </div>
                  </div>

                  {/* Agent & Skill Group Binding */}
                  <div className="grid grid-cols-2 gap-4 mb-4">
                    <div className="bg-surface-warm rounded-card p-4">
                      <div className="flex items-center justify-between mb-2">
                        <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">使用的代理</p>
                        <button type="button" onClick={() => setShowBindAgent(true)} className="rounded-chip px-2 py-0.5 text-[11px] font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors">
                          {activeProject.boundAgentName ? "更换" : "选择"}
                        </button>
                      </div>
                      {boundAgent ? (
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-card bg-accent/10 flex items-center justify-center text-accent font-display font-semibold">{boundAgent.name.charAt(0)}</div>
                          <div>
                            <p className="text-sm font-medium text-ink">{boundAgent.name}</p>
                            <p className="text-xs text-ink-muted">{boundAgent.installPath}</p>
                          </div>
                          <StatusBadge tone={boundAgent.status === "healthy" ? "stable" : "attention"} label={boundAgent.status === "healthy" ? "正常" : "异常"} size="sm" />
                        </div>
                      ) : (
                        <p className="text-sm text-ink-muted py-2">尚未选择代理，点击"选择"按钮绑定</p>
                      )}
                    </div>

                    <div className="bg-surface-warm rounded-card p-4">
                      <div className="flex items-center justify-between mb-2">
                        <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">技能组</p>
                        <button type="button" onClick={() => setShowBindGroup(true)} className="rounded-chip px-2 py-0.5 text-[11px] font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors">
                          {activeProject.boundSkillGroup ? "更换" : "选择"}
                        </button>
                      </div>
                      {activeProject.boundSkillGroup ? (
                        <div>
                          <p className="text-sm font-medium text-ink">{activeProject.boundSkillGroup}</p>
                          <p className="text-xs text-ink-muted">{activeProject.skillNames?.length ?? 0} 个技能</p>
                          {activeProject.skillNames?.length > 0 && (
                            <div className="flex flex-wrap gap-1 mt-1">
                              {activeProject.skillNames.slice(0, 5).map((s) => (
                                <span key={s} className="rounded-chip px-1.5 py-0.5 text-[10px] bg-chip text-chip-ink">{s}</span>
                              ))}
                              {activeProject.skillNames.length > 5 && <span className="text-[10px] text-ink-muted">+{activeProject.skillNames.length - 5}</span>}
                            </div>
                          )}
                        </div>
                      ) : (
                        <p className="text-sm text-ink-muted py-2">尚未绑定技能组，点击"选择"按钮绑定</p>
                      )}
                    </div>
                  </div>

                  {/* Skill Names */}
                  {activeProject.skillNames?.length > 0 && (
                    <div className="mb-4">
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">项目技能</p>
                      <div className="flex flex-wrap gap-1">
                        {activeProject.skillNames.map((skill) => (
                          <span key={skill} className="rounded-chip px-2.5 py-0.5 text-xs font-medium bg-chip text-chip-ink">{skill}</span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Recent Changes */}
                  {activeProject.recent?.length > 0 && (
                    <div>
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">最近变更</p>
                      {activeProject.recent.slice(0, 5).map((entry) => (
                        <p key={entry} className="text-sm text-ink-soft">{entry}</p>
                      ))}
                    </div>
                  )}

                  {/* Actions */}
                  <div className="flex gap-2 mt-4 pt-4 border-t border-border-soft">
                    {activeProject.path && (
                      <button type="button" onClick={() => selectApi().openInFinder(activeProject.path)} className="rounded-chip px-3 py-1.5 text-xs font-medium bg-surface-warm text-ink hover:bg-surface-hover transition-colors">
                        在 Finder 中打开
                      </button>
                    )}
                    <button type="button" onClick={handleReconcileProject} disabled={!activeProject.boundAgentId || !activeProject.boundSkillGroup} className="rounded-chip px-3 py-1.5 text-xs font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors disabled:opacity-40">
                      协调技能
                    </button>
                    <button type="button" onClick={() => handleDeleteProject(activeProject.id)} className="rounded-chip px-3 py-1.5 text-xs font-medium bg-red-500/10 text-red-600 hover:opacity-80 transition-opacity">
                      删除项目
                    </button>
                  </div>
                </article>

                {/* Skill Groups Overview */}
                {skillGroups.length > 0 && (
                  <article className="bg-surface rounded-panel shadow-panel p-5">
                    <div className="flex items-center justify-between mb-3">
                      <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body">技能组</p>
                      <button type="button" onClick={() => setShowCreateGroup(true)} className="rounded-chip px-2 py-0.5 text-[11px] font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors">
                        新建
                      </button>
                    </div>
                    <div className="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-3">
                      {skillGroups.map((group) => (
                        <div
                          key={group.id}
                          className="bg-surface-warm rounded-card p-3 cursor-pointer hover:bg-surface-hover transition-colors"
                          onClick={() => setShowGroupDetail(group)}
                        >
                          <div className="flex items-center justify-between">
                            <p className="text-sm font-medium text-ink">{group.name}</p>
                            <StatusBadge tone={group.sourceType === "ai-generated" ? "attention" : "stable"} label={group.sourceType === "ai-generated" ? "AI" : "手动"} size="sm" />
                          </div>
                          <p className="text-xs text-ink-muted mt-1">{group.skillCount} 个技能</p>
                          {group.boundAgentName && (
                            <p className="text-xs text-accent mt-0.5">代理: {group.boundAgentName}</p>
                          )}
                          {group.skillNames?.length > 0 && (
                            <div className="flex flex-wrap gap-0.5 mt-1">
                              {group.skillNames.slice(0, 3).map((s) => (
                                <span key={s} className="rounded-chip px-1 py-0.5 text-[9px] bg-chip text-chip-ink">{s}</span>
                              ))}
                              {group.skillNames.length > 3 && <span className="text-[9px] text-ink-muted">+{group.skillNames.length - 3}</span>}
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </article>
                )}
              </>
            ) : null}
          </div>
        </div>
      ) : (
        <EmptyState
          title="暂无项目"
          description="创建新项目或刷新以扫描本地 Git 项目"
          action={{ label: "新建项目", onClick: () => setShowCreateProject(true) }}
        />
      )}

      {/* Create Project Modal */}
      <Modal open={showCreateProject} onClose={() => setShowCreateProject(false)} title="新建项目" subtitle="选择本地目录创建项目">
        <div className="flex flex-col gap-4">
          <div>
            <label className="block text-sm font-medium text-ink mb-1">项目名称 *</label>
            <input type="text" value={newProjectName} onChange={(e) => setNewProjectName(e.target.value)} placeholder="输入项目名称" className="w-full bg-surface rounded-card px-4 py-2 text-sm text-ink placeholder:text-ink-muted border border-border focus:outline-none focus:ring-1 focus:ring-accent" />
          </div>
          <div>
            <label className="block text-sm font-medium text-ink mb-1">项目目录</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={newProjectPath}
                onChange={(e) => setNewProjectPath(e.target.value)}
                placeholder="点击右侧按钮选择目录"
                readOnly
                className="flex-1 bg-surface rounded-card px-4 py-2 text-sm text-ink placeholder:text-ink-muted border border-border focus:outline-none focus:ring-1 focus:ring-accent cursor-default"
              />
              <button
                type="button"
                onClick={handleSelectDirectory}
                className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors whitespace-nowrap"
              >
                选择目录
              </button>
            </div>
            {newProjectPath && (
              <p className="text-xs text-ink-muted mt-1">已选择: {newProjectPath}</p>
            )}
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" onClick={() => setShowCreateProject(false)} className="rounded-pill px-5 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors">取消</button>
            <button type="button" onClick={handleCreateProject} disabled={!newProjectName.trim()} className="rounded-pill px-5 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40">创建</button>
          </div>
        </div>
      </Modal>

      {/* Create Skill Group Modal */}
      <Modal open={showCreateGroup} onClose={() => { setShowCreateGroup(false); setNewGroupSelectedSkills([]); setNewGroupSelectedAgent(""); setSkillSearchQuery(""); }} title="创建技能组" subtitle="新建技能组并选择技能和代理">
        <div className="flex flex-col gap-4">
          <div>
            <label className="block text-sm font-medium text-ink mb-1">名称 *</label>
            <input type="text" value={newGroupName} onChange={(e) => setNewGroupName(e.target.value)} placeholder="输入技能组名称" className="w-full bg-surface rounded-card px-4 py-2 text-sm text-ink placeholder:text-ink-muted border border-border focus:outline-none focus:ring-1 focus:ring-accent" />
          </div>
          <div>
            <label className="block text-sm font-medium text-ink mb-1">描述</label>
            <textarea value={newGroupDesc} onChange={(e) => setNewGroupDesc(e.target.value)} placeholder="输入技能组描述（可选）" className="w-full min-h-[60px] bg-surface rounded-card px-4 py-2 text-sm text-ink placeholder:text-ink-muted border border-border focus:outline-none focus:ring-1 focus:ring-accent resize-y" />
          </div>

          {/* Agent Selection */}
          <div>
            <label className="block text-sm font-medium text-ink mb-1">绑定代理</label>
            {agents.length > 0 ? (
              <div className="flex flex-col gap-1.5 max-h-[120px] overflow-y-auto">
                {agents.map((agent) => (
                  <button
                    key={agent.id}
                    type="button"
                    onClick={() => setNewGroupSelectedAgent(newGroupSelectedAgent === agent.id ? "" : agent.id)}
                    className={`flex items-center gap-2 p-2 rounded-card text-left text-sm transition-colors ${newGroupSelectedAgent === agent.id ? "bg-accent/8 ring-1 ring-accent/30" : "bg-surface hover:bg-surface-hover"}`}
                  >
                    <div className="w-6 h-6 rounded-card bg-accent/10 flex items-center justify-center text-accent text-xs font-semibold shrink-0">{agent.name.charAt(0)}</div>
                    <span className="flex-1 text-ink">{agent.name}</span>
                    <StatusBadge tone={agent.status === "healthy" ? "stable" : "attention"} label={agent.status === "healthy" ? "正常" : "异常"} size="sm" />
                    {newGroupSelectedAgent === agent.id && <span className="text-accent text-sm">✓</span>}
                  </button>
                ))}
              </div>
            ) : (
              <p className="text-sm text-ink-muted py-2">未发现已安装的代理软件</p>
            )}
          </div>

          {/* Skill Selection */}
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="block text-sm font-medium text-ink">选择技能</label>
              {newGroupSelectedSkills.length > 0 && (
                <span className="text-xs text-accent">已选 {newGroupSelectedSkills.length} 个</span>
              )}
            </div>
            <input
              type="text"
              value={skillSearchQuery}
              onChange={(e) => setSkillSearchQuery(e.target.value)}
              placeholder="搜索技能..."
              className="w-full bg-surface rounded-card px-3 py-1.5 text-sm text-ink placeholder:text-ink-muted border border-border focus:outline-none focus:ring-1 focus:ring-accent mb-2"
            />
            {newGroupSelectedSkills.length > 0 && (
              <div className="flex flex-wrap gap-1 mb-2">
                {newGroupSelectedSkills.map((skill) => (
                  <span key={skill} className="rounded-chip px-2 py-0.5 text-xs font-medium bg-accent/10 text-accent flex items-center gap-1">
                    {skill}
                    <button type="button" onClick={() => toggleSkillSelection(skill)} className="text-ink-muted hover:text-red-500 transition-colors">✕</button>
                  </span>
                ))}
              </div>
            )}
            <div className="max-h-[180px] overflow-y-auto flex flex-col gap-1">
              {filteredAvailableSkills.length > 0 ? (
                filteredAvailableSkills.map((skill) => (
                  <button
                    key={skill.name}
                    type="button"
                    onClick={() => toggleSkillSelection(skill.name)}
                    className={`flex items-center gap-2 p-2 rounded-card text-left text-sm transition-colors ${newGroupSelectedSkills.includes(skill.name) ? "bg-accent/8 ring-1 ring-accent/30" : "bg-surface hover:bg-surface-hover"}`}
                  >
                    <span className="flex-1 text-ink">{skill.name}</span>
                    <span className="text-[10px] text-ink-muted">{skill.source}</span>
                    {newGroupSelectedSkills.includes(skill.name) && <span className="text-accent text-sm">✓</span>}
                  </button>
                ))
              ) : (
                <p className="text-sm text-ink-muted py-2 text-center">
                  {availableSkills.length === 0 ? "暂无可选技能，请先安装技能或同步商店" : "未找到匹配的技能"}
                </p>
              )}
            </div>
          </div>

          <div className="flex justify-end gap-2">
            <button type="button" onClick={() => { setShowCreateGroup(false); setNewGroupSelectedSkills([]); setNewGroupSelectedAgent(""); setSkillSearchQuery(""); }} className="rounded-pill px-5 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors">取消</button>
            <button type="button" onClick={handleCreateGroup} disabled={!newGroupName.trim()} className="rounded-pill px-5 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40">创建</button>
          </div>
        </div>
      </Modal>

      {/* Bind Agent Modal */}
      <Modal open={showBindAgent} onClose={() => setShowBindAgent(false)} title="选择代理" subtitle="为项目绑定使用的代理">
        <div className="flex flex-col gap-3">
          {agents.length > 0 ? (
            agents.map((agent) => (
              <button
                key={agent.id}
                type="button"
                onClick={() => handleBindAgent(agent.id)}
                className={`flex items-center gap-3 p-3 rounded-card text-left transition-colors ${activeProject?.boundAgentId === agent.id ? "bg-accent/8 ring-1 ring-accent/30" : "bg-surface hover:bg-surface-hover"}`}
              >
                <div className="w-10 h-10 rounded-card bg-accent/10 flex items-center justify-center text-accent font-display font-semibold shrink-0">{agent.name.charAt(0)}</div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-ink">{agent.name}</p>
                  <p className="text-xs text-ink-muted truncate">{agent.installPath}</p>
                </div>
                <StatusBadge tone={agent.status === "healthy" ? "stable" : "attention"} label={agent.status === "healthy" ? "正常" : "异常"} size="sm" />
                {activeProject?.boundAgentId === agent.id && <span className="text-accent text-sm">✓</span>}
              </button>
            ))
          ) : (
            <p className="text-ink-soft text-center py-4">未发现已安装的代理软件</p>
          )}
        </div>
      </Modal>

      {/* Bind Skill Group Modal */}
      <Modal open={showBindGroup} onClose={() => setShowBindGroup(false)} title="选择技能组" subtitle="为项目绑定技能组">
        <div className="flex flex-col gap-3">
          {skillGroups.length > 0 ? (
            skillGroups.map((group) => (
              <button
                key={group.id}
                type="button"
                onClick={() => handleBindGroup(group.name)}
                className={`flex items-center gap-3 p-3 rounded-card text-left transition-colors ${activeProject?.boundSkillGroup === group.name ? "bg-accent/8 ring-1 ring-accent/30" : "bg-surface hover:bg-surface-hover"}`}
              >
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-ink">{group.name}</p>
                  <p className="text-xs text-ink-muted">{group.description || "无描述"}</p>
                  <p className="text-xs text-ink-muted">{group.skillCount} 个技能</p>
                  {group.boundAgentName && <p className="text-xs text-accent">代理: {group.boundAgentName}</p>}
                </div>
                {activeProject?.boundSkillGroup === group.name && <span className="text-accent text-sm">✓</span>}
              </button>
            ))
          ) : (
            <div className="text-center py-4">
              <p className="text-ink-soft mb-2">暂无技能组</p>
              <button type="button" onClick={() => { setShowBindGroup(false); setShowCreateGroup(true); }} className="rounded-chip px-3 py-1 text-xs font-medium bg-accent-glow text-accent hover:bg-accent hover:text-white transition-colors">
                创建技能组
              </button>
            </div>
          )}
        </div>
      </Modal>

      {/* Skill Group Detail Modal */}
      <Modal open={showGroupDetail !== null} onClose={() => { setShowGroupDetail(null); setAddSkillName(""); }} title={showGroupDetail?.name ?? "技能组详情"} subtitle="技能组管理">
        {showGroupDetail && (
          <div className="flex flex-col gap-4">
            <div className="bg-surface-warm rounded-card p-4">
              <p className="text-sm text-ink-soft">{showGroupDetail.description || "无描述"}</p>
              <div className="flex gap-3 mt-2 text-xs text-ink-muted">
                <span>{showGroupDetail.skillCount} 个技能</span>
                <span>创建于 {showGroupDetail.createdAt || "未知"}</span>
                <StatusBadge tone={showGroupDetail.sourceType === "ai-generated" ? "attention" : "stable"} label={showGroupDetail.sourceType === "ai-generated" ? "AI 生成" : "手动创建"} size="sm" />
              </div>
              {showGroupDetail.boundAgentName && (
                <p className="text-xs text-accent mt-1">绑定代理: {showGroupDetail.boundAgentName}</p>
              )}
            </div>

            <div>
              <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">包含技能</p>
              {showGroupDetail.skillNames?.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {showGroupDetail.skillNames.map((skill) => (
                    <span key={skill} className="rounded-chip px-2.5 py-0.5 text-xs font-medium bg-chip text-chip-ink flex items-center gap-1">
                      {skill}
                      <button type="button" onClick={() => handleRemoveSkillFromGroup(skill)} className="text-ink-muted hover:text-red-500 transition-colors">✕</button>
                    </span>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-ink-muted">暂无技能</p>
              )}
            </div>

            <div className="flex gap-2">
              <input type="text" value={addSkillName} onChange={(e) => setAddSkillName(e.target.value)} placeholder="输入技能名称添加" className="flex-1 bg-surface rounded-card px-3 py-2 text-sm text-ink placeholder:text-ink-muted border border-border focus:outline-none focus:ring-1 focus:ring-accent" />
              <button type="button" onClick={handleAddSkillToGroup} disabled={!addSkillName.trim()} className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40">添加</button>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button type="button" onClick={() => handleDeleteGroup(showGroupDetail.id)} className="rounded-pill px-4 py-2 text-sm font-medium bg-red-500/10 text-red-600 hover:opacity-80 transition-opacity">
                删除技能组
              </button>
              <button type="button" onClick={() => { setShowGroupDetail(null); setAddSkillName(""); }} className="rounded-pill px-4 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors">
                关闭
              </button>
            </div>
          </div>
        )}
      </Modal>

      {/* Reconcile Plan Modal */}
      <Modal open={reconcilePlan !== null} onClose={() => setReconcilePlan(null)} title="技能协调计划" subtitle={`项目: ${reconcilePlan?.projectName ?? ""}`}>
        {reconcilePlan && (
          <div className="flex flex-col gap-4">
            {reconcilePlan.blockReason && (
              <div className="bg-red-500/10 rounded-card p-3">
                <p className="text-sm text-red-600">⚠️ {reconcilePlan.blockReason}</p>
              </div>
            )}

            {reconcilePlan.install.length > 0 && (
              <div>
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">需要安装 ({reconcilePlan.install.length})</p>
                <div className="flex flex-wrap gap-1">
                  {reconcilePlan.install.map((item) => (
                    <span key={item.skillId} className="rounded-chip px-2.5 py-0.5 text-xs font-medium bg-green-500/10 text-green-700">{item.name} (v{item.version})</span>
                  ))}
                </div>
              </div>
            )}

            {reconcilePlan.update.length > 0 && (
              <div>
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">需要更新 ({reconcilePlan.update.length})</p>
                <div className="flex flex-wrap gap-1">
                  {reconcilePlan.update.map((item) => (
                    <span key={item.skillId} className="rounded-chip px-2.5 py-0.5 text-xs font-medium bg-yellow-500/10 text-yellow-700">{item.name} → v{item.version}</span>
                  ))}
                </div>
              </div>
            )}

            {reconcilePlan.repair.length > 0 && (
              <div>
                <p className="uppercase tracking-widest text-[11px] text-ink-muted font-body mb-2">需要修复 ({reconcilePlan.repair.length})</p>
                <div className="flex flex-wrap gap-1">
                  {reconcilePlan.repair.map((item) => (
                    <span key={item.skillId} className="rounded-chip px-2.5 py-0.5 text-xs font-medium bg-red-500/10 text-red-600">{item.name}</span>
                  ))}
                </div>
              </div>
            )}

            {reconcilePlan.install.length === 0 && reconcilePlan.update.length === 0 && reconcilePlan.repair.length === 0 && !reconcilePlan.blockReason && (
              <div className="bg-green-500/10 rounded-card p-4 text-center">
                <p className="text-sm text-green-700">✅ 所有技能已是最新状态，无需协调</p>
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button type="button" onClick={() => setReconcilePlan(null)} className="rounded-pill px-4 py-2 text-sm font-medium bg-surface text-ink border border-border hover:bg-surface-hover transition-colors">
                关闭
              </button>
              {(reconcilePlan.install.length > 0 || reconcilePlan.update.length > 0 || reconcilePlan.repair.length > 0) && !reconcilePlan.blockReason && (
                <button type="button" onClick={handleExecuteReconcile} disabled={reconcileExecuting} className="rounded-pill px-4 py-2 text-sm font-medium bg-accent text-white hover:bg-accent-warm transition-colors disabled:opacity-40">
                  {reconcileExecuting ? "执行中..." : "执行协调"}
                </button>
              )}
            </div>
          </div>
        )}
      </Modal>
    </section>
  );
}

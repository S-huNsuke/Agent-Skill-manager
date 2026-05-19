import { mockSnapshot } from "./mockData";
import type {
  ActivityItem,
  AgentDetailViewModel,
  AgentViewModel,
  AppInfoViewModel,
  AppSnapshot,
  AssistantTaskViewModel,
  AssistantChatMessageViewModel,
  AssistantChatResponseViewModel,
  AutomationSettingsViewModel,
  AISettingsViewModel,
  CatalogSourceViewModel,
  DiagnosticItemViewModel,
  GeneralSettingsViewModel,
  LogEntryViewModel,
  ProjectViewModel,
  RecommendedAction,
  SkillDetailViewModel,
  SkillExplanationViewModel,
  SkillGroupViewModel,
  SkillViewModel,
  StoreItemViewModel,
  SuggestionTemplate,
  SyncResultViewModel,
  SystemHealthStatus,
  TaskHistoryItem,
  ReconcilePlanViewModel,
} from "./types";

export interface FrontendApi {
  getSnapshot(): Promise<AppSnapshot>;
  getDashboard(): Promise<AppSnapshot["dashboard"]>;
  getAgents(): Promise<AgentViewModel[]>;
  getSkills(): Promise<SkillViewModel[]>;
  getStoreItems(): Promise<StoreItemViewModel[]>;
  getProjects(): Promise<ProjectViewModel[]>;
  getAssistantTask(): Promise<AssistantTaskViewModel>;
  getDiagnostics(): Promise<DiagnosticItemViewModel[]>;
  refreshSnapshot(): Promise<AppSnapshot>;
  installSkill(agentID: string, skillName: string, sourcePath: string): Promise<string>;
  uninstallSkill(agentID: string, skillName: string): Promise<string>;
  updateSkill(agentID: string, skillName: string, sourcePath: string): Promise<string>;
  submitGoal(goal: string): Promise<AssistantTaskViewModel>;
  chatAssistant(message: string, history: AssistantChatMessageViewModel[]): Promise<AssistantChatResponseViewModel>;
  advanceAssistantTask(taskID: string, action: string): Promise<AssistantTaskViewModel>;
  resetAssistantTask(): Promise<AssistantTaskViewModel>;
  getAgentSkills(agentID: string): Promise<SkillViewModel[]>;
  openInFinder(path: string): Promise<string>;
  repairAgent(agentID: string): Promise<string>;
  getAgentDetail(agentID: string): Promise<AgentDetailViewModel>;
  explainSkill(agentID: string, skillName: string): Promise<SkillExplanationViewModel>;
  generateSkillExplanation(agentID: string, skillName: string): Promise<string>;
  getRecentActivities(limit: number): Promise<ActivityItem[]>;
  getSystemHealthStatus(): Promise<SystemHealthStatus>;
  getRecommendedActions(): Promise<RecommendedAction[]>;
  getSkillDetail(agentID: string, skillName: string): Promise<SkillDetailViewModel>;
  getCatalogSources(): Promise<CatalogSourceViewModel[]>;
  syncCatalogSource(sourceID: string): Promise<SyncResultViewModel>;
  syncAllSources(): Promise<SyncResultViewModel[]>;
  addCatalogSource(name: string, url: string): Promise<CatalogSourceViewModel>;
  removeCatalogSource(sourceID: string): Promise<string>;
  explainStoreSkill(sourceName: string, skillName: string): Promise<SkillExplanationViewModel>;
  getSkillGroups(): Promise<SkillGroupViewModel[]>;
  createSkillGroup(name: string, description: string, skillNames: string, agentID: string): Promise<SkillGroupViewModel>;
  deleteSkillGroup(groupID: string): Promise<string>;
  addSkillToGroup(groupID: string, skillName: string): Promise<string>;
  removeSkillFromGroup(groupID: string, skillName: string): Promise<string>;
  createProject(name: string, path: string): Promise<ProjectViewModel>;
  deleteProject(projectID: string): Promise<string>;
  bindAgentToProject(projectID: string, agentID: string): Promise<string>;
  bindSkillGroupToProject(projectID: string, groupName: string): Promise<string>;
  refreshProjects(): Promise<ProjectViewModel[]>;
  selectDirectory(title: string): Promise<string>;
  reconcileProject(projectID: string): Promise<ReconcilePlanViewModel>;
  executeReconcilePlan(projectID: string, planJSON: string): Promise<string>;
  getTaskHistory(limit: number): Promise<TaskHistoryItem[]>;
  deleteTaskHistoryItem(taskID: string): Promise<string>;
  getSuggestionTemplates(): Promise<SuggestionTemplate[]>;
  getAppInfoFull(): Promise<AppInfoViewModel>;
  getGeneralSettings(): Promise<GeneralSettingsViewModel>;
  saveGeneralSettings(settings: GeneralSettingsViewModel): Promise<string>;
  getAutomationSettings(): Promise<AutomationSettingsViewModel>;
  saveAutomationSettings(settings: AutomationSettingsViewModel): Promise<string>;
  getAISettings(): Promise<AISettingsViewModel>;
  saveAISettings(settings: AISettingsViewModel): Promise<string>;
  getLogs(level: string, limit: number): Promise<LogEntryViewModel[]>;
  exportDiagnostics(): Promise<string>;
  batchUpdateSkills(agentID: string, skillNames: string): Promise<string>;
  batchUninstallSkills(agentID: string, skillNames: string): Promise<string>;
}

const resolved = <T,>(value: T): Promise<T> => Promise.resolve(structuredClone(value));

export const mockApi: FrontendApi = {
  getSnapshot: () => resolved(mockSnapshot),
  getDashboard: () => resolved(mockSnapshot.dashboard),
  getAgents: () => resolved(mockSnapshot.agents),
  getSkills: () => resolved(mockSnapshot.skills),
  getStoreItems: () => resolved(mockSnapshot.store),
  getProjects: () => resolved(mockSnapshot.projects),
  getAssistantTask: () => resolved(mockSnapshot.assistant),
  getDiagnostics: () => resolved(mockSnapshot.diagnostics),
  refreshSnapshot: () => resolved(mockSnapshot),
  installSkill: () => resolved("ok"),
  uninstallSkill: () => resolved("ok"),
  updateSkill: () => resolved("ok"),
  submitGoal: (goal: string) => resolved({
    id: "assistant-active",
    request: goal,
    status: "planning",
    nextStep: "正在分析目标并规划技能安装步骤",
    summary: `正在为「${goal}」规划技能安装方案。`,
    recommendation: "AI 助手正在分析您的目标，稍后将给出技能推荐。",
    reason: "",
    records: [`用户提交目标: ${goal}`, "状态: 规划中"],
    planJson: "",
    planSteps: [],
    resolvedActions: [],
  }),
  chatAssistant: (message: string) => resolved({
    reply: `我收到你的消息了：${message}`,
    provider: "none",
    model: "",
  }),
  advanceAssistantTask: (taskID: string, action: string) => resolved({
    id: taskID,
    request: "",
    status: action === "cancel" ? "cancelled" : action === "report" ? "completed" : "resolving",
    nextStep: action === "cancel" ? "已取消" : "继续执行中",
    summary: `操作 ${action} 已执行`,
    recommendation: "",
    reason: "",
    records: [],
    planJson: "",
    planSteps: [],
    resolvedActions: [],
  }),
  resetAssistantTask: () => resolved({
    id: "assistant-idle",
    request: "",
    status: "queued",
    nextStep: "等待用户输入目标",
    summary: "AI 助手待命中，输入目标即可开始规划。",
    recommendation: "输入一个目标，让 AI 帮你规划技能安装与修复。",
    reason: "",
    records: [],
    planJson: "",
    planSteps: [],
    resolvedActions: [],
  }),
  getAgentSkills: () => resolved([]),
  openInFinder: () => resolved("ok"),
  repairAgent: () => resolved("ok"),
  getAgentDetail: (agentID: string) => resolved({ id: agentID, displayName: "", found: false, installPath: "", installPaths: [], skillsPath: "", skillsPaths: [], health: "", lastScannedAt: "", lastErrorCode: "", lastErrorMessage: "", skillCount: 0, skillNames: [] }),
  explainSkill: (agentID: string, skillName: string) => resolved({ agentId: agentID, agentName: "", skillName, found: false, skillPath: "", readmeFile: "", readmeContent: "", files: [], aiExplanation: "" }),
  generateSkillExplanation: () => resolved(""),
  getRecentActivities: () => resolved([]),
  getSystemHealthStatus: () => resolved({ overallStatus: "ok", agentHealth: [], diskSpace: { totalGb: 0, freeGb: 0, usedPct: 0 }, checkedAt: "" }),
  getRecommendedActions: () => resolved([]),
  getSkillDetail: (agentID: string, skillName: string) => resolved({ id: `${agentID}-${skillName}`, name: skillName, agentId: agentID, agentName: "", version: "", author: "", description: "", tags: [], installPath: "", installedAt: "", source: "", files: [], projectCount: 0, found: false }),
  getCatalogSources: () => resolved([]),
  syncCatalogSource: (sourceID: string) => resolved({ sourceId: sourceID, success: true, newSkills: 0, updatedSkills: 0, errors: [] }),
  syncAllSources: () => resolved([]),
  addCatalogSource: (name: string, url: string) => resolved({ id: "", name, url, isBuiltin: false, enabled: true, lastSyncedAt: "", lastSyncStatus: "", skillCount: 0 }),
  removeCatalogSource: () => resolved("ok"),
  explainStoreSkill: (_sourceName: string, skillName: string) => resolved({ agentId: "store", agentName: "", skillName, found: true, skillPath: "", readmeFile: "", readmeContent: "这是一个 AI 代理技能，用于增强代理的能力。", files: [], aiExplanation: "这个技能可以帮助 AI 代理完成特定任务，提升代理的能力范围。" }),
  getSkillGroups: () => resolved([]),
  createSkillGroup: (name: string, description: string, skillNames: string, agentID: string) => resolved({ id: "", name, description, sourceType: "manual", skillCount: 0, projectCount: 0, createdAt: "", skillNames: [], boundAgentId: agentID, boundAgentName: "" }),
  deleteSkillGroup: () => resolved("ok"),
  addSkillToGroup: () => resolved("ok"),
  removeSkillFromGroup: () => resolved("ok"),
  createProject: (name: string, path: string) => resolved({ id: "", name, path, stage: "新建", boundSkillGroup: "", boundAgentId: "", boundAgentName: "", skillNames: [], summary: "", needs: [], localAgents: [], recent: [], createdAt: "" }),
  deleteProject: () => resolved("ok"),
  bindAgentToProject: () => resolved("ok"),
  bindSkillGroupToProject: () => resolved("ok"),
  refreshProjects: () => resolved([]),
  selectDirectory: () => resolved(""),
  reconcileProject: () => resolved({ projectId: "", projectName: "", install: [], update: [], repair: [], blockReason: "" }),
  executeReconcilePlan: () => resolved("ok"),
  getTaskHistory: () => resolved([]),
  deleteTaskHistoryItem: () => resolved("ok"),
  getSuggestionTemplates: () => resolved([]),
  getAppInfoFull: () => resolved({ name: "Agent Skills Manager", version: "0.1.0", buildTime: "", goVersion: "", os: "", arch: "" }),
  getGeneralSettings: () => resolved({ theme: "light", fontSize: "medium", notificationsEnabled: true, language: "zh-CN" }),
  saveGeneralSettings: () => resolved("ok"),
  getAutomationSettings: () => resolved({ autoSyncCatalog: true, autoCheckUpdates: true, autoApplySkillGroups: false, healthCheckSchedule: "daily", autoRepair: false }),
  saveAutomationSettings: () => resolved("ok"),
  getAISettings: () => resolved({ provider: "none", model: "", apiKey: "", baseUrl: "" }),
  saveAISettings: () => resolved("ok"),
  getLogs: (_level: string, _limit: number) => resolved([]),
  exportDiagnostics: () => resolved("{}"),
  batchUpdateSkills: () => resolved("ok"),
  batchUninstallSkills: () => resolved("ok"),
};

const bindingPaths = [
  ["app", "App"],
  ["github.com/caojun/agent-skills-manager/internal/app", "App"],
];

function findWailsMethod(methodName: string): ((...args: unknown[]) => Promise<unknown>) | null {
  const w = window as unknown as Record<string, unknown>;
  const go = w.go as Record<string, Record<string, Record<string, (...args: unknown[]) => Promise<unknown>>>> | undefined;
  if (!go) return null;

  for (const [pkgPath, structName] of bindingPaths) {
    const pkg = go[pkgPath];
    if (pkg) {
      const struct = pkg[structName];
      if (struct) {
        const method = struct[methodName];
        if (method && typeof method === "function") {
          return method;
        }
      }
    }
  }
  return null;
}

function hasWailsSnapshotBinding(): boolean {
  return findWailsMethod("GetSnapshot") !== null;
}

async function wailsCall<T>(methodName: string, ...args: unknown[]): Promise<T> {
  const method = findWailsMethod(methodName);
  if (method) {
    return (await method(...args)) as T;
  }

  throw new Error(`Method ${methodName} not available in Wails bindings`);
}

export const wailsApi: FrontendApi = {
  getSnapshot: () => wailsCall<AppSnapshot>("GetSnapshot"),
  getDashboard: () => wailsCall<AppSnapshot["dashboard"]>("GetDashboard"),
  getAgents: () => wailsCall<AgentViewModel[]>("GetAgents"),
  getSkills: () => wailsCall<SkillViewModel[]>("GetSkills"),
  getStoreItems: () => wailsCall<StoreItemViewModel[]>("GetStoreItems"),
  getProjects: () => wailsCall<ProjectViewModel[]>("GetProjects"),
  getAssistantTask: () => wailsCall<AssistantTaskViewModel>("GetAssistantTask"),
  getDiagnostics: () => wailsCall<DiagnosticItemViewModel[]>("GetDiagnostics"),
  refreshSnapshot: () => wailsCall<AppSnapshot>("GetSnapshot"),
  installSkill: (agentID: string, skillName: string, sourcePath: string) => wailsCall<string>("InstallSkill", agentID, skillName, sourcePath),
  uninstallSkill: (agentID: string, skillName: string) => wailsCall<string>("UninstallSkill", agentID, skillName),
  updateSkill: (agentID: string, skillName: string, sourcePath: string) => wailsCall<string>("UpdateSkill", agentID, skillName, sourcePath),
  submitGoal: (goal: string) => wailsCall<AssistantTaskViewModel>("SubmitGoal", goal),
  chatAssistant: (message: string, history: AssistantChatMessageViewModel[]) => wailsCall<AssistantChatResponseViewModel>("ChatAssistant", message, history),
  advanceAssistantTask: (taskID: string, action: string) => wailsCall<AssistantTaskViewModel>("AdvanceAssistantTask", taskID, action),
  resetAssistantTask: () => wailsCall<AssistantTaskViewModel>("ResetAssistantTask"),
  getAgentSkills: (agentID: string) => wailsCall<SkillViewModel[]>("GetAgentSkills", agentID),
  openInFinder: (path: string) => wailsCall<string>("OpenInFinder", path),
  repairAgent: (agentID: string) => wailsCall<string>("RepairAgent", agentID),
  getAgentDetail: (agentID: string) => wailsCall<AgentDetailViewModel>("GetAgentDetail", agentID),
  explainSkill: (agentID: string, skillName: string) => wailsCall<SkillExplanationViewModel>("ExplainSkill", agentID, skillName),
  generateSkillExplanation: (agentID: string, skillName: string) => wailsCall<string>("GenerateSkillExplanation", agentID, skillName),
  getRecentActivities: (limit: number) => wailsCall<ActivityItem[]>("GetRecentActivities", limit),
  getSystemHealthStatus: () => wailsCall<SystemHealthStatus>("GetSystemHealthStatus"),
  getRecommendedActions: () => wailsCall<RecommendedAction[]>("GetRecommendedActions"),
  getSkillDetail: (agentID: string, skillName: string) => wailsCall<SkillDetailViewModel>("GetSkillDetail", agentID, skillName),
  getCatalogSources: () => wailsCall<CatalogSourceViewModel[]>("GetCatalogSources"),
  syncCatalogSource: (sourceID: string) => wailsCall<SyncResultViewModel>("SyncCatalogSource", sourceID),
  syncAllSources: () => wailsCall<SyncResultViewModel[]>("SyncAllSources"),
  addCatalogSource: (name: string, url: string) => wailsCall<CatalogSourceViewModel>("AddCatalogSource", name, url),
  removeCatalogSource: (sourceID: string) => wailsCall<string>("RemoveCatalogSource", sourceID),
  explainStoreSkill: (sourceName: string, skillName: string) => wailsCall<SkillExplanationViewModel>("ExplainStoreSkill", sourceName, skillName),
  getSkillGroups: () => wailsCall<SkillGroupViewModel[]>("GetSkillGroups"),
  createSkillGroup: (name: string, description: string, skillNames: string, agentID: string) => wailsCall<SkillGroupViewModel>("CreateSkillGroup", name, description, skillNames, agentID),
  deleteSkillGroup: (groupID: string) => wailsCall<string>("DeleteSkillGroup", groupID),
  addSkillToGroup: (groupID: string, skillName: string) => wailsCall<string>("AddSkillToGroup", groupID, skillName),
  removeSkillFromGroup: (groupID: string, skillName: string) => wailsCall<string>("RemoveSkillFromGroup", groupID, skillName),
  createProject: (name: string, path: string) => wailsCall<ProjectViewModel>("CreateProject", name, path),
  deleteProject: (projectID: string) => wailsCall<string>("DeleteProject", projectID),
  bindAgentToProject: (projectID: string, agentID: string) => wailsCall<string>("BindAgentToProject", projectID, agentID),
  bindSkillGroupToProject: (projectID: string, groupName: string) => wailsCall<string>("BindSkillGroupToProject", projectID, groupName),
  refreshProjects: () => wailsCall<ProjectViewModel[]>("RefreshProjects"),
  selectDirectory: (title: string) => wailsCall<string>("SelectDirectory", title),
  reconcileProject: (projectID: string) => wailsCall<ReconcilePlanViewModel>("ReconcileProject", projectID),
  executeReconcilePlan: (projectID: string, planJSON: string) => wailsCall<string>("ExecuteReconcilePlan", projectID, planJSON),
  getTaskHistory: (limit: number) => wailsCall<TaskHistoryItem[]>("GetTaskHistory", limit),
  deleteTaskHistoryItem: (taskID: string) => wailsCall<string>("DeleteTaskHistoryItem", taskID),
  getSuggestionTemplates: () => wailsCall<SuggestionTemplate[]>("GetSuggestionTemplates"),
  getAppInfoFull: () => wailsCall<AppInfoViewModel>("GetAppInfoFull"),
  getGeneralSettings: () => wailsCall<GeneralSettingsViewModel>("GetGeneralSettings"),
  saveGeneralSettings: (settings: GeneralSettingsViewModel) => wailsCall<string>("SaveGeneralSettings", settings),
  getAutomationSettings: () => wailsCall<AutomationSettingsViewModel>("GetAutomationSettings"),
  saveAutomationSettings: (settings: AutomationSettingsViewModel) => wailsCall<string>("SaveAutomationSettings", settings),
  getAISettings: () => wailsCall<AISettingsViewModel>("GetAISettings"),
  saveAISettings: (settings: AISettingsViewModel) => wailsCall<string>("SaveAISettings", settings),
  getLogs: (level: string, limit: number) => wailsCall<LogEntryViewModel[]>("GetLogs", level, limit),
  exportDiagnostics: () => wailsCall<string>("ExportDiagnostics"),
  batchUpdateSkills: (agentID: string, skillNames: string) => wailsCall<string>("BatchUpdateSkills", agentID, skillNames),
  batchUninstallSkills: (agentID: string, skillNames: string) => wailsCall<string>("BatchUninstallSkills", agentID, skillNames),
};

export function isRunningInWails(): boolean {
  const w = window as unknown as Record<string, unknown>;
  return typeof w.go === "object" && w.go !== null;
}

export function hasWailsBindings(): boolean {
  return hasWailsSnapshotBinding();
}

export async function waitForApi(timeoutMs = 3000, intervalMs = 50): Promise<FrontendApi> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (hasWailsSnapshotBinding()) {
      return wailsApi;
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }

  return hasWailsSnapshotBinding() ? wailsApi : mockApi;
}

export function selectApi(): FrontendApi {
  return hasWailsSnapshotBinding() ? wailsApi : mockApi;
}

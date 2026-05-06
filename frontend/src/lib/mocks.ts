export type TaskStatus =
  | "queued"
  | "planning"
  | "resolving"
  | "executing"
  | "verifying"
  | "recovering"
  | "completed"
  | "failed"
  | "blocked"
  | "cancelled";

export type StatusTone = "stable" | "attention" | "critical" | "muted";

export interface DashboardHighlight {
  id: string;
  title: string;
  value: string;
  detail: string;
  tone: StatusTone;
  tag: string;
}

export interface DashboardTask {
  id: string;
  title: string;
  status: TaskStatus;
  owner: string;
  time: string;
  detail: string;
}

export interface DashboardViewModel {
  title: string;
  summary: string;
  spotlight: string;
  highlights: DashboardHighlight[];
  tasks: DashboardTask[];
  notes: string[];
}

export interface AgentViewModel {
  id: string;
  name: string;
  mode: string;
  status: "healthy" | "degraded";
  summary: string;
  focus: string;
  installPath: string;
  skills: number;
}

export interface SkillViewModel {
  id: string;
  name: string;
  group: string;
  installedAt: string;
  summary: string;
  statusLabel: string;
  projects: number;
  agent: string;
  healthStatus: "ok" | "warning" | "error";
  healthMessage: string;
}

export interface StoreItemViewModel {
  id: string;
  name: string;
  author: string;
  source: string;
  status: "ready" | "failed" | "review" | "installed" | "available";
  summary: string;
  installs: string;
  impact: string;
  compatibility: string[];
  reason?: string;
  homepage: string;
  localCachePath: string;
}

export interface ProjectViewModel {
  id: string;
  name: string;
  path: string;
  stage: string;
  boundSkillGroup: string;
  boundAgentId: string;
  boundAgentName: string;
  skillNames: string[];
  summary: string;
  needs: string[];
  localAgents: string[];
  recent: string[];
  createdAt: string;
}

export interface AssistantTaskViewModel {
  id: string;
  request: string;
  status: TaskStatus;
  blocker?: string;
  nextStep: string;
  summary: string;
  recommendation: string;
  reason: string;
  records: string[];
}

export interface DiagnosticItemViewModel {
  id: string;
  label: string;
  value: string;
  tone: StatusTone;
}

/** 代理详细信息视图模型 */
export interface AgentDetailViewModel {
  id: string;
  displayName: string;
  found: boolean;
  installPath: string;
  installPaths: string[];
  skillsPath: string;
  skillsPaths: string[];
  health: string;
  lastScannedAt: string;
  lastErrorCode: string;
  lastErrorMessage: string;
  skillCount: number;
  skillNames: string[];
}

/** 技能解释视图模型 */
export interface SkillExplanationViewModel {
  agentId: string;
  agentName: string;
  skillName: string;
  found: boolean;
  skillPath: string;
  readmeFile: string;
  readmeContent: string;
  files: string[];
}

/** 活动记录条目 */
export interface ActivityItem {
  id: string;
  type: string;
  agentId: string;
  skillName?: string;
  projectId?: string;
  time: string;
  status: string;
  detail: string;
}

/** 代理健康条目 */
export interface AgentHealthItem {
  agentId: string;
  name: string;
  status: string;
  detail: string;
}

/** 磁盘空间信息 */
export interface DiskSpaceInfo {
  totalGb: number;
  freeGb: number;
  usedPct: number;
}

/** 系统健康状态 */
export interface SystemHealthStatus {
  overallStatus: string;
  agentHealth: AgentHealthItem[];
  diskSpace: DiskSpaceInfo;
  checkedAt: string;
}

/** 推荐操作条目 */
export interface RecommendedAction {
  id: string;
  priority: string;
  action: string;
  reason: string;
  type: string;
}

/** 技能详情视图模型 */
export interface SkillDetailViewModel {
  id: string;
  name: string;
  agentId: string;
  agentName: string;
  version: string;
  author: string;
  description: string;
  tags: string[];
  installPath: string;
  installedAt: string;
  source: string;
  files: string[];
  projectCount: number;
  found: boolean;
}

/** 商店源视图模型 */
export interface CatalogSourceViewModel {
  id: string;
  name: string;
  url: string;
  isBuiltin: boolean;
  enabled: boolean;
  lastSyncedAt: string;
  lastSyncStatus: string;
  skillCount: number;
}

/** 同步结果视图模型 */
export interface SyncResultViewModel {
  sourceId: string;
  success: boolean;
  newSkills: number;
  updatedSkills: number;
  errors: string[];
}

/** 技能组视图模型 */
export interface SkillGroupViewModel {
  id: string;
  name: string;
  description: string;
  sourceType: string;
  skillCount: number;
  projectCount: number;
  createdAt: string;
  skillNames: string[];
  boundAgentId: string;
  boundAgentName: string;
}

/** 任务历史条目 */
export interface TaskHistoryItem {
  id: string;
  goal: string;
  status: string;
  startedAt: string;
  finishedAt: string;
  summary: string;
}

/** 协调计划视图模型 */
export interface ReconcilePlanViewModel {
  projectId: string;
  projectName: string;
  install: ReconcileActionItem[];
  update: ReconcileActionItem[];
  repair: ReconcileActionItem[];
  blockReason: string;
}

/** 协调操作条目 */
export interface ReconcileActionItem {
  skillId: string;
  version: string;
  name: string;
}

/** 建议提示模板 */
export interface SuggestionTemplate {
  id: string;
  category: string;
  title: string;
  description: string;
  promptTemplate: string;
  isBuiltin: boolean;
}

/** 应用信息视图模型 */
export interface AppInfoViewModel {
  name: string;
  version: string;
  buildTime: string;
  goVersion: string;
  os: string;
  arch: string;
}

/** 通用设置视图模型 */
export interface GeneralSettingsViewModel {
  theme: string;
  fontSize: string;
  notificationsEnabled: boolean;
  language: string;
}

/** 自动化设置视图模型 */
export interface AutomationSettingsViewModel {
  autoSyncCatalog: boolean;
  autoCheckUpdates: boolean;
  autoApplySkillGroups: boolean;
  healthCheckSchedule: string;
  autoRepair: boolean;
}

/** AI 设置视图模型 */
export interface AISettingsViewModel {
  provider: string;
  model: string;
  apiKey: string;
  baseUrl: string;
}

/** 日志条目视图模型 */
export interface LogEntryViewModel {
  level: string;
  message: string;
  timestamp: string;
}

export interface AppSnapshot {
  dashboard: DashboardViewModel;
  agents: AgentViewModel[];
  skills: SkillViewModel[];
  store: StoreItemViewModel[];
  projects: ProjectViewModel[];
  assistant: AssistantTaskViewModel;
  diagnostics: DiagnosticItemViewModel[];
  agentDetail?: AgentDetailViewModel;
}

export const mockSnapshot: AppSnapshot = {
  dashboard: {
    title: "本机技能与运行状态",
    summary: "查看本机 AI 代理、已安装技能和任务执行状态。",
    spotlight: "系统已就绪，等待发现本地代理。",
    highlights: [
      {
        id: "health",
        title: "任务状态",
        value: "0 个任务",
        detail: "暂无运行中的任务。",
        tone: "stable",
        tag: "概况",
      },
    ],
    tasks: [],
    notes: [
      "安装技能前请先查看兼容性信息。",
      "每个项目只能绑定一个技能组。",
    ],
  },
  agents: [],
  skills: [],
  store: [],
  projects: [],
  assistant: {
    id: "assistant-idle",
    request: "",
    status: "queued",
    nextStep: "等待用户输入目标",
    summary: "AI 助手待命中，输入目标即可开始规划。",
    recommendation: "",
    reason: "",
    records: [],
  },
  diagnostics: [
    {
      id: "diag-1",
      label: "Wails 绑定",
      value: "使用示例数据（未连接后端）",
      tone: "attention",
    },
    {
      id: "diag-2",
      label: "前端路由",
      value: "已就绪",
      tone: "stable",
    },
  ],
};

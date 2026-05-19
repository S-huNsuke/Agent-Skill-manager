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
  status: "healthy" | "degraded" | "not_installed";
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
  planJson: string;
  planSteps: AssistantPlanStep[];
  resolvedActions: AssistantResolvedAction[];
}

export interface AssistantPlanStep {
  action: string;
  label: string;
  detail: string;
}

export interface AssistantResolvedAction {
  skillId: string;
  version: string;
  targetAgent: string;
  action: string;
}

export interface AssistantChatMessageViewModel {
  role: "user" | "assistant" | "system";
  content: string;
}

export interface AssistantChatResponseViewModel {
  reply: string;
  provider: string;
  model: string;
  error?: string;
}

export interface DiagnosticItemViewModel {
  id: string;
  label: string;
  value: string;
  tone: StatusTone;
}

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

export interface SkillExplanationViewModel {
  agentId: string;
  agentName: string;
  skillName: string;
  found: boolean;
  skillPath: string;
  readmeFile: string;
  readmeContent: string;
  files: string[];
  aiExplanation: string;
}

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

export interface AgentHealthItem {
  agentId: string;
  name: string;
  status: string;
  detail: string;
}

export interface DiskSpaceInfo {
  totalGb: number;
  freeGb: number;
  usedPct: number;
}

export interface SystemHealthStatus {
  overallStatus: string;
  agentHealth: AgentHealthItem[];
  diskSpace: DiskSpaceInfo;
  checkedAt: string;
}

export interface RecommendedAction {
  id: string;
  priority: string;
  action: string;
  reason: string;
  type: string;
}

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

export interface SyncResultViewModel {
  sourceId: string;
  success: boolean;
  newSkills: number;
  updatedSkills: number;
  errors: string[];
}

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

export interface TaskHistoryItem {
  id: string;
  goal: string;
  status: string;
  startedAt: string;
  finishedAt: string;
  summary: string;
}

export interface ReconcilePlanViewModel {
  projectId: string;
  projectName: string;
  install: ReconcileActionItem[];
  update: ReconcileActionItem[];
  repair: ReconcileActionItem[];
  blockReason: string;
}

export interface ReconcileActionItem {
  skillId: string;
  version: string;
  name: string;
}

export interface SuggestionTemplate {
  id: string;
  category: string;
  title: string;
  description: string;
  promptTemplate: string;
  isBuiltin: boolean;
}

export interface AppInfoViewModel {
  name: string;
  version: string;
  buildTime: string;
  goVersion: string;
  os: string;
  arch: string;
}

export interface GeneralSettingsViewModel {
  theme: string;
  fontSize: string;
  notificationsEnabled: boolean;
  language: string;
}

export interface AutomationSettingsViewModel {
  autoSyncCatalog: boolean;
  autoCheckUpdates: boolean;
  autoApplySkillGroups: boolean;
  healthCheckSchedule: string;
  autoRepair: boolean;
}

export interface AISettingsViewModel {
  provider: "none" | "openai" | "anthropic" | "gemini" | "openai-compatible";
  model: string;
  apiKey: string;
  baseUrl: string;
}

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

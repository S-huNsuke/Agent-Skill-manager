package app

/** 仪表盘高亮条目 */
type DashboardHighlight struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
	Tone   string `json:"tone"`
	Tag    string `json:"tag"`
}

/** 仪表盘任务条目 */
type DashboardTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Owner  string `json:"owner"`
	Time   string `json:"time"`
	Detail string `json:"detail"`
}

/** 仪表盘视图模型 */
type DashboardSnapshot struct {
	Title      string              `json:"title"`
	Summary    string              `json:"summary"`
	Spotlight  string              `json:"spotlight"`
	Highlights []DashboardHighlight `json:"highlights"`
	Tasks      []DashboardTask     `json:"tasks"`
	Notes      []string            `json:"notes"`
}

/** 适配器视图模型 */
type AgentViewModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Focus       string `json:"focus"`
	InstallPath string `json:"installPath"`
	Skills      int    `json:"skills"`
}

/** 代理详细信息视图模型 */
type AgentDetailViewModel struct {
	ID               string   `json:"id"`
	DisplayName      string   `json:"displayName"`
	Found            bool     `json:"found"`
	InstallPath      string   `json:"installPath"`
	InstallPaths     []string `json:"installPaths"`
	SkillsPath       string   `json:"skillsPath"`
	SkillsPaths      []string `json:"skillsPaths"`
	Health           string   `json:"health"`
	LastScannedAt    string   `json:"lastScannedAt"`
	LastErrorCode    string   `json:"lastErrorCode"`
	LastErrorMessage string   `json:"lastErrorMessage"`
	SkillCount       int      `json:"skillCount"`
	SkillNames       []string `json:"skillNames"`
}

/** 技能解释视图模型，用于 AI 分析技能用途 */
type SkillExplanationViewModel struct {
	AgentID       string   `json:"agentId"`
	AgentName     string   `json:"agentName"`
	SkillName     string   `json:"skillName"`
	Found         bool     `json:"found"`
	SkillPath     string   `json:"skillPath"`
	ReadmeFile    string   `json:"readmeFile"`
	ReadmeContent string   `json:"readmeContent"`
	Files         []string `json:"files"`
}

/** 技能视图模型 */
type SkillViewModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Group         string `json:"group"`
	InstalledAt   string `json:"installedAt"`
	Summary       string `json:"summary"`
	StatusLabel   string `json:"statusLabel"`
	Projects      int    `json:"projects"`
	Agent         string `json:"agent"`
	HealthStatus  string `json:"healthStatus"`
	HealthMessage string `json:"healthMessage"`
}

/** 商店条目视图模型 */
type StoreItemViewModel struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Author         string   `json:"author"`
	Source         string   `json:"source"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	Installs       string   `json:"installs"`
	Impact         string   `json:"impact"`
	Compatibility  []string `json:"compatibility"`
	Reason         string   `json:"reason,omitempty"`
	Homepage       string   `json:"homepage"`
	LocalCachePath string   `json:"localCachePath"`
}

/** 项目视图模型 */
type ProjectViewModel struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	Stage           string   `json:"stage"`
	BoundSkillGroup string   `json:"boundSkillGroup"`
	BoundAgentID    string   `json:"boundAgentId"`
	BoundAgentName  string   `json:"boundAgentName"`
	SkillNames      []string `json:"skillNames"`
	Summary         string   `json:"summary"`
	Needs           []string `json:"needs"`
	LocalAgents     []string `json:"localAgents"`
	Recent          []string `json:"recent"`
	CreatedAt       string   `json:"createdAt"`
}

/** 助手任务视图模型 */
type AssistantTaskViewModel struct {
	ID             string   `json:"id"`
	Request        string   `json:"request"`
	Status         string   `json:"status"`
	Blocker        string   `json:"blocker,omitempty"`
	NextStep       string   `json:"nextStep"`
	Summary        string   `json:"summary"`
	Recommendation string   `json:"recommendation"`
	Reason         string   `json:"reason"`
	Records        []string `json:"records"`
}

/** 诊断条目视图模型 */
type DiagnosticItemViewModel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
	Tone  string `json:"tone"`
}

/** 应用快照，包含所有视图模型 */
type AppSnapshot struct {
	Dashboard   DashboardSnapshot       `json:"dashboard"`
	Agents      []AgentViewModel        `json:"agents"`
	Skills      []SkillViewModel        `json:"skills"`
	Store       []StoreItemViewModel    `json:"store"`
	Projects    []ProjectViewModel      `json:"projects"`
	Assistant   AssistantTaskViewModel  `json:"assistant"`
	Diagnostics []DiagnosticItemViewModel `json:"diagnostics"`
}

/** 活动记录条目 */
type ActivityItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	AgentID   string `json:"agentId"`
	SkillName string `json:"skillName,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	Time      string `json:"time"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
}

/** 系统健康状态 */
type SystemHealthStatus struct {
	OverallStatus string            `json:"overallStatus"`
	AgentHealth   []AgentHealthItem `json:"agentHealth"`
	DiskSpace     DiskSpaceInfo     `json:"diskSpace"`
	CheckedAt     string            `json:"checkedAt"`
}

/** 代理健康条目 */
type AgentHealthItem struct {
	AgentID string `json:"agentId"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Detail  string `json:"detail"`
}

/** 磁盘空间信息 */
type DiskSpaceInfo struct {
	TotalGB float64 `json:"totalGb"`
	FreeGB  float64 `json:"freeGb"`
	UsedPct float64 `json:"usedPct"`
}

/** 推荐操作条目 */
type RecommendedAction struct {
	ID       string `json:"id"`
	Priority string `json:"priority"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Type     string `json:"type"`
}

/** 技能详情视图模型 */
type SkillDetailViewModel struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	AgentID      string   `json:"agentId"`
	AgentName    string   `json:"agentName"`
	Version      string   `json:"version"`
	Author       string   `json:"author"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags"`
	InstallPath  string   `json:"installPath"`
	InstalledAt  string   `json:"installedAt"`
	Source       string   `json:"source"`
	Files        []string `json:"files"`
	ProjectCount int      `json:"projectCount"`
	Found        bool     `json:"found"`
}

/** 商店源视图模型 */
type CatalogSourceViewModel struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	URL            string `json:"url"`
	IsBuiltin      bool   `json:"isBuiltin"`
	Enabled        bool   `json:"enabled"`
	LastSyncedAt   string `json:"lastSyncedAt"`
	LastSyncStatus string `json:"lastSyncStatus"`
	SkillCount     int    `json:"skillCount"`
}

/** 同步结果视图模型 */
type SyncResultViewModel struct {
	SourceID      string   `json:"sourceId"`
	Success       bool     `json:"success"`
	NewSkills     int      `json:"newSkills"`
	UpdatedSkills int      `json:"updatedSkills"`
	Errors        []string `json:"errors"`
}

/** 技能组视图模型 */
type SkillGroupViewModel struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	SourceType     string   `json:"sourceType"`
	SkillCount     int      `json:"skillCount"`
	ProjectCount   int      `json:"projectCount"`
	CreatedAt      string   `json:"createdAt"`
	SkillNames     []string `json:"skillNames"`
	BoundAgentID   string   `json:"boundAgentId"`
	BoundAgentName string   `json:"boundAgentName"`
}

/** 任务历史条目 */
type TaskHistoryItem struct {
	ID         string `json:"id"`
	Goal       string `json:"goal"`
	Status     string `json:"status"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
	Summary    string `json:"summary"`
}

/** 建议提示模板 */
type SuggestionTemplate struct {
	ID             string `json:"id"`
	Category       string `json:"category"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	PromptTemplate string `json:"promptTemplate"`
	IsBuiltin      bool   `json:"isBuiltin"`
}

/** 应用信息视图模型 */
type AppInfoViewModel struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

/** 通用设置视图模型 */
type GeneralSettingsViewModel struct {
	Theme                string `json:"theme"`
	FontSize             string `json:"fontSize"`
	NotificationsEnabled bool   `json:"notificationsEnabled"`
	Language             string `json:"language"`
}

/** 自动化设置视图模型 */
type AutomationSettingsViewModel struct {
	AutoSyncCatalog      bool   `json:"autoSyncCatalog"`
	AutoCheckUpdates     bool   `json:"autoCheckUpdates"`
	AutoApplySkillGroups bool   `json:"autoApplySkillGroups"`
	HealthCheckSchedule  string `json:"healthCheckSchedule"`
	AutoRepair           bool   `json:"autoRepair"`
}

/** AI 设置视图模型 */
type AISettingsViewModel struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
	BaseURL  string `json:"baseUrl"`
}

/** 日志条目视图模型 */
type LogEntryViewModel struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

/** 协调计划视图模型 */
type ReconcilePlanViewModel struct {
	ProjectID   string                `json:"projectId"`
	ProjectName string                `json:"projectName"`
	Install     []ReconcileActionItem `json:"install"`
	Update      []ReconcileActionItem `json:"update"`
	Repair      []ReconcileActionItem `json:"repair"`
	BlockReason string                `json:"blockReason"`
}

/** 协调操作条目 */
type ReconcileActionItem struct {
	SkillID string `json:"skillId"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

package domain

import "time"

type Agent struct {
	ID               string
	Kind             string
	Name             string
	Status           string
	InstallPath      string
	SkillsPath       string
	LastSeenAt       *time.Time
	LastErrorCode    string
	LastErrorMessage string
}

type Skill struct {
	ID          string
	Name        string
	Description string
	Tags        []string
}

type CatalogSource struct {
	ID                        string
	Name                      string
	URL                       string
	IsBuiltin                 bool
	Enabled                   bool
	LastSyncedAt              *time.Time
	LastSyncStatus            string
	LastSyncError             string
	CacheExpiresAt            *time.Time
	MinSupportedClientVersion string
}

type CatalogSkill struct {
	ID              string
	SourceID        string
	Name            string
	Version         string
	Author          string
	Description     string
	Homepage        string
	PackageURL      string
	ChecksumSHA256  string
	SupportedAgents []string
	SchemaVersion   string
}

type InstalledSkill struct {
	ID             string
	SkillID        string
	AgentID        string
	Version        string
	InstallState   string
	InstallPath    string
	SourceID       string
	LastCheckedAt  *time.Time
	ErrorMessage   string
	ConflictGroup  string
	IsManagedByApp bool
}

type SkillGroup struct {
	ID              string
	Name            string
	Description     string
	SourceType      string
	Version         string
	GoalPrompt      string
	PreferredAgents []string
	BoundAgentID    string
	BoundAgentName  string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type SkillGroupSkill struct {
	SkillGroupID string
	SkillID      string
	Required     bool
	Priority     int
	Reason       string
}

type Project struct {
	ID               string
	Name             string
	Type             string
	Path             string
	Description      string
	SkillGroupID     string
	AutoApplyEnabled bool
	BoundAgentID     string
	BoundAgentName   string
	LastActiveAt     *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Task struct {
	ID            string
	TaskType      string
	TriggerSource string
	ProjectID     string
	SkillGroupID  string
	Status        string
	StatusReason  string
	PlanJSON      string
	ActionLog     string
	ResultSummary string
	RetryCount    int
	StartedAt     *time.Time
	FinishedAt    *time.Time
}

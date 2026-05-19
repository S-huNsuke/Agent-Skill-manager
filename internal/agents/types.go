package agents

import (
	"context"
	"errors"
	"time"
)

type HealthStatus string

const (
	HealthNotInstalled                  HealthStatus = "not_installed"
	HealthInstalledButUnreadable        HealthStatus = "installed_but_unreadable"
	HealthInstalledButExecutableMissing HealthStatus = "installed_but_executable_missing"
	HealthInstalledButSkillPathMissing  HealthStatus = "installed_but_skill_path_missing"
	HealthInstalledButSkillPathEmpty    HealthStatus = "installed_but_skill_path_empty"
	HealthReady                         HealthStatus = "ready"
	ErrCodeInstallNotFound                           = "install_not_found"
	ErrCodeInstallUnreadable                         = "install_unreadable"
	ErrCodeExecutableNotFound                        = "executable_not_found"
	ErrCodeSkillPathMissing                          = "skill_path_missing"
	ErrCodeSkillPathEmpty                            = "skill_path_empty"
	ErrCodeNotImplemented                            = "not_implemented"
)

var ErrNotImplemented = errors.New("not implemented")

type AgentInstall struct {
	AgentID          string
	DisplayName      string
	InstallPath      string
	SkillsPath       string
	LastScannedAt    time.Time
	Health           HealthStatus
	LastErrorCode    string
	LastErrorMessage string
}

type SkillMutation struct {
	Name       string
	SourcePath string
	Version    string
}

type Adapter interface {
	ID() string
	SkillsRelativePath() string
	Discover(context.Context) (AgentInstall, error)
	DiscoverAll(context.Context) ([]AgentInstall, error)
	ListInstalledSkills(context.Context, AgentInstall) ([]string, error)
	InstallSkill(context.Context, AgentInstall, SkillMutation) error
	UninstallSkill(context.Context, AgentInstall, string) error
	UpdateSkill(context.Context, AgentInstall, SkillMutation) error
	ValidateSkillInstall(context.Context, AgentInstall, string) error
}

type LocalAdapterConfig struct {
	AgentID             string
	DisplayName         string
	DefaultInstallPaths []string
	OverrideInstallPath string
	SkillsRelativePath  string
	ExecutableNames     []string
	LookPath            func(string) (string, error)
	Now                 func() time.Time
}

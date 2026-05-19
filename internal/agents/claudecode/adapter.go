package claudecode

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
)

type Config struct {
	DefaultInstallPaths []string
	OverrideInstallPath string
	SkillsRelativePath  string
	LookPath            func(string) (string, error)
	Now                 func() time.Time
}

func NewAdapter(config Config) agents.Adapter {
	return agents.NewFilesystemAdapter(agents.LocalAdapterConfig{
		AgentID:             "claudecode",
		DisplayName:         "Claude Code",
		DefaultInstallPaths: withDefaultInstallPaths(config.DefaultInstallPaths, ".claude"),
		OverrideInstallPath: config.OverrideInstallPath,
		SkillsRelativePath:  withDefault(config.SkillsRelativePath, "skills"),
		ExecutableNames:     []string{"claude"},
		LookPath:            withLookPath(config.LookPath),
		Now:                 config.Now,
	})
}

func withDefaultInstallPaths(paths []string, dir string) []string {
	if paths != nil {
		return paths
	}
	return []string{defaultInstallPath(dir)}
}

func defaultInstallPath(dir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, dir)
}

func withDefault(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func withLookPath(lookPath func(string) (string, error)) func(string) (string, error) {
	if lookPath != nil {
		return lookPath
	}
	return exec.LookPath
}

package codex

import (
	"os"
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
		AgentID:             "codex",
		DisplayName:         "Codex",
		DefaultInstallPaths: withDefaultInstallPaths(config.DefaultInstallPaths, []string{".codex", ".codex/superpowers"}),
		OverrideInstallPath: config.OverrideInstallPath,
		SkillsRelativePath:  withDefault(config.SkillsRelativePath, "skills"),
		ExecutableNames:     []string{"codex"},
		LookPath:            withLookPath(config.LookPath),
		Now:                 config.Now,
	})
}

func withDefaultInstallPaths(paths []string, dirs []string) []string {
	if paths != nil {
		return paths
	}
	result := make([]string, 0, len(dirs))
	home, err := os.UserHomeDir()
	if err != nil {
		return result
	}
	for _, dir := range dirs {
		result = append(result, filepath.Join(home, dir))
	}
	return result
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
	return func(file string) (string, error) {
		return file, os.ErrNotExist
	}
}

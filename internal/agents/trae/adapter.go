package trae

import (
	"os"
	"path/filepath"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
)

// Config 存储 Trae 适配器的配置信息。
type Config struct {
	DefaultInstallPaths []string
	OverrideInstallPath string
	SkillsRelativePath  string
	LookPath            func(string) (string, error)
	Now                 func() time.Time
}

// NewAdapter 根据给定配置创建 Trae 适配器实例。
func NewAdapter(config Config) agents.Adapter {
	return agents.NewFilesystemAdapter(agents.LocalAdapterConfig{
		AgentID:             "trae",
		DisplayName:         "Trae",
		DefaultInstallPaths: withDefaultInstallPaths(config.DefaultInstallPaths, ".trae"),
		OverrideInstallPath: config.OverrideInstallPath,
		SkillsRelativePath:  withDefault(config.SkillsRelativePath, "skills"),
		ExecutableNames:     []string{"trae"},
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
	return func(file string) (string, error) {
		return file, os.ErrNotExist
	}
}

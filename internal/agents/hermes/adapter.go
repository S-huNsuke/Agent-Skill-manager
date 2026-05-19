package hermes

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

/** 创建 Hermes 代理适配器，同时扫描 ~/.hermes 和 ~/.agents 两个安装路径 */
func NewAdapter(config Config) agents.Adapter {
	return agents.NewFilesystemAdapter(agents.LocalAdapterConfig{
		AgentID:             "hermes",
		DisplayName:         "Hermes",
		DefaultInstallPaths: withDefaultInstallPaths(config.DefaultInstallPaths),
		OverrideInstallPath: config.OverrideInstallPath,
		SkillsRelativePath:  withDefault(config.SkillsRelativePath, "skills"),
		ExecutableNames:     []string{"hermes"},
		LookPath:            withLookPath(config.LookPath),
		Now:                 config.Now,
	})
}

func withDefaultInstallPaths(paths []string) []string {
	if paths != nil {
		return paths
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}
	return []string{
		filepath.Join(home, ".hermes"),
		filepath.Join(home, ".agents"),
	}
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

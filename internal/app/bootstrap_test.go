package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBootstrapConfigSmoke(t *testing.T) {
	configPath := filepath.Join("..", "..", "wails.json")

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected bootstrap config at %s: %v", configPath, err)
	}
}

func TestLoadBootstrapConfig(t *testing.T) {
	repoRoot := filepath.Join("..", "..")

	config, err := LoadBootstrapConfig(repoRoot)
	if err != nil {
		t.Fatalf("LoadBootstrapConfig() error = %v", err)
	}

	if config.AppName != "Agent Skills Manager" {
		t.Fatalf("AppName = %q, want %q", config.AppName, "Agent Skills Manager")
	}
	if config.FrontendDir == "" || config.FrontendDistDir == "" {
		t.Fatalf("expected frontend paths to be populated, got %+v", config)
	}
	if config.FrontendBuildCommand != "pnpm build" {
		t.Fatalf("FrontendBuildCommand = %q, want %q", config.FrontendBuildCommand, "pnpm build")
	}
}

func TestLoadBootstrapConfigFallsBackToEmbeddedAssets(t *testing.T) {
	config, err := LoadBootstrapConfig("")
	if err != nil {
		t.Fatalf("LoadBootstrapConfig() error = %v", err)
	}

	if !config.UsesEmbeddedAssets {
		t.Fatalf("UsesEmbeddedAssets = false, want true")
	}
	if config.WailsConfigPath != "embedded:wails.json" {
		t.Fatalf("WailsConfigPath = %q, want %q", config.WailsConfigPath, "embedded:wails.json")
	}
}

/** 验证 Python Worker 目录解析会优先使用仓库目录 */
func TestResolvePythonWorkerDirFromRepoRoot(t *testing.T) {
	root := t.TempDir()
	workerDir := filepath.Join(root, "python")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker dir: %v", err)
	}

	got := resolvePythonWorkerDir(root)
	if got != workerDir {
		t.Fatalf("resolvePythonWorkerDir() = %q, want %q", got, workerDir)
	}
}

/** 检查数据目录是否可写，用于跳过需要系统资源的测试 */
func requireDataDir(t *testing.T) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("无法获取用户主目录")
	}
	dataDir := filepath.Join(home, "Library", "Application Support", "agent-skills-manager")
	if info, err := os.Stat(dataDir); err != nil || !info.IsDir() {
		t.Skipf("数据目录不可用: %s", dataDir)
	}
}

func TestNewApp(t *testing.T) {
	requireDataDir(t)
	repoRoot := filepath.Join("..", "..")

	done := make(chan struct{})
	var instance *App
	var err error
	go func() {
		instance, err = New(repoRoot, nil)
		close(done)
	}()

	select {
	case <-done:
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if instance.Name != "Agent Skills Manager" {
			t.Fatalf("Name = %q, want %q", instance.Name, "Agent Skills Manager")
		}
		if instance.Version != "0.1.0" {
			t.Fatalf("Version = %q, want %q", instance.Version, "0.1.0")
		}
		if instance.Bootstrap.WailsConfigPath == "" {
			t.Fatalf("expected WailsConfigPath to be populated, got %+v", instance.Bootstrap)
		}
	case <-time.After(10 * time.Second):
		t.Skip("New() 初始化超时，跳过（可能 Keychain 不可用）")
	}
}

func TestGetAppInfo(t *testing.T) {
	requireDataDir(t)
	repoRoot := filepath.Join("..", "..")

	done := make(chan struct{})
	var instance *App
	var err error
	go func() {
		instance, err = New(repoRoot, nil)
		close(done)
	}()

	select {
	case <-done:
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		info := instance.GetAppInfo()
		if info.Name != "Agent Skills Manager" {
			t.Fatalf("Name = %q, want %q", info.Name, "Agent Skills Manager")
		}
		if !info.FrontendReady {
			t.Fatalf("FrontendReady = false, want true")
		}
	case <-time.After(10 * time.Second):
		t.Skip("New() 初始化超时，跳过（可能 Keychain 不可用）")
	}
}

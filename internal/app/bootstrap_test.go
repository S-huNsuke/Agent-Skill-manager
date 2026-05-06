package app

import (
	"os"
	"path/filepath"
	"testing"
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

func TestNewApp(t *testing.T) {
	repoRoot := filepath.Join("..", "..")

	instance, err := New(repoRoot, nil)
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
}

func TestGetAppInfo(t *testing.T) {
	repoRoot := filepath.Join("..", "..")

	instance, err := New(repoRoot, nil)
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
}

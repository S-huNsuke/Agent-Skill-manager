package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	appassets "github.com/caojun/agent-skills-manager"
	platformerrors "github.com/caojun/agent-skills-manager/internal/platform/errors"
)

type BootstrapConfig struct {
	AppName                string
	RepoRoot               string
	WailsConfigPath        string
	FrontendDir            string
	FrontendDistDir        string
	FrontendInstallCommand string
	FrontendBuildCommand   string
	FrontendDevCommand     string
	UsesEmbeddedAssets     bool
}

type wailsConfig struct {
	Name            string `json:"name"`
	FrontendInstall string `json:"frontend:install"`
	FrontendBuild   string `json:"frontend:build"`
	FrontendDev     string `json:"frontend:dev:watcher"`
}

func LoadBootstrapConfig(repoRoot string) (BootstrapConfig, error) {
	if repoRoot != "" {
		config, err := loadDiskBootstrapConfig(repoRoot)
		if err == nil {
			return config, nil
		}
	}

	var parsed wailsConfig
	if err := json.Unmarshal(appassets.WailsConfig, &parsed); err != nil {
		return BootstrapConfig{}, platformerrors.Wrap("decode embedded wails config", err)
	}

	return BootstrapConfig{
		AppName:                defaultAppName,
		WailsConfigPath:        "embedded:wails.json",
		FrontendDir:            "embedded:frontend",
		FrontendDistDir:        "embedded:frontend/dist",
		FrontendInstallCommand: parsed.FrontendInstall,
		FrontendBuildCommand:   parsed.FrontendBuild,
		FrontendDevCommand:     parsed.FrontendDev,
		UsesEmbeddedAssets:     true,
	}, nil
}

func loadDiskBootstrapConfig(repoRoot string) (BootstrapConfig, error) {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return BootstrapConfig{}, platformerrors.Wrap("resolve repo root", err)
	}

	wailsPath := filepath.Join(absRoot, "wails.json")
	payload, err := os.ReadFile(wailsPath)
	if err != nil {
		return BootstrapConfig{}, platformerrors.Wrap("read wails config", err)
	}

	var parsed wailsConfig
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return BootstrapConfig{}, platformerrors.Wrap("decode wails config", err)
	}

	frontendDir := filepath.Join(absRoot, "frontend")
	if _, err := os.Stat(frontendDir); err != nil {
		return BootstrapConfig{}, platformerrors.Wrap("stat frontend dir", err)
	}

	frontendDistDir := filepath.Join(frontendDir, "dist")

	return BootstrapConfig{
		AppName:                defaultAppName,
		RepoRoot:               absRoot,
		WailsConfigPath:        wailsPath,
		FrontendDir:            frontendDir,
		FrontendDistDir:        frontendDistDir,
		FrontendInstallCommand: parsed.FrontendInstall,
		FrontendBuildCommand:   parsed.FrontendBuild,
		FrontendDevCommand:     parsed.FrontendDev,
	}, nil
}

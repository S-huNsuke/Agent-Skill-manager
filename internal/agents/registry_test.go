package agents_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
	"github.com/caojun/agent-skills-manager/internal/agents/claudecode"
	"github.com/caojun/agent-skills-manager/internal/agents/codex"
	"github.com/caojun/agent-skills-manager/internal/agents/geminicli"
	"github.com/caojun/agent-skills-manager/internal/agents/openclaw"
)

func TestRegistryDiscoverAllReturnsReadyAndDegradedAgents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	mustMkdirAll(t, filepath.Join(codexHome, "skills"))
	mustWriteFile(t, filepath.Join(codexHome, "skills", "alpha.md"), "alpha")

	claudeHome := filepath.Join(root, "claudecode")
	mustMkdirAll(t, claudeHome)

	geminiHome := filepath.Join(root, "gemini")
	mustMkdirAll(t, filepath.Join(geminiHome, "skills"))

	registry := agents.NewRegistry(
		codex.NewAdapter(codex.Config{
			DefaultInstallPaths: []string{},
			Now:                 func() time.Time { return now },
			OverrideInstallPath: codexHome,
			SkillsRelativePath:  "skills",
		}),
		claudecode.NewAdapter(claudecode.Config{
			DefaultInstallPaths: []string{},
			Now:                 func() time.Time { return now },
			OverrideInstallPath: claudeHome,
			SkillsRelativePath:  "skills",
		}),
		geminicli.NewAdapter(geminicli.Config{
			DefaultInstallPaths: []string{},
			Now:                 func() time.Time { return now },
			OverrideInstallPath: geminiHome,
			SkillsRelativePath:  "skills",
		}),
		openclaw.NewAdapter(openclaw.Config{
			DefaultInstallPaths: []string{},
			Now:                 func() time.Time { return now },
			OverrideInstallPath: filepath.Join(root, "openclaw"),
			SkillsRelativePath:  "skills",
		}),
	)

	installs, err := registry.DiscoverAll(context.Background())
	if err != nil {
		t.Fatalf("DiscoverAll() error = %v", err)
	}

	if len(installs) != 4 {
		t.Fatalf("DiscoverAll() returned %d installs, want 4", len(installs))
	}

	byID := make(map[string]agents.AgentInstall, len(installs))
	for _, install := range installs {
		byID[install.AgentID] = install
	}

	assertInstallState(t, byID["codex"], agents.HealthReady, codexHome, filepath.Join(codexHome, "skills"), now, "")
	assertInstallState(t, byID["claudecode"], agents.HealthInstalledButSkillPathMissing, claudeHome, filepath.Join(claudeHome, "skills"), now, agents.ErrCodeSkillPathMissing)
	assertInstallState(t, byID["geminicli"], agents.HealthInstalledButSkillPathEmpty, geminiHome, filepath.Join(geminiHome, "skills"), now, agents.ErrCodeSkillPathEmpty)
	assertInstallState(t, byID["openclaw"], agents.HealthNotInstalled, "", "", now, agents.ErrCodeInstallNotFound)
}

func TestRegistryDiscoverPrefersHealthyOverrideOverDegradedDefault(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	defaultHome := filepath.Join(root, "default-codex")
	mustMkdirAll(t, defaultHome)

	overrideHome := filepath.Join(root, "override-codex")
	mustMkdirAll(t, filepath.Join(overrideHome, "skills"))
	mustWriteFile(t, filepath.Join(overrideHome, "skills", "healthy.md"), "healthy")

	adapter := codex.NewAdapter(codex.Config{
		DefaultInstallPaths: []string{defaultHome},
		Now:                 func() time.Time { return now },
		OverrideInstallPath: overrideHome,
		SkillsRelativePath:  "skills",
	})

	install, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	assertInstallState(t, install, agents.HealthReady, overrideHome, filepath.Join(overrideHome, "skills"), now, "")
}

func TestRegistryDiscoverPrefersOverrideWhenDefaultAndOverrideAreBothReady(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	defaultHome := filepath.Join(root, "default-codex")
	mustMkdirAll(t, filepath.Join(defaultHome, "skills"))
	mustWriteFile(t, filepath.Join(defaultHome, "skills", "default.md"), "default")

	overrideHome := filepath.Join(root, "override-codex")
	mustMkdirAll(t, filepath.Join(overrideHome, "skills"))
	mustWriteFile(t, filepath.Join(overrideHome, "skills", "override.md"), "override")

	adapter := codex.NewAdapter(codex.Config{
		DefaultInstallPaths: []string{defaultHome},
		Now:                 func() time.Time { return now },
		OverrideInstallPath: overrideHome,
		SkillsRelativePath:  "skills",
	})

	install, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	assertInstallState(t, install, agents.HealthReady, overrideHome, filepath.Join(overrideHome, "skills"), now, "")
}

func TestRegistryDiscoverPrefersOverrideWhenDefaultAndOverrideShareDegradedRank(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	defaultHome := filepath.Join(root, "default-codex")
	mustMkdirAll(t, defaultHome)

	overrideHome := filepath.Join(root, "override-codex")
	mustMkdirAll(t, overrideHome)

	adapter := codex.NewAdapter(codex.Config{
		DefaultInstallPaths: []string{defaultHome},
		Now:                 func() time.Time { return now },
		OverrideInstallPath: overrideHome,
		SkillsRelativePath:  "skills",
	})

	install, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	assertInstallState(t, install, agents.HealthInstalledButSkillPathMissing, overrideHome, filepath.Join(overrideHome, "skills"), now, agents.ErrCodeSkillPathMissing)
}

func TestRegistryDiscoverDoesNotTreatExecutableDirectoryAsInstallRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	binDir := filepath.Join(root, "bin")
	mustMkdirAll(t, binDir)
	mustWriteFile(t, filepath.Join(binDir, "codex"), "#!/bin/sh\n")

	adapter := codex.NewAdapter(codex.Config{
		DefaultInstallPaths: []string{},
		Now:                 func() time.Time { return now },
		LookPath: func(string) (string, error) {
			return filepath.Join(binDir, "codex"), nil
		},
		SkillsRelativePath: "skills",
	})

	install, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	assertInstallState(t, install, agents.HealthNotInstalled, "", "", now, agents.ErrCodeInstallNotFound)
}

func TestRegistryListInstalledSkillsReturnsSkillsForReadyAgent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	mustMkdirAll(t, filepath.Join(codexHome, "skills"))
	mustWriteFile(t, filepath.Join(codexHome, "skills", "alpha.md"), "alpha")
	mustWriteFile(t, filepath.Join(codexHome, "skills", "beta.txt"), "beta")

	adapter := codex.NewAdapter(codex.Config{
		DefaultInstallPaths: []string{},
		Now:                 func() time.Time { return now },
		OverrideInstallPath: codexHome,
		SkillsRelativePath:  "skills",
	})

	install, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	skills, err := adapter.ListInstalledSkills(context.Background(), install)
	if err != nil {
		t.Fatalf("ListInstalledSkills() error = %v", err)
	}

	if len(skills) != 2 || skills[0] != "alpha.md" || skills[1] != "beta.txt" {
		t.Fatalf("ListInstalledSkills() = %#v, want [alpha.md beta.txt]", skills)
	}
}

func TestRegistryDiscoverIgnoresNoiseWhenDeterminingReadyState(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	mustMkdirAll(t, filepath.Join(codexHome, "skills"))
	mustWriteFile(t, filepath.Join(codexHome, "skills", ".DS_Store"), "junk")
	mustWriteFile(t, filepath.Join(codexHome, "skills", "scratch.tmp"), "junk")

	adapter := codex.NewAdapter(codex.Config{
		DefaultInstallPaths: []string{},
		Now:                 func() time.Time { return now },
		OverrideInstallPath: codexHome,
		SkillsRelativePath:  "skills",
	})

	install, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	assertInstallState(t, install, agents.HealthInstalledButSkillPathEmpty, codexHome, filepath.Join(codexHome, "skills"), now, agents.ErrCodeSkillPathEmpty)
}

func TestRegistryListInstalledSkillsIncludesDirectoryBundles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	mustMkdirAll(t, filepath.Join(codexHome, "skills", "bundle-skill"))
	mustWriteFile(t, filepath.Join(codexHome, "skills", "single.md"), "single")

	adapter := codex.NewAdapter(codex.Config{
		DefaultInstallPaths: []string{},
		Now:                 func() time.Time { return now },
		OverrideInstallPath: codexHome,
		SkillsRelativePath:  "skills",
	})

	install, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	skills, err := adapter.ListInstalledSkills(context.Background(), install)
	if err != nil {
		t.Fatalf("ListInstalledSkills() error = %v", err)
	}

	if len(skills) != 2 || skills[0] != "bundle-skill" || skills[1] != "single.md" {
		t.Fatalf("ListInstalledSkills() = %#v, want [bundle-skill single.md]", skills)
	}
}

func TestRegistryListInstalledSkillsFiltersNoiseEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 2, 21, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	mustMkdirAll(t, filepath.Join(codexHome, "skills", "bundle-skill"))
	mustWriteFile(t, filepath.Join(codexHome, "skills", "single.md"), "single")
	mustWriteFile(t, filepath.Join(codexHome, "skills", ".DS_Store"), "junk")
	mustWriteFile(t, filepath.Join(codexHome, "skills", "scratch.tmp"), "junk")

	adapter := codex.NewAdapter(codex.Config{
		DefaultInstallPaths: []string{},
		Now:                 func() time.Time { return now },
		OverrideInstallPath: codexHome,
		SkillsRelativePath:  "skills",
	})

	install, err := adapter.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	skills, err := adapter.ListInstalledSkills(context.Background(), install)
	if err != nil {
		t.Fatalf("ListInstalledSkills() error = %v", err)
	}

	if len(skills) != 2 || skills[0] != "bundle-skill" || skills[1] != "single.md" {
		t.Fatalf("ListInstalledSkills() = %#v, want [bundle-skill single.md]", skills)
	}
}

func assertInstallState(
	t *testing.T,
	install agents.AgentInstall,
	wantHealth agents.HealthStatus,
	wantInstallPath string,
	wantSkillsPath string,
	wantScannedAt time.Time,
	wantErrCode string,
) {
	t.Helper()

	if install.Health != wantHealth {
		t.Fatalf("%s health = %q, want %q", install.AgentID, install.Health, wantHealth)
	}

	if install.InstallPath != wantInstallPath {
		t.Fatalf("%s install path = %q, want %q", install.AgentID, install.InstallPath, wantInstallPath)
	}

	if install.SkillsPath != wantSkillsPath {
		t.Fatalf("%s skills path = %q, want %q", install.AgentID, install.SkillsPath, wantSkillsPath)
	}

	if !install.LastScannedAt.Equal(wantScannedAt) {
		t.Fatalf("%s last scanned at = %s, want %s", install.AgentID, install.LastScannedAt, wantScannedAt)
	}

	if install.LastErrorCode != wantErrCode {
		t.Fatalf("%s last error code = %q, want %q", install.AgentID, install.LastErrorCode, wantErrCode)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

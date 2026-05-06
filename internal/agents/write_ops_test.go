package agents_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
	"github.com/caojun/agent-skills-manager/internal/agents/codex"
)

/** 验证 InstallSkill 将技能文件复制到适配器技能目录并写入所有权标记 */
func TestInstallSkillWritesToAgentSkillsPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	mustMkdirAll(t, filepath.Join(codexHome, "skills"))
	mustWriteFile(t, filepath.Join(codexHome, "skills", "existing.md"), "existing")

	sourceDir := filepath.Join(root, "source", "my-skill")
	mustMkdirAll(t, sourceDir)
	mustWriteFile(t, filepath.Join(sourceDir, "prompt.md"), "hello")

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

	mutation := agents.SkillMutation{
		Name:       "my-skill",
		SourcePath: sourceDir,
	}

	if err := adapter.InstallSkill(context.Background(), install, mutation); err != nil {
		t.Fatalf("InstallSkill() error = %v", err)
	}

	installedFile := filepath.Join(codexHome, "skills", "my-skill", "prompt.md")
	data, err := os.ReadFile(installedFile)
	if err != nil {
		t.Fatalf("read installed file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("installed file content = %q, want %q", string(data), "hello")
	}

	markerPath := filepath.Join(codexHome, "skills", "my-skill", ".asm-managed")
	markerData, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("read ownership marker: %v", err)
	}

	var marker map[string]interface{}
	if err := json.Unmarshal(markerData, &marker); err != nil {
		t.Fatalf("parse ownership marker: %v", err)
	}
	if marker["skill_name"] != "my-skill" {
		t.Fatalf("marker skill_name = %v, want my-skill", marker["skill_name"])
	}
	if marker["managed_by"] != "agent-skills-manager" {
		t.Fatalf("marker managed_by = %v, want agent-skills-manager", marker["managed_by"])
	}
}

/** 验证 InstallSkill 拒绝覆盖非应用管理的已有技能 */
func TestInstallSkillRejectsOverwriteOfUnmanagedSkill(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	mustMkdirAll(t, filepath.Join(codexHome, "skills", "existing-skill"))
	mustWriteFile(t, filepath.Join(codexHome, "skills", "existing-skill", "prompt.md"), "old")

	sourceDir := filepath.Join(root, "source", "existing-skill")
	mustMkdirAll(t, sourceDir)
	mustWriteFile(t, filepath.Join(sourceDir, "prompt.md"), "new")

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

	mutation := agents.SkillMutation{
		Name:       "existing-skill",
		SourcePath: sourceDir,
	}

	err = adapter.InstallSkill(context.Background(), install, mutation)
	if err == nil {
		t.Fatal("InstallSkill() should reject overwrite of unmanaged skill")
	}
}

/** 验证 UninstallSkill 删除应用管理的技能及其所有权标记 */
func TestUninstallSkillRemovesManagedSkill(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	skillDir := filepath.Join(codexHome, "skills", "my-skill")
	mustMkdirAll(t, skillDir)
	mustWriteFile(t, filepath.Join(skillDir, "prompt.md"), "hello")

	marker := map[string]interface{}{
		"skill_name":  "my-skill",
		"version":     "1.0.0",
		"managed_by":  "agent-skills-manager",
		"installed_at": now.Format(time.RFC3339),
	}
	markerJSON, _ := json.Marshal(marker)
	mustWriteFile(t, filepath.Join(skillDir, ".asm-managed"), string(markerJSON))

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

	if err := adapter.UninstallSkill(context.Background(), install, "my-skill"); err != nil {
		t.Fatalf("UninstallSkill() error = %v", err)
	}

	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatal("skill directory should be removed after uninstall")
	}
}

/** 验证 UninstallSkill 拒绝删除非应用管理的技能 */
func TestUninstallSkillRejectsUnmanagedSkill(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	skillDir := filepath.Join(codexHome, "skills", "unmanaged-skill")
	mustMkdirAll(t, skillDir)
	mustWriteFile(t, filepath.Join(skillDir, "prompt.md"), "hello")

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

	err = adapter.UninstallSkill(context.Background(), install, "unmanaged-skill")
	if err == nil {
		t.Fatal("UninstallSkill() should reject removal of unmanaged skill")
	}

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		t.Fatal("unmanaged skill directory should still exist after rejected uninstall")
	}
}

/** 验证 UpdateSkill 替换应用管理的技能文件并更新所有权标记版本 */
func TestUpdateSkillReplacesManagedSkillFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	skillDir := filepath.Join(codexHome, "skills", "my-skill")
	mustMkdirAll(t, skillDir)
	mustWriteFile(t, filepath.Join(skillDir, "prompt.md"), "old content")

	marker := map[string]interface{}{
		"skill_name":  "my-skill",
		"version":     "1.0.0",
		"managed_by":  "agent-skills-manager",
		"installed_at": now.Format(time.RFC3339),
	}
	markerJSON, _ := json.Marshal(marker)
	mustWriteFile(t, filepath.Join(skillDir, ".asm-managed"), string(markerJSON))

	sourceDir := filepath.Join(root, "source", "my-skill")
	mustMkdirAll(t, sourceDir)
	mustWriteFile(t, filepath.Join(sourceDir, "prompt.md"), "new content")

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

	mutation := agents.SkillMutation{
		Name:       "my-skill",
		SourcePath: sourceDir,
	}

	if err := adapter.UpdateSkill(context.Background(), install, mutation); err != nil {
		t.Fatalf("UpdateSkill() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(skillDir, "prompt.md"))
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if string(data) != "new content" {
		t.Fatalf("updated file content = %q, want %q", string(data), "new content")
	}

	markerData, err := os.ReadFile(filepath.Join(skillDir, ".asm-managed"))
	if err != nil {
		t.Fatalf("read updated marker: %v", err)
	}
	var updatedMarker map[string]interface{}
	if err := json.Unmarshal(markerData, &updatedMarker); err != nil {
		t.Fatalf("parse updated marker: %v", err)
	}
	if updatedMarker["version"] == "1.0.0" {
		t.Fatal("marker version should be updated")
	}
}

/** 验证 ValidateSkillInstall 确认已安装技能的完整性 */
func TestValidateSkillInstallConfirmsValidInstall(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	skillDir := filepath.Join(codexHome, "skills", "my-skill")
	mustMkdirAll(t, skillDir)
	mustWriteFile(t, filepath.Join(skillDir, "prompt.md"), "hello")

	marker := map[string]interface{}{
		"skill_name":  "my-skill",
		"version":     "1.0.0",
		"managed_by":  "agent-skills-manager",
		"installed_at": now.Format(time.RFC3339),
	}
	markerJSON, _ := json.Marshal(marker)
	mustWriteFile(t, filepath.Join(skillDir, ".asm-managed"), string(markerJSON))

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

	if err := adapter.ValidateSkillInstall(context.Background(), install, "my-skill"); err != nil {
		t.Fatalf("ValidateSkillInstall() error = %v", err)
	}
}

/** 验证 ValidateSkillInstall 检测缺失的技能目录 */
func TestValidateSkillInstallDetectsMissingSkill(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	now := time.Date(2026, time.May, 4, 10, 0, 0, 0, time.UTC)

	codexHome := filepath.Join(root, "codex")
	mustMkdirAll(t, filepath.Join(codexHome, "skills"))
	mustWriteFile(t, filepath.Join(codexHome, "skills", "other.md"), "other")

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

	err = adapter.ValidateSkillInstall(context.Background(), install, "nonexistent-skill")
	if err == nil {
		t.Fatal("ValidateSkillInstall() should detect missing skill")
	}
}

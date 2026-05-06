package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Registry struct {
	adapters []Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	cloned := make([]Adapter, 0, len(adapters))
	for _, adapter := range adapters {
		if adapter != nil {
			cloned = append(cloned, adapter)
		}
	}

	return &Registry{adapters: cloned}
}

func (r *Registry) DiscoverAll(ctx context.Context) ([]AgentInstall, error) {
	var allInstalls []AgentInstall
	for _, adapter := range r.adapters {
		installs, err := adapter.DiscoverAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("discover %s: %w", adapter.ID(), err)
		}
		allInstalls = append(allInstalls, installs...)
	}
	return allInstalls, nil
}

/** 列出指定代理安装下所有可见技能名称 */
func (r *Registry) ListInstalledSkills(ctx context.Context, install AgentInstall) ([]string, error) {
	for _, adapter := range r.adapters {
		if adapter.ID() == install.AgentID {
			return adapter.ListInstalledSkills(ctx, install)
		}
	}
	return nil, fmt.Errorf("adapter %q not found", install.AgentID)
}

/** 查找指定代理 ID 对应的适配器 */
func (r *Registry) AdapterFor(agentID string) (Adapter, bool) {
	for _, adapter := range r.adapters {
		if adapter.ID() == agentID {
			return adapter, true
		}
	}
	return nil, false
}

/** 安装技能到指定代理 */
func (r *Registry) InstallSkill(ctx context.Context, install AgentInstall, mutation SkillMutation) error {
	for _, adapter := range r.adapters {
		if adapter.ID() == install.AgentID {
			return adapter.InstallSkill(ctx, install, mutation)
		}
	}
	return fmt.Errorf("adapter %q not found", install.AgentID)
}

/** 从指定代理卸载技能 */
func (r *Registry) UninstallSkill(ctx context.Context, install AgentInstall, skillName string) error {
	for _, adapter := range r.adapters {
		if adapter.ID() == install.AgentID {
			return adapter.UninstallSkill(ctx, install, skillName)
		}
	}
	return fmt.Errorf("adapter %q not found", install.AgentID)
}

/** 更新指定代理的技能 */
func (r *Registry) UpdateSkill(ctx context.Context, install AgentInstall, mutation SkillMutation) error {
	for _, adapter := range r.adapters {
		if adapter.ID() == install.AgentID {
			return adapter.UpdateSkill(ctx, install, mutation)
		}
	}
	return fmt.Errorf("adapter %q not found", install.AgentID)
}

/** 验证指定代理的技能安装状态 */
func (r *Registry) ValidateSkillInstall(ctx context.Context, install AgentInstall, skillName string) error {
	for _, adapter := range r.adapters {
		if adapter.ID() == install.AgentID {
			return adapter.ValidateSkillInstall(ctx, install, skillName)
		}
	}
	return fmt.Errorf("adapter %q not found", install.AgentID)
}

type filesystemAdapter struct {
	config LocalAdapterConfig
}

type installCandidate struct {
	path         string
	sourceWeight int
}

func NewFilesystemAdapter(config LocalAdapterConfig) Adapter {
	return &filesystemAdapter{config: config}
}

func (a *filesystemAdapter) ID() string {
	return a.config.AgentID
}

func (a *filesystemAdapter) Discover(context.Context) (AgentInstall, error) {
	now := time.Now().UTC()
	if a.config.Now != nil {
		now = a.config.Now()
	}

	base := AgentInstall{
		AgentID:       a.config.AgentID,
		DisplayName:   a.config.DisplayName,
		LastScannedAt: now,
		Health:        HealthNotInstalled,
		LastErrorCode: ErrCodeInstallNotFound,
	}

	best := base
	bestRank := discoveryRank(base.Health)
	bestSourceWeight := -1

	for _, candidate := range a.candidateInstallPaths() {
		install, ok := a.inspectCandidate(now, candidate.path)
		if !ok {
			continue
		}
		rank := discoveryRank(install.Health)
		if rank > bestRank || (rank == bestRank && candidate.sourceWeight > bestSourceWeight) {
			best = install
			bestRank = rank
			bestSourceWeight = candidate.sourceWeight
		}
	}

	return best, nil
}

/** 返回适配器所有有效的安装路径（而非仅最佳路径） */
func (a *filesystemAdapter) DiscoverAll(ctx context.Context) ([]AgentInstall, error) {
	now := time.Now().UTC()
	if a.config.Now != nil {
		now = a.config.Now()
	}

	var result []AgentInstall
	for _, candidate := range a.candidateInstallPaths() {
		install, ok := a.inspectCandidate(now, candidate.path)
		if !ok {
			continue
		}
		if install.Health != HealthNotInstalled {
			result = append(result, install)
		}
	}

	if len(result) == 0 {
		result = append(result, AgentInstall{
			AgentID:       a.config.AgentID,
			DisplayName:   a.config.DisplayName,
			LastScannedAt: now,
			Health:        HealthNotInstalled,
			LastErrorCode: ErrCodeInstallNotFound,
		})
	}

	return result, nil
}

func (a *filesystemAdapter) ListInstalledSkills(_ context.Context, install AgentInstall) ([]string, error) {
	if install.SkillsPath == "" {
		return nil, fmt.Errorf("no skills path discovered for %s", install.AgentID)
	}

	entries, err := os.ReadDir(install.SkillsPath)
	if err != nil {
		return nil, err
	}

	skills := visibleSkillEntries(entries)
	sort.Strings(skills)
	return skills, nil
}

func (a *filesystemAdapter) InstallSkill(_ context.Context, install AgentInstall, mutation SkillMutation) error {
	if install.Health != HealthReady {
		return fmt.Errorf("cannot install skill into agent %q with health %q", install.AgentID, install.Health)
	}

	targetDir := filepath.Join(install.SkillsPath, mutation.Name)
	markerPath := filepath.Join(targetDir, markerFileName)

	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		if _, markerErr := os.Stat(markerPath); os.IsNotExist(markerErr) {
			return fmt.Errorf("cannot overwrite unmanaged skill %q in agent %q", mutation.Name, install.AgentID)
		}
	}

	if err := os.RemoveAll(targetDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing skill dir: %w", err)
	}

	if err := copyDir(mutation.SourcePath, targetDir); err != nil {
		return fmt.Errorf("copy skill files: %w", err)
	}

	if err := writeOwnershipMarker(targetDir, mutation.Name, "1.0.0"); err != nil {
		return fmt.Errorf("write ownership marker: %w", err)
	}

	return nil
}

func (a *filesystemAdapter) UninstallSkill(_ context.Context, install AgentInstall, skillName string) error {
	targetDir := filepath.Join(install.SkillsPath, skillName)
	markerPath := filepath.Join(targetDir, markerFileName)

	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		return fmt.Errorf("cannot uninstall unmanaged skill %q in agent %q", skillName, install.AgentID)
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("remove skill dir: %w", err)
	}

	return nil
}

func (a *filesystemAdapter) UpdateSkill(_ context.Context, install AgentInstall, mutation SkillMutation) error {
	targetDir := filepath.Join(install.SkillsPath, mutation.Name)
	markerPath := filepath.Join(targetDir, markerFileName)

	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		return fmt.Errorf("cannot update unmanaged skill %q in agent %q", mutation.Name, install.AgentID)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("read skill dir: %w", err)
	}

	for _, entry := range entries {
		if entry.Name() == markerFileName {
			continue
		}
		entryPath := filepath.Join(targetDir, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("remove old skill file %q: %w", entry.Name(), err)
		}
	}

	if err := copyDir(mutation.SourcePath, targetDir); err != nil {
		return fmt.Errorf("copy updated skill files: %w", err)
	}

	if err := writeOwnershipMarker(targetDir, mutation.Name, "2.0.0"); err != nil {
		return fmt.Errorf("update ownership marker: %w", err)
	}

	return nil
}

func (a *filesystemAdapter) ValidateSkillInstall(_ context.Context, install AgentInstall, skillName string) error {
	targetDir := filepath.Join(install.SkillsPath, skillName)

	info, err := os.Stat(targetDir)
	if err != nil {
		return fmt.Errorf("skill dir %q does not exist or is unreadable: %w", targetDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skill path %q is not a directory", targetDir)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("cannot read skill dir %q: %w", targetDir, err)
	}

	hasContent := false
	for _, entry := range entries {
		if !isIgnoredSkillEntry(entry.Name()) {
			hasContent = true
			break
		}
	}
	if !hasContent {
		return fmt.Errorf("skill dir %q has no content entries", targetDir)
	}

	return nil
}

func (a *filesystemAdapter) candidateInstallPaths() []installCandidate {
	seen := make(map[string]struct{})
	candidates := make([]installCandidate, 0, len(a.config.DefaultInstallPaths)+2)

	appendCandidate := func(path string, sourceWeight int) {
		if path == "" {
			return
		}
		cleaned := filepath.Clean(path)
		if _, ok := seen[cleaned]; ok {
			return
		}
		seen[cleaned] = struct{}{}
		candidates = append(candidates, installCandidate{
			path:         cleaned,
			sourceWeight: sourceWeight,
		})
	}

	for _, path := range a.config.DefaultInstallPaths {
		appendCandidate(path, 0)
	}

	appendCandidate(a.config.OverrideInstallPath, 1)

	a.confirmExecutablePresence()

	return candidates
}

func (a *filesystemAdapter) inspectCandidate(now time.Time, installPath string) (AgentInstall, bool) {
	install := AgentInstall{
		AgentID:       a.config.AgentID,
		DisplayName:   a.config.DisplayName,
		InstallPath:   installPath,
		LastScannedAt: now,
	}

	info, err := os.Stat(installPath)
	if errors.Is(err, os.ErrNotExist) {
		return AgentInstall{}, false
	}
	if err != nil {
		install.Health = HealthInstalledButUnreadable
		install.LastErrorCode = ErrCodeInstallUnreadable
		install.LastErrorMessage = err.Error()
		return install, true
	}
	if !info.IsDir() {
		install.Health = HealthInstalledButUnreadable
		install.LastErrorCode = ErrCodeInstallUnreadable
		install.LastErrorMessage = "install path is not a directory"
		return install, true
	}

	skillsPath := filepath.Join(installPath, a.config.SkillsRelativePath)
	install.SkillsPath = skillsPath

	skillsInfo, err := os.Stat(skillsPath)
	if errors.Is(err, os.ErrNotExist) {
		install.Health = HealthInstalledButSkillPathMissing
		install.LastErrorCode = ErrCodeSkillPathMissing
		install.LastErrorMessage = "skills path does not exist"
		return install, true
	}
	if err != nil {
		install.Health = HealthInstalledButUnreadable
		install.LastErrorCode = ErrCodeInstallUnreadable
		install.LastErrorMessage = err.Error()
		return install, true
	}
	if !skillsInfo.IsDir() {
		install.Health = HealthInstalledButUnreadable
		install.LastErrorCode = ErrCodeInstallUnreadable
		install.LastErrorMessage = "skills path is not a directory"
		return install, true
	}

	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		install.Health = HealthInstalledButUnreadable
		install.LastErrorCode = ErrCodeInstallUnreadable
		install.LastErrorMessage = err.Error()
		return install, true
	}
	if len(visibleSkillEntries(entries)) == 0 {
		install.Health = HealthInstalledButSkillPathEmpty
		install.LastErrorCode = ErrCodeSkillPathEmpty
		install.LastErrorMessage = "skills path is empty"
		return install, true
	}

	install.Health = HealthReady
	return install, true
}

func discoveryRank(health HealthStatus) int {
	switch health {
	case HealthReady:
		return 4
	case HealthInstalledButSkillPathEmpty:
		return 3
	case HealthInstalledButSkillPathMissing:
		return 2
	case HealthInstalledButUnreadable:
		return 1
	case HealthNotInstalled:
		return 0
	default:
		return -1
	}
}

func (a *filesystemAdapter) confirmExecutablePresence() {
	if a.config.LookPath == nil {
		return
	}

	for _, executable := range a.config.ExecutableNames {
		found, err := a.config.LookPath(executable)
		if err == nil && found != "" {
			return
		}
	}
}

func visibleSkillEntries(entries []os.DirEntry) []string {
	skills := make([]string, 0, len(entries))
	for _, entry := range entries {
		if isIgnoredSkillEntry(entry.Name()) {
			continue
		}
		skills = append(skills, entry.Name())
	}
	return skills
}

func isIgnoredSkillEntry(name string) bool {
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	if strings.HasSuffix(name, "~") {
		return true
	}
	if strings.HasSuffix(name, ".tmp") {
		return true
	}
	if strings.HasSuffix(name, ".swp") {
		return true
	}
	return false
}

const markerFileName = ".asm-managed"

/** 递归复制源目录内容到目标目录 */
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

/** 写入应用管理所有权标记文件 */
func writeOwnershipMarker(skillDir, skillName, version string) error {
	now := time.Now().UTC()
	marker := map[string]interface{}{
		"skill_name":  skillName,
		"version":     version,
		"managed_by":  "agent-skills-manager",
		"installed_at": now.Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(skillDir, markerFileName), data, 0o644)
}

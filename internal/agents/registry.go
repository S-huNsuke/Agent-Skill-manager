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
	var errs []string
	for _, adapter := range r.adapters {
		installs, err := adapter.DiscoverAll(ctx)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", adapter.ID(), err))
			continue
		}
		allInstalls = append(allInstalls, installs...)
	}
	if len(errs) == len(r.adapters) {
		return nil, fmt.Errorf("all adapters failed: %s", strings.Join(errs, "; "))
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

func (a *filesystemAdapter) SkillsRelativePath() string {
	return a.config.SkillsRelativePath
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

	version := mutation.Version
	if version == "" {
		version = "1.0.0"
	}
	if err := writeOwnershipMarker(targetDir, mutation.Name, version); err != nil {
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

	var version string
	if mutation.Version != "" {
		version = mutation.Version
	} else {
		currentVersion := readInstalledVersion(targetDir)
		if currentVersion != "" {
			version = incrementVersion(currentVersion)
		} else {
			version = "1.0.0"
		}
	}

	if err := writeOwnershipMarker(targetDir, mutation.Name, version); err != nil {
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

	if !a.confirmExecutablePresence() {
		install.Health = HealthInstalledButExecutableMissing
		install.LastErrorCode = ErrCodeExecutableNotFound
		install.LastErrorMessage = "agent executable not found"
		return install, true
	}

	install.Health = HealthReady
	return install, true
}

func discoveryRank(health HealthStatus) int {
	switch health {
	case HealthReady:
		return 4
	case HealthInstalledButExecutableMissing:
		return 3
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

/** 检查可执行文件是否存在，返回是否找到至少一个 */
func (a *filesystemAdapter) confirmExecutablePresence() bool {
	if len(a.config.ExecutableNames) == 0 {
		return true
	}
	if a.config.LookPath == nil {
		return true
	}

	for _, executable := range a.config.ExecutableNames {
		found, err := a.config.LookPath(executable)
		if err == nil && found != "" {
			return true
		}
	}
	return false
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

/** 递归复制源目录内容到目标目录，保留源文件权限 */
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
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
			entryInfo, statErr := os.Stat(srcPath)
			perm := os.FileMode(0o644)
			if statErr == nil {
				perm = entryInfo.Mode()
			}
			if err := os.WriteFile(dstPath, data, perm); err != nil {
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
		"skill_name":   skillName,
		"version":      version,
		"managed_by":   "agent-skills-manager",
		"installed_at": now.Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(skillDir, markerFileName), data, 0o644)
}

/** 从标记文件中读取已安装技能的版本号 */
func readInstalledVersion(skillDir string) string {
	markerPath := filepath.Join(skillDir, markerFileName)
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return ""
	}

	var marker map[string]interface{}
	if err := json.Unmarshal(data, &marker); err != nil {
		return ""
	}

	version, ok := marker["version"].(string)
	if !ok {
		return ""
	}
	return version
}

/** 递增语义版本号的补丁版本（如 "1.0.0" -> "1.0.1"） */
func incrementVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "1.0.0"
	}

	patch := 0
	if _, err := fmt.Sscanf(parts[2], "%d", &patch); err != nil {
		return "1.0.0"
	}

	parts[2] = fmt.Sprintf("%d", patch+1)
	return strings.Join(parts, ".")
}

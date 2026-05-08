package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/caojun/agent-skills-manager/internal/agents"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetAgents() []AgentViewModel {
	if a.registry == nil {
		return []AgentViewModel{}
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return []AgentViewModel{}
	}

	type agentGroup struct {
		agentID     string
		displayName string
		bestHealth  agents.HealthStatus
		totalSkills int
		primaryPath string
		allPaths    []string
	}

	groups := make(map[string]*agentGroup)
	var order []string

	for _, install := range installs {
		g, exists := groups[install.AgentID]
		if !exists {
			g = &agentGroup{
				agentID:     install.AgentID,
				displayName: install.DisplayName,
				bestHealth:  install.Health,
			}
			groups[install.AgentID] = g
			order = append(order, install.AgentID)
		}

		if healthRank(install.Health) > healthRank(g.bestHealth) {
			g.bestHealth = install.Health
		}

		if install.InstallPath != "" {
			g.allPaths = append(g.allPaths, install.InstallPath)
			if g.primaryPath == "" || install.Health == agents.HealthReady {
				g.primaryPath = install.InstallPath
			}
		}

		if install.Health == agents.HealthReady {
			skills, skillErr := a.registry.ListInstalledSkills(context.Background(), install)
			if skillErr == nil {
				g.totalSkills += len(skills)
			}
		}
	}

	result := make([]AgentViewModel, 0, len(groups))
	for _, agentID := range order {
		g := groups[agentID]
		status := "degraded"
		mode := "未安装"
		summary := ""

		switch g.bestHealth {
		case agents.HealthReady:
			status = "healthy"
			mode = "运行中"
			if len(g.allPaths) > 1 {
				summary = fmt.Sprintf("安装路径: %s（共 %d 个路径）", g.primaryPath, len(g.allPaths))
			} else {
				summary = fmt.Sprintf("安装路径: %s", g.primaryPath)
			}
		case agents.HealthInstalledButSkillPathEmpty:
			status = "degraded"
			mode = "技能为空"
			summary = fmt.Sprintf("安装路径: %s（技能目录为空）", g.primaryPath)
		case agents.HealthInstalledButSkillPathMissing:
			status = "degraded"
			mode = "缺少技能目录"
			summary = fmt.Sprintf("安装路径: %s（无 skills 目录）", g.primaryPath)
		case agents.HealthInstalledButUnreadable:
			status = "degraded"
			mode = "读取异常"
			summary = fmt.Sprintf("安装路径: %s（读取失败）", g.primaryPath)
		case agents.HealthNotInstalled:
			status = "degraded"
			mode = "未安装"
			summary = "未在本机检测到安装"
		}

		result = append(result, AgentViewModel{
			ID:          g.agentID,
			Name:        g.displayName,
			Mode:        mode,
			Status:      status,
			Summary:     summary,
			Focus:       "",
			InstallPath: g.primaryPath,
			Skills:      g.totalSkills,
		})
	}

	return result
}

/** 返回所有代理下已安装技能的完整列表，按 AgentID+技能名 去重 */
func (a *App) GetSkills() []SkillViewModel {
	if a.registry == nil {
		return []SkillViewModel{}
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return []SkillViewModel{}
	}

	seen := make(map[string]bool)
	result := make([]SkillViewModel, 0)
	for _, install := range installs {
		if install.Health != agents.HealthReady {
			continue
		}

		skillNames, err := a.registry.ListInstalledSkills(context.Background(), install)
		if err != nil {
			continue
		}

		for _, name := range skillNames {
			key := fmt.Sprintf("%s-%s", install.AgentID, name)
			if seen[key] {
				continue
			}
			seen[key] = true

			skillDir := filepath.Join(install.SkillsPath, name)
			installedAt := ""
			healthStatus := "ok"
			healthMessage := ""

			info, statErr := os.Stat(skillDir)
			if statErr != nil {
				healthStatus = "error"
				healthMessage = fmt.Sprintf("技能目录不存在: %s", skillDir)
			} else {
				if !info.IsDir() {
					healthStatus = "error"
					healthMessage = fmt.Sprintf("路径不是目录: %s", skillDir)
				} else {
					installedAt = info.ModTime().Format("2006-01-02")

					entries, readErr := os.ReadDir(skillDir)
					if readErr != nil {
						healthStatus = "error"
						healthMessage = fmt.Sprintf("无法读取技能目录: %s", readErr.Error())
					} else {
						hasContent := false
						for _, entry := range entries {
							if !strings.HasPrefix(entry.Name(), ".") {
								hasContent = true
								break
							}
						}
						if !hasContent {
							healthStatus = "warning"
							healthMessage = "技能目录为空，没有有效文件"
						} else {
							desc, descErr := readSkillDescription(skillDir)
							if descErr == nil && desc != "" {
								healthMessage = desc
							}
						}
					}
				}
			}

			statusLabel := "已安装"
			markerPath := filepath.Join(skillDir, ".asm-managed")
			if _, markerErr := os.Stat(markerPath); markerErr == nil {
				statusLabel = "已管理"
			}

			summary := fmt.Sprintf("位于 %s", skillDir)
			if healthStatus == "ok" && healthMessage != "" {
				summary = healthMessage
				healthMessage = ""
			}

			result = append(result, SkillViewModel{
				ID:            key,
				Name:          name,
				Group:         install.DisplayName,
				InstalledAt:   installedAt,
				Summary:       summary,
				StatusLabel:   statusLabel,
				Projects:      0,
				Agent:         install.DisplayName,
				HealthStatus:  healthStatus,
				HealthMessage: healthMessage,
			})
		}
	}

	return result
}

/** 返回商店条目列表，从已同步的远程市场获取 */
func (a *App) InstallSkill(agentID string, skillName string, sourcePath string) string {
	if a.registry == nil {
		return "error: registry not initialized"
	}

	if sourcePath == "" {
		sourcePath = a.findSkillCachePath(skillName)
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	for _, install := range installs {
		if (install.AgentID == agentID || install.DisplayName == agentID) && install.Health == agents.HealthReady {
			mutation := agents.SkillMutation{
				Name:       skillName,
				SourcePath: sourcePath,
			}
			if err := a.registry.InstallSkill(context.Background(), install, mutation); err != nil {
				return fmt.Sprintf("error: %s", err)
			}
			return "ok"
		}
	}
	return "error: agent not found or not ready"
}

/** 从商店缓存中查找技能的本地缓存路径 */
func (a *App) findSkillCachePath(skillName string) string {
	a.catalogMu.RLock()
	defer a.catalogMu.RUnlock()

	for _, item := range a.catalogItems {
		if item.Name == skillName && item.LocalCachePath != "" {
			return item.LocalCachePath
		}
	}
	return ""
}

/** 从指定代理卸载技能 */
func (a *App) UninstallSkill(agentID string, skillName string) string {
	if a.registry == nil {
		return "error: registry not initialized"
	}
	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	for _, install := range installs {
		if (install.AgentID == agentID || install.DisplayName == agentID) && install.Health == agents.HealthReady {
			if err := a.registry.UninstallSkill(context.Background(), install, skillName); err != nil {
				return fmt.Sprintf("error: %s", err)
			}
			return "ok"
		}
	}
	return "error: agent not found or not ready"
}

/** 更新指定代理的技能 */
func (a *App) UpdateSkill(agentID string, skillName string, sourcePath string) string {
	if a.registry == nil {
		return "error: registry not initialized"
	}
	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	for _, install := range installs {
		if (install.AgentID == agentID || install.DisplayName == agentID) && install.Health == agents.HealthReady {
			mutation := agents.SkillMutation{
				Name:       skillName,
				SourcePath: sourcePath,
			}
			if err := a.registry.UpdateSkill(context.Background(), install, mutation); err != nil {
				return fmt.Sprintf("error: %s", err)
			}
			return "ok"
		}
	}
	return "error: agent not found or not ready"
}

func (a *App) GetAgentSkills(agentID string) []SkillViewModel {
	if a.registry == nil {
		return []SkillViewModel{}
	}
	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return []SkillViewModel{}
	}
	for _, install := range installs {
		if (install.AgentID == agentID || install.DisplayName == agentID) && install.Health == agents.HealthReady {
			skillNames, err := a.registry.ListInstalledSkills(context.Background(), install)
			if err != nil {
				return []SkillViewModel{}
			}
			var result []SkillViewModel
			for _, name := range skillNames {
				skillDir := filepath.Join(install.SkillsPath, name)
				installedAt := ""
				if info, statErr := os.Stat(skillDir); statErr == nil {
					installedAt = info.ModTime().Format("2006-01-02")
				}
				statusLabel := "已安装"
				markerPath := filepath.Join(skillDir, ".asm-managed")
				if _, markerErr := os.Stat(markerPath); markerErr == nil {
					statusLabel = "已管理"
				}
				result = append(result, SkillViewModel{
					ID:          fmt.Sprintf("%s-%s", install.AgentID, name),
					Name:        name,
					Group:       install.DisplayName,
					InstalledAt: installedAt,
					Summary:     fmt.Sprintf("位于 %s", skillDir),
					StatusLabel: statusLabel,
					Projects:    0,
					Agent:       install.DisplayName,
				})
			}
			return result
		}
	}
	return []SkillViewModel{}
}

/** 在 Finder 中打开指定路径 */
func (a *App) OpenInFinder(path string) string {
	if _, err := os.Stat(path); err != nil {
		return fmt.Sprintf("error: path not found: %s", path)
	}
	cmd := exec.Command("open", path)
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	return "ok"
}

/** 打开目录选择对话框，返回用户选择的目录路径 */
func (a *App) SelectDirectory(title string) string {
	if a.ctx == nil {
		return ""
	}

	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return ""
	}
	return dir
}

/** 修复代理的技能目录（创建缺失的 skills 目录） */
func (a *App) RepairAgent(agentID string) string {
	if a.registry == nil {
		return "error: registry not initialized"
	}
	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	for _, install := range installs {
		if install.AgentID == agentID || install.DisplayName == agentID {
			if install.InstallPath == "" {
				return "error: no install path found"
			}
			skillsPath := filepath.Join(install.InstallPath, "skills")
			if _, err := os.Stat(skillsPath); os.IsNotExist(err) {
				if mkdirErr := os.MkdirAll(skillsPath, 0o755); mkdirErr != nil {
					return fmt.Sprintf("error: failed to create skills directory: %s", mkdirErr)
				}
				return "ok"
			}
			return "ok: skills directory already exists"
		}
	}
	return "error: agent not found"
}

/** 获取代理的详细信息，聚合所有安装路径 */
func (a *App) GetAgentDetail(agentID string) AgentDetailViewModel {
	if a.registry == nil {
		return AgentDetailViewModel{ID: agentID, Found: false}
	}
	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return AgentDetailViewModel{ID: agentID, Found: false}
	}

	var matched []agents.AgentInstall
	for _, install := range installs {
		if install.AgentID == agentID || install.DisplayName == agentID {
			matched = append(matched, install)
		}
	}

	if len(matched) == 0 {
		return AgentDetailViewModel{ID: agentID, Found: false}
	}

	var bestInstall *agents.AgentInstall
	for i := range matched {
		if bestInstall == nil || healthRank(matched[i].Health) > healthRank(bestInstall.Health) {
			bestInstall = &matched[i]
		}
	}

	totalSkillCount := 0
	allSkillNames := make(map[string]bool)
	var allInstallPaths []string
	var allSkillsPaths []string

	for _, install := range matched {
		if install.InstallPath != "" {
			allInstallPaths = append(allInstallPaths, install.InstallPath)
		}
		if install.SkillsPath != "" {
			allSkillsPaths = append(allSkillsPaths, install.SkillsPath)
		}
		if install.Health == agents.HealthReady {
			names, listErr := a.registry.ListInstalledSkills(context.Background(), install)
			if listErr == nil {
				totalSkillCount += len(names)
				for _, name := range names {
					allSkillNames[name] = true
				}
			}
		}
	}

	uniqueSkillNames := make([]string, 0, len(allSkillNames))
	for name := range allSkillNames {
		uniqueSkillNames = append(uniqueSkillNames, name)
	}
	sort.Strings(uniqueSkillNames)

	if len(allInstallPaths) == 0 {
		allInstallPaths = make([]string, 0)
	}
	if len(allSkillsPaths) == 0 {
		allSkillsPaths = make([]string, 0)
	}

	return AgentDetailViewModel{
		ID:               bestInstall.AgentID,
		DisplayName:      bestInstall.DisplayName,
		Found:            true,
		InstallPath:      bestInstall.InstallPath,
		InstallPaths:     allInstallPaths,
		SkillsPath:       bestInstall.SkillsPath,
		SkillsPaths:      allSkillsPaths,
		Health:           string(bestInstall.Health),
		LastScannedAt:    bestInstall.LastScannedAt.Format("2006-01-02 15:04:05"),
		LastErrorCode:    bestInstall.LastErrorCode,
		LastErrorMessage: bestInstall.LastErrorMessage,
		SkillCount:       totalSkillCount,
		SkillNames:       uniqueSkillNames,
	}
}

/** 读取技能的完整描述内容，用于 AI 分析 */
func (a *App) ExplainSkill(agentID string, skillName string) SkillExplanationViewModel {
	result := SkillExplanationViewModel{
		AgentID:   agentID,
		SkillName: skillName,
		Found:     false,
		Files:     make([]string, 0),
	}

	if a.registry == nil {
		return result
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return result
	}

	for _, install := range installs {
		if install.AgentID == agentID || install.DisplayName == agentID {
			if install.Health != agents.HealthReady {
				continue
			}
			skillDir := filepath.Join(install.SkillsPath, skillName)
			if info, statErr := os.Stat(skillDir); statErr != nil || !info.IsDir() {
				continue
			}

			result.Found = true
			result.SkillPath = skillDir
			result.AgentName = install.DisplayName

			readmeFiles := []string{"README.md", "readme.md", "description.md", "skill.md", "index.md"}
			for _, name := range readmeFiles {
				path := filepath.Join(skillDir, name)
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					continue
				}
				content := string(data)
				if len(content) > 8000 {
					content = content[:8000] + "\n...(内容过长，已截断)"
				}
				result.ReadmeContent = content
				result.ReadmeFile = name
				break
			}

			entries, dirErr := os.ReadDir(skillDir)
			if dirErr == nil {
				for _, entry := range entries {
					if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
						result.Files = append(result.Files, entry.Name())
					}
				}
			}

			if result.ReadmeContent == "" {
				result.ReadmeContent = fmt.Sprintf("技能「%s」没有 README 或描述文件。目录下包含以下文件：\n%s",
					skillName, strings.Join(result.Files, "\n"))
			}

			return result
		}
	}

	return result
}

/** 读取技能目录下的描述文件 */
func readSkillDescription(skillDir string) (string, error) {
	candidates := []string{
		"README.md",
		"readme.md",
		"description.md",
		"skill.md",
	}

	for _, name := range candidates {
		path := filepath.Join(skillDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if len(trimmed) > 120 {
				return trimmed[:120] + "...", nil
			}
			return trimmed, nil
		}
	}

	return "", fmt.Errorf("no description found")
}

/** 返回最近的活动记录 */

/** 返回技能的详细信息 */
func (a *App) GetSkillDetail(agentID string, skillName string) SkillDetailViewModel {
	result := SkillDetailViewModel{
		ID:      fmt.Sprintf("%s-%s", agentID, skillName),
		Name:    skillName,
		AgentID: agentID,
		Found:   false,
		Files:   make([]string, 0),
		Tags:    make([]string, 0),
	}

	if a.registry == nil {
		return result
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return result
	}

	for _, install := range installs {
		if (install.AgentID == agentID || install.DisplayName == agentID) && install.Health == agents.HealthReady {
			skillDir := filepath.Join(install.SkillsPath, skillName)
			if info, statErr := os.Stat(skillDir); statErr != nil || !info.IsDir() {
				continue
			}

			result.Found = true
			result.AgentName = install.DisplayName
			result.InstallPath = skillDir

			if info, statErr := os.Stat(skillDir); statErr == nil {
				result.InstalledAt = info.ModTime().Format("2006-01-02")
			}

			markerPath := filepath.Join(skillDir, ".asm-managed")
			if _, markerErr := os.Stat(markerPath); markerErr == nil {
				result.Source = "本应用管理"
			} else {
				result.Source = "手动安装"
			}

			if desc, descErr := readSkillDescription(skillDir); descErr == nil {
				result.Description = desc
			}

			entries, dirErr := os.ReadDir(skillDir)
			if dirErr == nil {
				for _, entry := range entries {
					if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
						result.Files = append(result.Files, entry.Name())
					}
				}
			}

			return result
		}
	}
	return result
}

/** 返回商店源列表 */
func (a *App) BatchUpdateSkills(agentID string, skillNames string) string {
	if a.registry == nil {
		return "error: registry not initialized"
	}
	names := strings.Split(skillNames, ",")
	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	for _, install := range installs {
		if (install.AgentID == agentID || install.DisplayName == agentID) && install.Health == agents.HealthReady {
			for _, name := range names {
				mutation := agents.SkillMutation{Name: strings.TrimSpace(name)}
				if err := a.registry.UpdateSkill(context.Background(), install, mutation); err != nil {
					return fmt.Sprintf("error updating %s: %s", name, err)
				}
			}
			return "ok"
		}
	}
	return "error: agent not found or not ready"
}

/** 批量卸载技能 */
func (a *App) BatchUninstallSkills(agentID string, skillNames string) string {
	if a.registry == nil {
		return "error: registry not initialized"
	}
	names := strings.Split(skillNames, ",")
	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	for _, install := range installs {
		if (install.AgentID == agentID || install.DisplayName == agentID) && install.Health == agents.HealthReady {
			for _, name := range names {
				if err := a.registry.UninstallSkill(context.Background(), install, strings.TrimSpace(name)); err != nil {
					return fmt.Sprintf("error uninstalling %s: %s", name, err)
				}
			}
			return "ok"
		}
	}
	return "error: agent not found or not ready"
}

/** StatusTone 类型别名，用于诊断信息 */

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goRuntime "runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
	"github.com/caojun/agent-skills-manager/internal/ai"
	"github.com/caojun/agent-skills-manager/internal/domain"
	"github.com/caojun/agent-skills-manager/internal/reconcile"
	"github.com/caojun/agent-skills-manager/internal/skillgroups"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

/** 返回完整的应用快照，供前端一次性加载 */
func (a *App) GetSnapshot() AppSnapshot {
	return AppSnapshot{
		Dashboard:   a.GetDashboard(),
		Agents:      a.GetAgents(),
		Skills:      a.GetSkills(),
		Store:       a.GetStoreItems(),
		Projects:    a.GetProjects(),
		Assistant:   a.GetAssistantTask(),
		Diagnostics: a.GetDiagnostics(),
	}
}

/** 返回仪表盘快照，聚合自各后端服务 */
func (a *App) GetDashboard() DashboardSnapshot {
	agentCount := 0
	skillCount := 0
	healthyCount := 0

	if a.registry != nil {
		installs, err := a.registry.DiscoverAll(context.Background())
		if err == nil {
			seenAgents := make(map[string]bool)
			for _, install := range installs {
				if seenAgents[install.AgentID] {
					if install.Health == agents.HealthReady {
						skills, skillErr := a.registry.ListInstalledSkills(context.Background(), install)
						if skillErr == nil {
							skillCount += len(skills)
						}
					}
					continue
				}
				seenAgents[install.AgentID] = true

				if install.Health == agents.HealthReady {
					agentCount++
					healthyCount++
					skills, skillErr := a.registry.ListInstalledSkills(context.Background(), install)
					if skillErr == nil {
						skillCount += len(skills)
					}
				} else if install.Health != agents.HealthNotInstalled {
					agentCount++
				}
			}
		}
	}

	highlights := []DashboardHighlight{
		{
			ID: "agents", Title: "已发现代理", Value: fmt.Sprintf("%d 个", agentCount),
			Detail: fmt.Sprintf("%d 个运行正常", healthyCount), Tone: "stable", Tag: "代理",
		},
		{
			ID: "skills", Title: "已安装技能", Value: fmt.Sprintf("%d 个", skillCount),
			Detail: "跨所有代理统计", Tone: "stable", Tag: "技能",
		},
		{
			ID: "health", Title: "任务状态", Value: "0 个任务",
			Detail: "暂无运行中的任务。", Tone: "stable", Tag: "概况",
		},
	}

	return DashboardSnapshot{
		Title:     "本机技能与运行状态",
		Summary:   "查看本机 AI 代理、已安装技能和任务执行状态。",
		Spotlight: fmt.Sprintf("已发现 %d 个代理，%d 个技能", agentCount, skillCount),
		Highlights: highlights,
		Tasks:      []DashboardTask{},
		Notes: []string{
			"安装技能前请先查看兼容性信息。",
			"每个项目只能绑定一个技能组。",
		},
	}
}

/** 返回已发现的适配器列表，按 AgentID 合并多个安装路径，避免重复 */
func (a *App) GetAgents() []AgentViewModel {
	if a.registry == nil {
		return []AgentViewModel{}
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return []AgentViewModel{}
	}

	type agentGroup struct {
		agentID      string
		displayName  string
		bestHealth   agents.HealthStatus
		totalSkills  int
		primaryPath  string
		allPaths     []string
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
func (a *App) GetStoreItems() []StoreItemViewModel {
	a.catalogMu.RLock()
	defer a.catalogMu.RUnlock()

	result := make([]StoreItemViewModel, 0, len(a.catalogItems))
	result = append(result, a.catalogItems...)
	return result
}

/** 返回项目列表 */
func (a *App) GetProjects() []ProjectViewModel {
	a.projectsMu.RLock()
	defer a.projectsMu.RUnlock()

	result := make([]ProjectViewModel, 0, len(a.projects))
	result = append(result, a.projects...)
	return result
}

/** 创建新项目 */
func (a *App) CreateProject(name string, path string) ProjectViewModel {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	project := ProjectViewModel{
		ID:              fmt.Sprintf("proj-custom-%d", time.Now().UnixMilli()),
		Name:            name,
		Path:            path,
		Stage:           "新建",
		BoundSkillGroup: "",
		BoundAgentID:    "",
		BoundAgentName:  "",
		SkillNames:      make([]string, 0),
		Summary:         fmt.Sprintf("自定义项目: %s", name),
		Needs:           make([]string, 0),
		LocalAgents:     make([]string, 0),
		Recent:          make([]string, 0),
		CreatedAt:       time.Now().Format("2006-01-02 15:04"),
	}

	a.projects = append(a.projects, project)

	if a.projectsRepo != nil {
		_ = a.projectsRepo.Put(context.Background(), projectVMToDomain(project))
	}

	return project
}

/** 删除项目 */
func (a *App) DeleteProject(projectID string) string {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	for i, p := range a.projects {
		if p.ID == projectID {
			a.projects = append(a.projects[:i], a.projects[i+1:]...)

			if a.projectsRepo != nil {
				_ = a.projectsRepo.Delete(context.Background(), projectID)
			}

			return "ok"
		}
	}
	return "error: project not found"
}

/** 为项目绑定代理 */
func (a *App) BindAgentToProject(projectID string, agentID string) string {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	agentName := agentID
	if a.registry != nil {
		installs, err := a.registry.DiscoverAll(context.Background())
		if err == nil {
			for _, install := range installs {
				if install.AgentID == agentID || install.DisplayName == agentID {
					agentName = install.DisplayName
					break
				}
			}
		}
	}

	for i := range a.projects {
		if a.projects[i].ID == projectID {
			a.projects[i].BoundAgentID = agentID
			a.projects[i].BoundAgentName = agentName
			a.projects[i].Recent = append([]string{fmt.Sprintf("绑定代理: %s (%s)", agentName, time.Now().Format("15:04"))}, a.projects[i].Recent...)

			if a.projectsRepo != nil {
				_ = a.projectsRepo.Put(context.Background(), projectVMToDomain(a.projects[i]))
			}

			return "ok"
		}
	}
	return "error: project not found"
}

/** 为项目绑定技能组 */
func (a *App) BindSkillGroupToProject(projectID string, groupName string) string {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	var skillNames []string
	for _, g := range a.skillGroups {
		if g.Name == groupName {
			skillNames = g.SkillNames
			break
		}
	}

	for i := range a.projects {
		if a.projects[i].ID == projectID {
			a.projects[i].BoundSkillGroup = groupName
			if len(skillNames) > 0 {
				a.projects[i].SkillNames = skillNames
			}
			a.projects[i].Recent = append([]string{fmt.Sprintf("绑定技能组: %s (%s)", groupName, time.Now().Format("15:04"))}, a.projects[i].Recent...)

			if a.projectsRepo != nil {
				_ = a.projectsRepo.Put(context.Background(), projectVMToDomain(a.projects[i]))
			}

			return "ok"
		}
	}
	return "error: project not found"
}

/** 刷新项目列表（重新扫描本地目录） */
func (a *App) RefreshProjects() []ProjectViewModel {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	customProjects := make([]ProjectViewModel, 0)
	for _, p := range a.projects {
		if strings.HasPrefix(p.ID, "proj-custom-") {
			customProjects = append(customProjects, p)
		}
	}

	scanned := scanLocalProjects()
	a.projects = append(scanned, customProjects...)

	result := make([]ProjectViewModel, 0, len(a.projects))
	result = append(result, a.projects...)
	return result
}

/** 返回当前助手任务 */
func (a *App) GetAssistantTask() AssistantTaskViewModel {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()

	if a.activeTask != nil {
		return *a.activeTask
	}

	return AssistantTaskViewModel{
		ID:             "assistant-idle",
		Request:        "",
		Status:         "queued",
		NextStep:       "等待用户输入目标",
		Summary:        "AI 助手待命中，输入目标即可开始规划。",
		Recommendation: "输入一个目标，让 AI 帮你规划技能安装与修复。",
		Reason:         "",
		Records:        []string{},
	}
}

/** 返回诊断信息列表 */
func (a *App) GetDiagnostics() []DiagnosticItemViewModel {
	info := a.GetAppInfo()

	agentCount := 0
	healthyCount := 0
	if a.registry != nil {
		installs, err := a.registry.DiscoverAll(context.Background())
		if err == nil {
			seenAgents := make(map[string]bool)
			for _, install := range installs {
				if seenAgents[install.AgentID] {
					continue
				}
				seenAgents[install.AgentID] = true
				agentCount++
				if install.Health == agents.HealthReady {
					healthyCount++
				}
			}
		}
	}

	items := []DiagnosticItemViewModel{
		{
			ID: "diag-app", Label: "应用版本",
			Value: fmt.Sprintf("%s %s", info.Name, info.Version),
			Tone:  "stable",
		},
		{
			ID: "diag-frontend", Label: "前端资源",
			Value: func() string {
				if info.UsesEmbeddedAssets {
					return "使用嵌入资源"
				}
				return info.FrontendDistDir
			}(),
			Tone: "stable",
		},
		{
			ID: "diag-wails", Label: "Wails 绑定",
			Value: "已连接后端",
			Tone:  "stable",
		},
		{
			ID: "diag-agents", Label: "代理发现",
			Value: fmt.Sprintf("已发现 %d 个代理（%d 个正常）", agentCount, healthyCount),
			Tone: func() StatusTone {
				if healthyCount == 0 && agentCount > 0 {
					return "attention"
				}
				return "stable"
			}(),
		},
	}

	return items
}

/** 重新扫描并返回最新快照 */
func (a *App) RefreshSnapshot() AppSnapshot {
	return a.GetSnapshot()
}

/** 安装技能到指定代理，如果 sourcePath 为空则从商店缓存查找 */
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

/** 提交 AI 助手目标 */
func (a *App) SubmitGoal(goal string) AssistantTaskViewModel {
	var result AssistantTaskViewModel
	ctx := context.Background()

	if a.bridge != nil {
		// 收集当前系统状态作为上下文
		agents := a.GetAgents()
		skills := a.GetSkills()
		projects := a.GetProjects()
		storeItems := a.GetStoreItems()

		// 构建上下文信息
		contextData := map[string]any{
			"agents_count":      len(agents),
			"skills_count":      len(skills),
			"projects_count":    len(projects),
			"store_items_count": len(storeItems),
		}

		// 添加代理列表
		agentsList := make([]map[string]any, 0, len(agents))
		for _, agent := range agents {
			agentsList = append(agentsList, map[string]any{
				"id":     agent.ID,
				"name":   agent.Name,
				"status": agent.Status,
				"skills": agent.Skills,
			})
		}
		contextData["agents"] = agentsList

		// 添加技能列表
		skillsList := make([]map[string]any, 0, len(skills))
		for _, skill := range skills {
			skillsList = append(skillsList, map[string]any{
				"name":    skill.Name,
				"agent":   skill.Agent,
				"status":  skill.StatusLabel,
				"summary": skill.Summary,
			})
		}
		contextData["skills"] = skillsList

		// 添加商店技能列表（前20个）
		storeSkillsList := make([]map[string]any, 0)
		for i, item := range storeItems {
			if i >= 20 {
				break
			}
			storeSkillsList = append(storeSkillsList, map[string]any{
				"name":    item.Name,
				"author":  item.Author,
				"summary": item.Summary,
			})
		}
		contextData["available_skills"] = storeSkillsList

		resp, err := a.bridge.Run(ctx, ai.WorkerRequest{
			Action: "plan",
			Payload: map[string]any{
				"goal":    goal,
				"context": contextData,
			},
		})
		if err == nil && resp.Status == "ok" {
			data := resp.Data
			planGoal, _ := data["goal"].(string)
			if planGoal == "" {
				planGoal = goal
			}

			stepsData, _ := data["steps"].([]any)
			records := make([]string, 0, len(stepsData)+1)
			records = append(records, fmt.Sprintf("用户提交目标: %s", goal))

			nextStep := "规划完成，等待用户确认"
			for _, stepAny := range stepsData {
				step, ok := stepAny.(map[string]any)
				if !ok {
					continue
				}
				label, _ := step["label"].(string)
				detail, _ := step["detail"].(string)
				if label != "" {
					records = append(records, fmt.Sprintf("步骤: %s — %s", label, detail))
					if nextStep == "规划完成，等待用户确认" {
						nextStep = fmt.Sprintf("下一步: %s", label)
					}
				}
			}

			result = AssistantTaskViewModel{
				ID:             fmt.Sprintf("task-%d", time.Now().UnixMilli()),
				Request:        goal,
				Status:         "planning",
				NextStep:       nextStep,
				Summary:        fmt.Sprintf("已为「%s」生成执行计划，共 %d 个步骤。", planGoal, len(stepsData)),
				Recommendation: "AI 助手已生成执行计划，请查看步骤并确认执行。",
				Reason:         "",
				Records:        records,
			}
		}
	}

	if result.ID == "" {
		result = AssistantTaskViewModel{
			ID:             fmt.Sprintf("task-%d", time.Now().UnixMilli()),
			Request:        goal,
			Status:         "planning",
			NextStep:       "正在分析目标并规划技能安装步骤",
			Summary:        fmt.Sprintf("正在为「%s」规划技能安装方案。", goal),
			Recommendation: "AI 助手正在分析您的目标，稍后将给出技能推荐。",
			Reason:         "",
			Records:        []string{fmt.Sprintf("用户提交目标: %s", goal), "状态: 规划中"},
		}
	}

	a.assistantMu.Lock()
	a.activeTask = &result
	a.assistantMu.Unlock()

	if a.taskRepo != nil {
		now := time.Now()
		_ = a.taskRepo.Put(context.Background(), domain.Task{
			ID:            result.ID,
			TaskType:      "ai_assistant",
			TriggerSource: goal,
			Status:        "planning",
			StartedAt:     &now,
		})
	}

	return result
}

/** 推进 AI 助手任务到下一阶段 */
func (a *App) AdvanceAssistantTask(taskID string, action string) AssistantTaskViewModel {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()

	if a.activeTask == nil || a.activeTask.ID != taskID {
		return AssistantTaskViewModel{
			ID:     "assistant-idle",
			Status: "queued",
			NextStep: "等待用户输入目标",
			Summary: "AI 助手待命中，输入目标即可开始规划。",
			Recommendation: "输入一个目标，让 AI 帮你规划技能安装与修复。",
			Records: []string{},
		}
	}

	task := a.activeTask
	records := make([]string, len(task.Records))
	copy(records, task.Records)

	switch action {
	case "resolve":
		task.Status = "resolving"
		task.NextStep = "正在解析依赖关系"
		records = append(records, "开始解析依赖关系...")

		if a.bridge != nil {
			resp, err := a.bridge.Run(context.Background(), ai.WorkerRequest{
				Action: "resolve",
				Payload: map[string]any{
					"goal":  task.Request,
					"plan":  task.Records,
				},
			})
			if err == nil && resp.Status == "ok" {
				records = append(records, "依赖解析完成")
				if detail, ok := resp.Data["summary"].(string); ok && detail != "" {
					records = append(records, fmt.Sprintf("解析结果: %s", detail))
				}
				task.NextStep = "依赖解析完成，等待执行确认"
				task.Summary = fmt.Sprintf("已为「%s」完成依赖解析，可以开始执行。", task.Request)
			} else {
				records = append(records, "依赖解析使用本地回退方案")
				task.NextStep = "依赖解析完成（本地模式），等待执行确认"
			}
		} else {
			records = append(records, "依赖解析完成（本地模式）")
			task.NextStep = "依赖解析完成，等待执行确认"
		}

	case "execute":
		task.Status = "executing"
		task.NextStep = "正在执行安装操作"
		records = append(records, "开始执行安装操作...")

		skillNames := a.extractSkillNamesFromRecords(records)
		agentID := a.findAgentIDForTask(task)
		executedCount := 0
		for _, name := range skillNames {
			installResult := a.InstallSkill(agentID, name, "")
			if installResult == "ok" {
				executedCount++
				records = append(records, fmt.Sprintf("✓ 技能 %s 安装成功", name))
			} else {
				records = append(records, fmt.Sprintf("✗ 技能 %s 安装失败: %s", name, installResult))
			}
		}

		if executedCount > 0 {
			task.NextStep = fmt.Sprintf("已安装 %d 个技能，等待验证", executedCount)
			task.Summary = fmt.Sprintf("已为「%s」安装 %d 个技能。", task.Request, executedCount)
		} else {
			task.NextStep = "执行完成（无技能需要安装），等待验证"
			task.Summary = fmt.Sprintf("已为「%s」完成执行步骤。", task.Request)
		}

	case "verify":
		task.Status = "verifying"
		task.NextStep = "正在验证安装结果"
		records = append(records, "开始验证安装结果...")

		if a.registry != nil {
			installs, err := a.registry.DiscoverAll(context.Background())
			if err == nil {
				totalSkills := 0
				for _, install := range installs {
					if install.Health == agents.HealthReady {
						skills, skillErr := a.registry.ListInstalledSkills(context.Background(), install)
						if skillErr == nil {
							totalSkills += len(skills)
						}
					}
				}
				records = append(records, fmt.Sprintf("验证完成: 当前共 %d 个技能已安装", totalSkills))
			} else {
				records = append(records, "验证完成: 无法检查代理状态")
			}
		} else {
			records = append(records, "验证完成: 代理注册表未初始化")
		}

		task.NextStep = "验证完成，生成报告"

	case "report":
		task.Status = "completed"
		task.NextStep = "任务已完成"
		records = append(records, "生成执行报告...")

		if a.bridge != nil {
			resp, err := a.bridge.Run(context.Background(), ai.WorkerRequest{
				Action: "report",
				Payload: map[string]any{
					"goal":    task.Request,
					"records": records,
				},
			})
			if err == nil && resp.Status == "ok" {
				if summary, ok := resp.Data["summary"].(string); ok && summary != "" {
					task.Summary = summary
					records = append(records, fmt.Sprintf("报告: %s", summary))
				}
			} else {
				task.Summary = fmt.Sprintf("任务「%s」已完成。", task.Request)
				records = append(records, fmt.Sprintf("任务「%s」已完成。", task.Request))
			}
		} else {
			task.Summary = fmt.Sprintf("任务「%s」已完成。", task.Request)
			records = append(records, fmt.Sprintf("任务「%s」已完成。", task.Request))
		}

		task.Recommendation = "任务已完成，可以开始新的目标。"

	case "cancel":
		task.Status = "cancelled"
		task.NextStep = "任务已取消"
		task.Summary = fmt.Sprintf("任务「%s」已取消。", task.Request)
		records = append(records, "用户取消了任务")

	default:
		records = append(records, fmt.Sprintf("未知操作: %s", action))
	}

	task.Records = records

	if a.taskRepo != nil {
		now := time.Now()
		var finishedAt *time.Time
		if task.Status == "completed" || task.Status == "cancelled" || task.Status == "failed" {
			finishedAt = &now
		}
		_ = a.taskRepo.Put(context.Background(), domain.Task{
			ID:             task.ID,
			TaskType:       "ai_assistant",
			TriggerSource:  task.Request,
			Status:         task.Status,
			ResultSummary:  task.Summary,
			StartedAt:      &now,
			FinishedAt:     finishedAt,
		})
	}

	return *task
}

/** 为 AI 助手任务查找合适的代理 ID，优先使用项目绑定的代理 */
func (a *App) findAgentIDForTask(task *AssistantTaskViewModel) string {
	a.projectsMu.RLock()
	for _, p := range a.projects {
		if p.BoundAgentID != "" {
			a.projectsMu.RUnlock()
			return p.BoundAgentID
		}
	}
	a.projectsMu.RUnlock()

	if a.registry != nil {
		installs, err := a.registry.DiscoverAll(context.Background())
		if err == nil {
			for _, install := range installs {
				if install.Health == agents.HealthReady {
					return install.AgentID
				}
			}
		}
	}

	return ""
}

/** 从执行记录中提取技能名称 */
func (a *App) extractSkillNamesFromRecords(records []string) []string {
	skillSet := make(map[string]bool)
	for _, record := range records {
		if strings.HasPrefix(record, "步骤:") {
			parts := strings.SplitN(record, "—", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(strings.TrimPrefix(parts[0], "步骤:"))
				if name != "" && !strings.Contains(name, "规划") && !strings.Contains(name, "分析") {
					skillSet[name] = true
				}
			}
		}
	}

	a.catalogMu.RLock()
	for _, item := range a.catalogItems {
		for _, record := range records {
			if strings.Contains(record, item.Name) {
				skillSet[item.Name] = true
			}
		}
	}
	a.catalogMu.RUnlock()

	names := make([]string, 0, len(skillSet))
	for name := range skillSet {
		names = append(names, name)
	}
	return names
}

/** 重置 AI 助手任务 */
func (a *App) ResetAssistantTask() AssistantTaskViewModel {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()

	a.activeTask = nil

	return AssistantTaskViewModel{
		ID:             "assistant-idle",
		Request:        "",
		Status:         "queued",
		NextStep:       "等待用户输入目标",
		Summary:        "AI 助手待命中，输入目标即可开始规划。",
		Recommendation: "输入一个目标，让 AI 帮你规划技能安装与修复。",
		Records:        []string{},
	}
}

/** 获取指定代理的技能列表详情 */
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
func (a *App) GetRecentActivities(limit int) []ActivityItem {
	result := make([]ActivityItem, 0)
	if a.registry == nil {
		return result
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return result
	}

	count := 0
	for _, install := range installs {
		if install.Health != agents.HealthReady {
			continue
		}
		skillNames, listErr := a.registry.ListInstalledSkills(context.Background(), install)
		if listErr != nil {
			continue
		}
		for _, name := range skillNames {
			if count >= limit {
				break
			}
			skillDir := filepath.Join(install.SkillsPath, name)
			modTime := ""
			if info, statErr := os.Stat(skillDir); statErr == nil {
				modTime = info.ModTime().Format("2006-01-02 15:04")
			}
			result = append(result, ActivityItem{
				ID:        fmt.Sprintf("activity-%s-%s", install.AgentID, name),
				Type:      "install",
				AgentID:   install.AgentID,
				SkillName: name,
				Time:      modTime,
				Status:    "success",
				Detail:    fmt.Sprintf("在 %s 中发现技能 %s", install.DisplayName, name),
			})
			count++
		}
	}
	return result
}

/** 返回系统健康状态 */
func (a *App) GetSystemHealthStatus() SystemHealthStatus {
	status := SystemHealthStatus{
		OverallStatus: "ok",
		AgentHealth:   make([]AgentHealthItem, 0),
		CheckedAt:     time.Now().Format("2006-01-02 15:04:05"),
	}

	if a.registry == nil {
		status.OverallStatus = "error"
		return status
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		status.OverallStatus = "error"
		return status
	}

	seenAgents := make(map[string]bool)
	hasWarning := false
	for _, install := range installs {
		if seenAgents[install.AgentID] {
			continue
		}
		seenAgents[install.AgentID] = true

		item := AgentHealthItem{
			AgentID: install.AgentID,
			Name:    install.DisplayName,
			Status:  string(install.Health),
			Detail:  install.DisplayName,
		}

		switch install.Health {
		case agents.HealthReady:
			item.Status = "healthy"
			item.Detail = "运行正常"
		case agents.HealthInstalledButSkillPathEmpty:
			item.Status = "warning"
			item.Detail = "技能目录为空"
			hasWarning = true
		case agents.HealthInstalledButSkillPathMissing:
			item.Status = "warning"
			item.Detail = "缺少技能目录"
			hasWarning = true
		case agents.HealthInstalledButUnreadable:
			item.Status = "error"
			item.Detail = "读取异常"
			hasWarning = true
		case agents.HealthNotInstalled:
			item.Status = "not_installed"
			item.Detail = "未安装"
		}

		status.AgentHealth = append(status.AgentHealth, item)
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(home, &stat); err == nil {
			totalGB := float64(stat.Blocks) * float64(stat.Bsize) / 1e9
			freeGB := float64(stat.Bavail) * float64(stat.Bsize) / 1e9
			status.DiskSpace = DiskSpaceInfo{
				TotalGB: totalGB,
				FreeGB:  freeGB,
				UsedPct: (totalGB - freeGB) / totalGB * 100,
			}
		}
	}

	if hasWarning {
		status.OverallStatus = "warning"
	}
	return status
}

/** 返回推荐操作列表 */
func (a *App) GetRecommendedActions() []RecommendedAction {
	result := make([]RecommendedAction, 0)
	if a.registry == nil {
		return result
	}

	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return result
	}

	seenAgents := make(map[string]bool)
	for _, install := range installs {
		if seenAgents[install.AgentID] {
			continue
		}
		seenAgents[install.AgentID] = true

		switch install.Health {
		case agents.HealthInstalledButSkillPathMissing:
			result = append(result, RecommendedAction{
				ID:       fmt.Sprintf("repair-%s", install.AgentID),
				Priority: "high",
				Action:   fmt.Sprintf("修复 %s 的技能目录", install.DisplayName),
				Reason:   "技能目录缺失，无法安装或管理技能",
				Type:     "repair",
			})
		case agents.HealthInstalledButSkillPathEmpty:
			result = append(result, RecommendedAction{
				ID:       fmt.Sprintf("setup-%s", install.AgentID),
				Priority: "medium",
				Action:   fmt.Sprintf("为 %s 安装推荐技能", install.DisplayName),
				Reason:   "技能目录为空，建议安装基础技能",
				Type:     "install",
			})
		case agents.HealthInstalledButUnreadable:
			result = append(result, RecommendedAction{
				ID:       fmt.Sprintf("fix-%s", install.AgentID),
				Priority: "high",
				Action:   fmt.Sprintf("修复 %s 的读取权限", install.DisplayName),
				Reason:   "无法读取代理目录，可能存在权限问题",
				Type:     "repair",
			})
		}
	}
	return result
}

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
func (a *App) GetCatalogSources() []CatalogSourceViewModel {
	a.catalogMu.RLock()
	defer a.catalogMu.RUnlock()

	result := make([]CatalogSourceViewModel, 0, len(a.catalogSources))
	result = append(result, a.catalogSources...)
	return result
}

/** 同步商店源，从 GitHub 仓库获取技能列表 */
func (a *App) SyncCatalogSource(sourceID string) SyncResultViewModel {
	a.catalogMu.Lock()
	defer a.catalogMu.Unlock()

	var target *CatalogSourceViewModel
	for i := range a.catalogSources {
		if a.catalogSources[i].ID == sourceID {
			target = &a.catalogSources[i]
			break
		}
	}
	if target == nil {
		return SyncResultViewModel{
			SourceID: sourceID,
			Success: false,
			NewSkills: 0,
			UpdatedSkills: 0,
			Errors: []string{"来源不存在"},
		}
	}

	skills, err := fetchGitHubSkills(target.URL)
	if err != nil {
		target.LastSyncStatus = "failed"

		if a.catalogSrcRepo != nil {
			_ = a.catalogSrcRepo.Put(context.Background(), catalogSourceVMToDomain(*target))
		}

		return SyncResultViewModel{
			SourceID: sourceID,
			Success: false,
			NewSkills: 0,
			UpdatedSkills: 0,
			Errors: []string{err.Error()},
		}
	}

	now := time.Now().Format("2006-01-02 15:04")
	target.LastSyncedAt = now
	target.LastSyncStatus = "success"

	existingMap := make(map[string]bool)
	for _, item := range a.catalogItems {
		if item.Source == target.Name {
			existingMap[item.Name] = true
		}
	}

	newCount := 0
	for _, skill := range skills {
		if existingMap[skill.Name] {
			continue
		}
		newCount++

		installed := a.isSkillInstalled(skill.Name)
		status := "available"
		if installed {
			status = "installed"
		}

		a.catalogItems = append(a.catalogItems, StoreItemViewModel{
			ID:             fmt.Sprintf("%s-%s", sourceID, skill.Name),
			Name:           skill.Name,
			Author:         skill.Author,
			Source:         target.Name,
			Status:         status,
			Summary:        skill.Description,
			Installs:       fmt.Sprintf("来自 %s", target.Name),
			Impact:         "技能将安装到指定代理的技能目录",
			Compatibility:  skill.SupportedAgents,
			Homepage:       skill.Homepage,
			LocalCachePath: skill.CachePath,
		})
	}

	target.SkillCount = len(skills)

	if a.catalogSrcRepo != nil {
		_ = a.catalogSrcRepo.Put(context.Background(), catalogSourceVMToDomain(*target))

		ctx := context.Background()
		domainSkills := make([]domain.CatalogSkill, 0, len(skills))
		for _, skill := range skills {
			domainSkills = append(domainSkills, domain.CatalogSkill{
				ID:              fmt.Sprintf("%s-%s", sourceID, skill.Name),
				SourceID:        sourceID,
				Name:            skill.Name,
				Version:         "latest",
				Author:          skill.Author,
				Description:     skill.Description,
				Homepage:        skill.Homepage,
				SupportedAgents: skill.SupportedAgents,
			})
		}
		_ = a.catalogSkillRepo.ReplaceBySource(ctx, sourceID, domainSkills)
	}

	return SyncResultViewModel{
		SourceID:      sourceID,
		Success:       true,
		NewSkills:     newCount,
		UpdatedSkills: 0,
		Errors:        make([]string, 0),
	}
}

/** 同步所有已启用的商店源 */
func (a *App) SyncAllSources() []SyncResultViewModel {
	a.catalogMu.RLock()
	sourceIDs := make([]string, 0)
	for _, src := range a.catalogSources {
		if src.Enabled {
			sourceIDs = append(sourceIDs, src.ID)
		}
	}
	a.catalogMu.RUnlock()

	results := make([]SyncResultViewModel, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		results = append(results, a.SyncCatalogSource(id))
	}
	return results
}

/** 添加自定义商店源 */
func (a *App) AddCatalogSource(name string, url string) CatalogSourceViewModel {
	a.catalogMu.Lock()
	defer a.catalogMu.Unlock()

	id := fmt.Sprintf("custom-%d", time.Now().UnixMilli())
	source := CatalogSourceViewModel{
		ID:             id,
		Name:           name,
		URL:            url,
		IsBuiltin:      false,
		Enabled:        true,
		LastSyncedAt:   "",
		LastSyncStatus: "",
		SkillCount:     0,
	}
	a.catalogSources = append(a.catalogSources, source)

	if a.catalogSrcRepo != nil {
		_ = a.catalogSrcRepo.Put(context.Background(), catalogSourceVMToDomain(source))
	}

	return source
}

/** 移除商店源（内置源不可移除） */
func (a *App) RemoveCatalogSource(sourceID string) string {
	a.catalogMu.Lock()
	defer a.catalogMu.Unlock()

	for i, src := range a.catalogSources {
		if src.ID == sourceID {
			if src.IsBuiltin {
				return "error: cannot remove builtin source"
			}
			a.catalogSources = append(a.catalogSources[:i], a.catalogSources[i+1:]...)

			filtered := make([]StoreItemViewModel, 0)
			for _, item := range a.catalogItems {
				if item.Source != src.Name {
					filtered = append(filtered, item)
				}
			}
			a.catalogItems = filtered

			if a.catalogSrcRepo != nil {
				_ = a.catalogSkillRepo.DeleteBySource(context.Background(), sourceID)
				_ = a.catalogSrcRepo.Delete(context.Background(), sourceID)
			}

			return "ok"
		}
	}
	return "error: source not found"
}

/** 检查技能是否已安装 */
func (a *App) isSkillInstalled(skillName string) bool {
	if a.registry == nil {
		return false
	}
	installs, err := a.registry.DiscoverAll(context.Background())
	if err != nil {
		return false
	}
	for _, install := range installs {
		skillNames, err := a.registry.ListInstalledSkills(context.Background(), install)
		if err != nil {
			continue
		}
		for _, name := range skillNames {
			if name == skillName {
				return true
			}
		}
	}
	return false
}

/** 解释商店技能的用途，从远程仓库获取 README 内容 */
func (a *App) ExplainStoreSkill(sourceName string, skillName string) SkillExplanationViewModel {
	result := SkillExplanationViewModel{
		AgentID:   "store",
		SkillName: skillName,
		Found:     false,
		Files:     make([]string, 0),
	}

	a.catalogMu.RLock()
	var targetItem *StoreItemViewModel
	for i := range a.catalogItems {
		if a.catalogItems[i].Source == sourceName && a.catalogItems[i].Name == skillName {
			targetItem = &a.catalogItems[i]
			break
		}
	}
	a.catalogMu.RUnlock()

	if targetItem == nil {
		return result
	}

	result.Found = true
	result.AgentName = sourceName
	result.SkillPath = targetItem.Homepage

	if targetItem.Homepage != "" {
		repo := parseGitHubRepo(targetItem.Homepage)
		if repo != "" {
			branches := []string{"main", "master"}
			filenames := []string{"SKILL.md", "README.md", "readme.md"}
		outer:
			for _, branch := range branches {
				for _, filename := range filenames {
					rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/skills/%s/%s", repo, branch, skillName, filename)
					if content, err := fetchRawContent(rawURL); err == nil && content != "" {
						if len(content) > 8000 {
							content = content[:8000] + "\n...(内容过长，已截断)"
						}
						result.ReadmeContent = content
						result.ReadmeFile = filename
						break outer
					}
				}
			}
		}
	}

	if result.ReadmeContent == "" {
		result.ReadmeContent = targetItem.Summary
		result.ReadmeFile = "summary"
	}

	return result
}

/** 获取远程文件内容 */
func fetchRawContent(rawURL string) (string, error) {
	resp, err := httpGetWithTimeout(rawURL, 10*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16384))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

/** 返回技能组列表 */
func (a *App) GetSkillGroups() []SkillGroupViewModel {
	a.projectsMu.RLock()
	defer a.projectsMu.RUnlock()

	result := make([]SkillGroupViewModel, 0, len(a.skillGroups))
	result = append(result, a.skillGroups...)
	return result
}

/** 创建技能组，支持同时指定技能列表和绑定代理 */
func (a *App) CreateSkillGroup(name string, description string, skillNames string, agentID string) SkillGroupViewModel {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	var names []string
	if skillNames != "" {
		names = strings.Split(skillNames, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
	} else {
		names = make([]string, 0)
	}

	agentName := ""
	if agentID != "" && a.registry != nil {
		installs, err := a.registry.DiscoverAll(context.Background())
		if err == nil {
			for _, install := range installs {
				if install.AgentID == agentID || install.DisplayName == agentID {
					agentName = install.DisplayName
					break
				}
			}
		}
	}

	group := SkillGroupViewModel{
		ID:             fmt.Sprintf("sg-%d", time.Now().UnixMilli()),
		Name:           name,
		Description:    description,
		SourceType:     "manual",
		SkillCount:     len(names),
		ProjectCount:   0,
		CreatedAt:      time.Now().Format("2006-01-02 15:04"),
		SkillNames:     names,
		BoundAgentID:   agentID,
		BoundAgentName: agentName,
	}

	a.skillGroups = append(a.skillGroups, group)

	if a.skillGroupsRepo != nil {
		_ = a.skillGroupsRepo.Put(context.Background(), skillGroupVMToDomain(group))
		for _, skillName := range names {
			_ = a.skillGroupsRepo.PutSkillGroupSkill(context.Background(), domain.SkillGroupSkill{
				SkillGroupID: group.ID,
				SkillID:      skillName,
				Required:     true,
				Priority:     0,
				Reason:       "",
			})
		}
	}

	return group
}

/** 删除技能组 */
func (a *App) DeleteSkillGroup(groupID string) string {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	for i, g := range a.skillGroups {
		if g.ID == groupID {
			a.skillGroups = append(a.skillGroups[:i], a.skillGroups[i+1:]...)

			if a.skillGroupsRepo != nil {
				_ = a.skillGroupsRepo.Delete(context.Background(), groupID)
			}

			return "ok"
		}
	}
	return "error: group not found"
}

/** 为技能组添加技能 */
func (a *App) AddSkillToGroup(groupID string, skillName string) string {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	for i := range a.skillGroups {
		if a.skillGroups[i].ID == groupID {
			for _, existing := range a.skillGroups[i].SkillNames {
				if existing == skillName {
					return "error: skill already in group"
				}
			}
			a.skillGroups[i].SkillNames = append(a.skillGroups[i].SkillNames, skillName)
			a.skillGroups[i].SkillCount = len(a.skillGroups[i].SkillNames)

			if a.skillGroupsRepo != nil {
				_ = a.skillGroupsRepo.PutSkillGroupSkill(context.Background(), domain.SkillGroupSkill{
					SkillGroupID: groupID,
					SkillID:      skillName,
					Required:     true,
				})
				_ = a.skillGroupsRepo.Put(context.Background(), skillGroupVMToDomain(a.skillGroups[i]))
			}

			return "ok"
		}
	}
	return "error: group not found"
}

/** 从技能组移除技能 */
func (a *App) RemoveSkillFromGroup(groupID string, skillName string) string {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	for i := range a.skillGroups {
		if a.skillGroups[i].ID == groupID {
			for j, name := range a.skillGroups[i].SkillNames {
				if name == skillName {
					a.skillGroups[i].SkillNames = append(a.skillGroups[i].SkillNames[:j], a.skillGroups[i].SkillNames[j+1:]...)
					a.skillGroups[i].SkillCount = len(a.skillGroups[i].SkillNames)

					if a.skillGroupsRepo != nil {
						_ = a.skillGroupsRepo.DeleteSkillGroupSkill(context.Background(), groupID, skillName)
						_ = a.skillGroupsRepo.Put(context.Background(), skillGroupVMToDomain(a.skillGroups[i]))
					}

					return "ok"
				}
			}
			return "error: skill not in group"
		}
	}
	return "error: group not found"
}

/** 返回任务历史列表 */
func (a *App) GetTaskHistory(limit int) []TaskHistoryItem {
	if a.taskRepo == nil {
		return []TaskHistoryItem{}
	}

	tasks, err := a.taskRepo.ListRecent(context.Background(), limit)
	if err != nil {
		return []TaskHistoryItem{}
	}

	items := make([]TaskHistoryItem, 0, len(tasks))
	for _, t := range tasks {
		startedAtStr := ""
		if t.StartedAt != nil {
			startedAtStr = t.StartedAt.Format("2006-01-02 15:04")
		}
		finishedAtStr := ""
		if t.FinishedAt != nil {
			finishedAtStr = t.FinishedAt.Format("2006-01-02 15:04")
		}
		items = append(items, TaskHistoryItem{
			ID:         t.ID,
			Goal:       t.TriggerSource,
			Status:     t.Status,
			StartedAt:  startedAtStr,
			FinishedAt: finishedAtStr,
			Summary:    t.ResultSummary,
		})
	}

	return items
}

/** 为项目生成技能协调计划，分析缺失、过期和异常技能 */
func (a *App) ReconcileProject(projectID string) ReconcilePlanViewModel {
	a.projectsMu.RLock()
	var project *ProjectViewModel
	for i := range a.projects {
		if a.projects[i].ID == projectID {
			project = &a.projects[i]
			break
		}
	}
	a.projectsMu.RUnlock()

	if project == nil {
		return ReconcilePlanViewModel{ProjectID: projectID, BlockReason: "项目不存在"}
	}

	var desiredSkills []skillgroups.DesiredSkill
	if project.BoundSkillGroup != "" {
		a.projectsMu.RLock()
		var group *SkillGroupViewModel
		for i := range a.skillGroups {
			if a.skillGroups[i].Name == project.BoundSkillGroup {
				group = &a.skillGroups[i]
				break
			}
		}
		a.projectsMu.RUnlock()

		if group != nil {
			bindings := make([]domain.SkillGroupSkill, 0, len(group.SkillNames))
			for _, name := range group.SkillNames {
				bindings = append(bindings, domain.SkillGroupSkill{
					SkillGroupID: group.ID,
					SkillID:      name,
					Required:     true,
				})
			}

			sgSvc := skillgroups.Service{}
			desired, err := sgSvc.DesiredSkills(skillGroupVMToDomain(*group), bindings)
			if err != nil {
				return ReconcilePlanViewModel{ProjectID: projectID, ProjectName: project.Name, BlockReason: err.Error()}
			}
			desiredSkills = desired
		}
	}

	var catalogSkills []domain.CatalogSkill
	if a.catalogSkillRepo != nil {
		catalogSkills, _ = a.catalogSkillRepo.ListAll(context.Background())
	}

	var installedSkills []domain.InstalledSkill
	if a.registry != nil {
		installs, err := a.registry.DiscoverAll(context.Background())
		if err == nil {
			for _, install := range installs {
				if install.Health == agents.HealthReady {
					names, listErr := a.registry.ListInstalledSkills(context.Background(), install)
					if listErr == nil {
						for _, name := range names {
							installedSkills = append(installedSkills, domain.InstalledSkill{
								ID:            fmt.Sprintf("%s-%s", install.AgentID, name),
								SkillID:       name,
								AgentID:       install.AgentID,
								Version:       "installed",
								InstallState:  "installed",
								InstallPath:   filepath.Join(install.SkillsPath, name),
							})
						}
					}
				}
			}
		}
	}

	reconcileSvc := reconcile.Service{}
	plan, err := reconcileSvc.Plan(desiredSkills, catalogSkills, installedSkills)
	if err != nil {
		return ReconcilePlanViewModel{ProjectID: projectID, ProjectName: project.Name, BlockReason: err.Error()}
	}

	result := ReconcilePlanViewModel{
		ProjectID:   projectID,
		ProjectName: project.Name,
		Install:     make([]ReconcileActionItem, 0),
		Update:      make([]ReconcileActionItem, 0),
		Repair:      make([]ReconcileActionItem, 0),
		BlockReason: plan.BlockReason,
	}

	for _, action := range plan.Install {
		result.Install = append(result.Install, ReconcileActionItem{SkillID: action.SkillID, Version: action.Version, Name: action.SkillID})
	}
	for _, action := range plan.Update {
		result.Update = append(result.Update, ReconcileActionItem{SkillID: action.SkillID, Version: action.Version, Name: action.SkillID})
	}
	for _, action := range plan.Repair {
		result.Repair = append(result.Repair, ReconcileActionItem{SkillID: action.SkillID, Version: action.Version, Name: action.SkillID})
	}

	return result
}

/** 执行协调计划，安装缺失、更新过期和修复异常技能 */
func (a *App) ExecuteReconcilePlan(projectID string, planJSON string) string {
	var plan ReconcilePlanViewModel
	if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
		return fmt.Sprintf("error: invalid plan JSON: %s", err)
	}

	a.projectsMu.RLock()
	var project *ProjectViewModel
	for i := range a.projects {
		if a.projects[i].ID == projectID {
			project = &a.projects[i]
			break
		}
	}
	a.projectsMu.RUnlock()

	if project == nil {
		return "error: project not found"
	}

	agentID := project.BoundAgentID
	if agentID == "" {
		return "error: project has no bound agent"
	}

	errors := make([]string, 0)

	for _, action := range plan.Install {
		if err := a.InstallSkill(agentID, action.Name, ""); err != "ok" {
			errors = append(errors, fmt.Sprintf("install %s: %s", action.Name, err))
		}
	}

	for _, action := range plan.Update {
		if err := a.UpdateSkill(agentID, action.Name, ""); err != "ok" {
			errors = append(errors, fmt.Sprintf("update %s: %s", action.Name, err))
		}
	}

	for _, action := range plan.Repair {
		a.RepairAgent(agentID)
		if err := a.UpdateSkill(agentID, action.Name, ""); err != "ok" {
			errors = append(errors, fmt.Sprintf("repair %s: %s", action.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Sprintf("completed with %d errors: %s", len(errors), strings.Join(errors, "; "))
	}
	return "ok"
}

/** 返回建议提示模板 */
func (a *App) GetSuggestionTemplates() []SuggestionTemplate {
	return []SuggestionTemplate{
		{ID: "tpl-1", Category: "Web 开发", Title: "搭建 Web 开发环境", Description: "安装前端开发相关的技能", PromptTemplate: "帮我搭建一个 Web 开发项目，安装前端开发相关的技能", IsBuiltin: true},
		{ID: "tpl-2", Category: "测试", Title: "添加测试技能", Description: "安装单元测试和集成测试技能", PromptTemplate: "为我的项目添加测试技能，包括单元测试和集成测试", IsBuiltin: true},
		{ID: "tpl-3", Category: "数据分析", Title: "数据分析环境", Description: "安装数据处理和分析技能", PromptTemplate: "帮我配置数据分析环境，安装数据处理和可视化相关技能", IsBuiltin: true},
		{ID: "tpl-4", Category: "优化", Title: "优化当前配置", Description: "检查并优化现有技能配置", PromptTemplate: "检查我当前的技能配置，推荐优化方案", IsBuiltin: true},
		{ID: "tpl-5", Category: "DevOps", Title: "DevOps 工具链", Description: "安装 CI/CD 和部署相关技能", PromptTemplate: "帮我配置 DevOps 工具链，安装 CI/CD 和部署相关技能", IsBuiltin: true},
	}
}

/** 返回应用详细信息 */
func (a *App) GetAppInfoFull() AppInfoViewModel {
	return AppInfoViewModel{
		Name:      a.Name,
		Version:   a.Version,
		BuildTime: a.startedAt.Format("2006-01-02 15:04:05"),
		GoVersion: goRuntime.Version(),
		OS:        goRuntime.GOOS,
		Arch:      goRuntime.GOARCH,
	}
}

/** 返回通用设置 */
func (a *App) GetGeneralSettings() GeneralSettingsViewModel {
	defaults := GeneralSettingsViewModel{
		Theme:                "light",
		FontSize:             "medium",
		NotificationsEnabled: true,
		Language:             "zh-CN",
	}

	if a.settingsRepo == nil {
		return defaults
	}

	value, err := a.settingsRepo.Get(context.Background(), "general")
	if err != nil || value == "" {
		return defaults
	}

	var settings GeneralSettingsViewModel
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return defaults
	}
	return settings
}

/** 保存通用设置 */
func (a *App) SaveGeneralSettings(settings GeneralSettingsViewModel) string {
	if a.settingsRepo == nil {
		return "ok"
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	if err := a.settingsRepo.Put(context.Background(), "general", string(data)); err != nil {
		return fmt.Sprintf("error: %s", err)
	}
	return "ok"
}

/** 返回自动化设置 */
func (a *App) GetAutomationSettings() AutomationSettingsViewModel {
	defaults := AutomationSettingsViewModel{
		AutoSyncCatalog:      true,
		AutoCheckUpdates:     true,
		AutoApplySkillGroups: false,
		HealthCheckSchedule:  "daily",
		AutoRepair:           false,
	}

	if a.settingsRepo == nil {
		return defaults
	}

	value, err := a.settingsRepo.Get(context.Background(), "automation")
	if err != nil || value == "" {
		return defaults
	}

	var settings AutomationSettingsViewModel
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return defaults
	}
	return settings
}

/** 保存自动化设置 */
func (a *App) SaveAutomationSettings(settings AutomationSettingsViewModel) string {
	if a.settingsRepo == nil {
		return "ok"
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	if err := a.settingsRepo.Put(context.Background(), "automation", string(data)); err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	if a.scheduler != nil {
		a.scheduler.ApplySettings(settings)
	}

	return "ok"
}

/** 获取 AI 设置 */
func (a *App) GetAISettings() AISettingsViewModel {
	defaults := AISettingsViewModel{
		Provider: "none",
		Model:    "",
		APIKey:   "",
		BaseURL:  "",
	}

	if a.settingsRepo == nil {
		return defaults
	}

	val, err := a.settingsRepo.Get(context.Background(), "ai")
	if err != nil || val == "" {
		return defaults
	}

	var settings AISettingsViewModel
	if err := json.Unmarshal([]byte(val), &settings); err != nil {
		return defaults
	}
	return settings
}

/** 保存 AI 设置，同时更新 bridge 实例 */
func (a *App) SaveAISettings(settings AISettingsViewModel) string {
	if a.settingsRepo != nil {
		data, err := json.Marshal(settings)
		if err != nil {
			return fmt.Sprintf("error: %s", err)
		}
		if err := a.settingsRepo.Put(context.Background(), "ai", string(data)); err != nil {
			return fmt.Sprintf("error: %s", err)
		}
	}

	if a.bridge != nil {
		if lb, ok := a.bridge.(*ai.LocalBridge); ok {
			lb.UpdateConfig(settings.Provider, settings.Model)
		}
	}

	return "ok"
}

/** 返回日志条目列表 */
func (a *App) GetLogs(level string, limit int) []LogEntryViewModel {
	return []LogEntryViewModel{}
}

/** 批量更新技能 */
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
type StatusTone = string

/** 根据健康状态返回发现优先级排名 */
func healthRank(health agents.HealthStatus) int {
	switch health {
	case agents.HealthReady:
		return 4
	case agents.HealthInstalledButSkillPathEmpty:
		return 3
	case agents.HealthInstalledButSkillPathMissing:
		return 2
	case agents.HealthInstalledButUnreadable:
		return 1
	case agents.HealthNotInstalled:
		return 0
	default:
		return -1
	}
}

/** GitHub 技能条目，从仓库目录结构或 README 解析 */
type gitHubSkill struct {
	Name            string
	Author          string
	Description     string
	Homepage        string
	SupportedAgents []string
	CachePath       string
}

/** GitHub API Contents 目录条目 */
type gitHubContentEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

/** 从 GitHub 仓库获取技能列表，自动检测仓库类型 */
func fetchGitHubSkills(repoURL string) ([]gitHubSkill, error) {
	repo := parseGitHubRepo(repoURL)
	if repo == "" {
		return nil, fmt.Errorf("invalid GitHub URL: %s", repoURL)
	}

	skills, err := fetchSkillsFromDirectory(repo)
	if err == nil && len(skills) > 0 {
		return skills, nil
	}

	skills, err = fetchSkillsFromReadme(repo)
	if err == nil && len(skills) > 0 {
		return skills, nil
	}

	if err != nil {
		return nil, err
	}

	return make([]gitHubSkill, 0), nil
}

/** 从仓库的 skills/ 目录获取技能列表 */
func fetchSkillsFromDirectory(repo string) ([]gitHubSkill, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/skills", repo)
	resp, err := httpGetWithTimeout(apiURL, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills directory: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API 速率限制，请稍后重试")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("no skills/ directory (status %d)", resp.StatusCode)
	}

	var entries []gitHubContentEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to parse directory listing: %v", err)
	}

	skills := make([]gitHubSkill, 0)
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}

		skill := gitHubSkill{
			Name:            entry.Name,
			Author:          repo,
			Description:     fmt.Sprintf("来自 %s 的技能", entry.Name),
			Homepage:        fmt.Sprintf("https://github.com/%s/tree/main/skills/%s", repo, entry.Name),
			SupportedAgents: []string{"Claude Code", "Codex", "Trae"},
		}

		desc := fetchSkillDescription(repo, entry.Name)
		if desc != "" {
			skill.Description = desc
		}

		cachePath := cacheSkillFiles(repo, entry.Name)
		if cachePath != "" {
			skill.CachePath = cachePath
		}

		skills = append(skills, skill)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

/** 获取技能描述，依次尝试多个分支和文件名 */
func fetchSkillDescription(repo string, skillName string) string {
	branches := []string{"main", "master"}
	filenames := []string{"SKILL.md", "README.md", "readme.md"}

	for _, branch := range branches {
		for _, filename := range filenames {
			rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/skills/%s/%s", repo, branch, skillName, filename)
			if desc, err := fetchReadmeDescription(rawURL); err == nil && desc != "" {
				return desc
			}
		}
	}
	return ""
}

/** 从仓库 README 解析 awesome-list 格式的技能列表，自动尝试多个分支名 */
func fetchSkillsFromReadme(repo string) ([]gitHubSkill, error) {
	branches := []string{"main", "master"}
	filenames := []string{"README.md", "readme.md", "Readme.md"}

	for _, branch := range branches {
		for _, filename := range filenames {
			rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, branch, filename)
			resp, err := httpGetWithTimeout(rawURL, 10*time.Second)
			if err != nil {
				continue
			}

			if resp.StatusCode == 200 {
				body, readErr := io.ReadAll(io.LimitReader(resp.Body, 65536))
				resp.Body.Close()
				if readErr != nil {
					continue
				}

				skills := parseAwesomeListReadme(string(body), repo)
				if len(skills) > 0 {
					return skills, nil
				}
				continue
			}

			resp.Body.Close()
		}
	}

	return nil, fmt.Errorf("no README found in %s (tried main/master branches)", repo)
}

/** 解析 awesome-list 格式的 README，提取技能条目 */
func parseAwesomeListReadme(content string, repo string) []gitHubSkill {
	skills := make([]gitHubSkill, 0)
	seen := make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(trimmed, "* ") {
			continue
		}

		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		trimmed = strings.TrimSpace(trimmed)

		name, desc, link := parseListItem(trimmed)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		homepage := link
		if homepage == "" {
			homepage = fmt.Sprintf("https://github.com/%s", repo)
		}

		if desc == "" {
			desc = fmt.Sprintf("来自 %s 的技能", repo)
		}

		supportedAgents := []string{"Claude Code"}
		if strings.Contains(strings.ToLower(name), "codex") || strings.Contains(strings.ToLower(desc), "codex") {
			supportedAgents = append(supportedAgents, "Codex")
		}
		if strings.Contains(strings.ToLower(name), "trae") || strings.Contains(strings.ToLower(desc), "trae") {
			supportedAgents = append(supportedAgents, "Trae")
		}
		if len(supportedAgents) == 1 {
			supportedAgents = []string{"Claude Code", "Codex", "Trae"}
		}

		skills = append(skills, gitHubSkill{
			Name:            name,
			Author:          repo,
			Description:     desc,
			Homepage:        homepage,
			SupportedAgents: supportedAgents,
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills
}

/** 解析 Markdown 列表项，提取名称、描述和链接 */
func parseListItem(item string) (name string, desc string, link string) {
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := linkRegex.FindStringSubmatch(item)
	if len(matches) >= 3 {
		name = matches[1]
		link = matches[2]
		rest := strings.TrimSpace(item[len(matches[0]):])
		if strings.HasPrefix(rest, " - ") {
			desc = strings.TrimPrefix(rest, " - ")
		} else if strings.HasPrefix(rest, " – ") {
			desc = strings.TrimPrefix(rest, " – ")
		} else if strings.HasPrefix(rest, ": ") {
			desc = strings.TrimPrefix(rest, ": ")
		} else if rest != "" {
			desc = rest
		}
	} else {
		if strings.Contains(item, " - ") {
			parts := strings.SplitN(item, " - ", 2)
			name = strings.TrimSpace(parts[0])
			desc = strings.TrimSpace(parts[1])
		} else if strings.Contains(item, ": ") {
			parts := strings.SplitN(item, ": ", 2)
			name = strings.TrimSpace(parts[0])
			desc = strings.TrimSpace(parts[1])
		} else {
			name = item
		}
	}

	name = strings.TrimSpace(strings.Trim(name, "**"))
	desc = strings.TrimSpace(desc)
	if len(desc) > 200 {
		desc = desc[:197] + "..."
	}

	return name, desc, link
}

/** 从 GitHub URL 解析仓库路径 (owner/repo) */
func parseGitHubRepo(url string) string {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "https://github.com/") {
		path := strings.TrimPrefix(url, "https://github.com/")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	if strings.HasPrefix(url, "git@github.com:") {
		path := strings.TrimPrefix(url, "git@github.com:")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return ""
}

/** 从 README/SKILL.md 获取技能描述（取第一段非标题文本） */
func fetchReadmeDescription(rawURL string) (string, error) {
	resp, err := httpGetWithTimeout(rawURL, 5*time.Second)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(body), "\n")
	var descLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(descLines) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		descLines = append(descLines, trimmed)
		if len(descLines) >= 3 {
			break
		}
	}

	if len(descLines) == 0 {
		return "", fmt.Errorf("no description found")
	}

	return strings.Join(descLines, " "), nil
}

/** 带超时的 HTTP GET 请求 */
func httpGetWithTimeout(url string, timeout time.Duration) (*http.Response, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	return client.Do(req)
}

/** 获取技能缓存目录路径 */
func getSkillCacheDir(repo string, skillName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	safeRepo := strings.ReplaceAll(repo, "/", "-")
	return filepath.Join(home, "Library", "Application Support", "agent-skills-manager", "skill-cache", safeRepo, skillName)
}

/** 从 GitHub 下载技能文件并缓存到本地目录，返回缓存路径 */
func cacheSkillFiles(repo string, skillName string) string {
	cacheDir := getSkillCacheDir(repo, skillName)
	if cacheDir == "" {
		return ""
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return ""
	}

	branches := []string{"main", "master"}
	var dirEntries []gitHubContentEntry

	for _, branch := range branches {
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/skills/%s?ref=%s", repo, skillName, branch)
		resp, err := httpGetWithTimeout(apiURL, 10*time.Second)
		if err != nil {
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(&dirEntries); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()
		break
	}

	if len(dirEntries) == 0 {
		return ""
	}

	for _, entry := range dirEntries {
		if entry.Type == "dir" {
			continue
		}

		for _, branch := range branches {
			rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/skills/%s/%s", repo, branch, skillName, entry.Name)
			resp, err := httpGetWithTimeout(rawURL, 10*time.Second)
			if err != nil {
				continue
			}
			if resp.StatusCode != 200 {
				resp.Body.Close()
				continue
			}

			filePath := filepath.Join(cacheDir, entry.Name)
			f, err := os.Create(filePath)
			if err != nil {
				resp.Body.Close()
				break
			}
			_, _ = io.Copy(f, io.LimitReader(resp.Body, 512*1024))
			f.Close()
			resp.Body.Close()
			break
		}
	}

	return cacheDir
}

package app

import (
	"context"
	"fmt"

	"github.com/caojun/agent-skills-manager/internal/agents"
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
		installs := a.getCachedInstalls()
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

	taskValue := "0 个任务"
	taskDetail := "暂无运行中的任务。"
	taskTone := "muted"

	a.assistantMu.Lock()
	if a.activeTask != nil {
		taskValue = fmt.Sprintf("1 个任务")
		taskDetail = a.activeTask.Summary
		if taskDetail == "" {
			taskDetail = a.activeTask.NextStep
		}
		switch a.activeTask.Status {
		case "completed":
			taskTone = "stable"
		case "failed", "blocked":
			taskTone = "critical"
		case "executing", "planning", "resolving", "verifying", "recovering":
			taskTone = "attention"
		default:
			taskTone = "muted"
		}
	}
	a.assistantMu.Unlock()

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
			ID: "health", Title: "任务状态", Value: taskValue,
			Detail: taskDetail, Tone: taskTone, Tag: "概况",
		},
	}

	return DashboardSnapshot{
		Title:      "本机技能与运行状态",
		Summary:    "查看本机 AI 代理、已安装技能和任务执行状态。",
		Spotlight:  fmt.Sprintf("已发现 %d 个代理，%d 个技能", agentCount, skillCount),
		Highlights: highlights,
		Tasks:      []DashboardTask{},
		Notes: []string{
			"安装技能前请先查看兼容性信息。",
			"每个项目只能绑定一个技能组。",
		},
	}
}

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

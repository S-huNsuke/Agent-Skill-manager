package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	goRuntime "runtime"
	"syscall"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
	"github.com/caojun/agent-skills-manager/internal/ai"
)

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
			Tone: func() string {
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
			lb.UpdateConfig(settings.Provider, settings.Model, settings.APIKey, settings.BaseURL)
		}
	}

	return "ok"
}

/** 返回日志条目列表 */
func (a *App) GetLogs(level string, limit int) []LogEntryViewModel {
	return []LogEntryViewModel{}
}

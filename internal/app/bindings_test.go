package app

import (
	"encoding/json"
	"testing"
)

/** 验证 GetDashboard 返回非空仪表盘快照 */
func TestGetDashboardReturnsStructuredSnapshot(t *testing.T) {
	app := &App{Name: defaultAppName, Version: defaultAppVersion}
	dashboard := app.GetDashboard()

	if dashboard.Title == "" {
		t.Error("dashboard title must not be empty")
	}
	if dashboard.Summary == "" {
		t.Error("dashboard summary must not be empty")
	}
}

/** 验证 GetAgents 在无 registry 时返回空列表 */
func TestGetAgentsReturnsAgentViewModels(t *testing.T) {
	app := &App{Name: defaultAppName, Version: defaultAppVersion}
	agents := app.GetAgents()

	if agents == nil {
		t.Error("agents must not be nil")
	}
}

/** 验证 GetSkills 返回技能视图模型列表 */
func TestGetSkillsReturnsSkillViewModels(t *testing.T) {
	app := &App{Name: defaultAppName, Version: defaultAppVersion}
	skills := app.GetSkills()

	if skills == nil {
		t.Error("skills must not be nil")
	}
}

/** 验证 GetStoreItems 返回商店条目视图模型列表 */
func TestGetStoreItemsReturnsStoreItemViewModels(t *testing.T) {
	app := &App{Name: defaultAppName, Version: defaultAppVersion}
	items := app.GetStoreItems()

	if items == nil {
		t.Error("store items must not be nil")
	}
}

/** 验证 GetProjects 返回项目视图模型列表 */
func TestGetProjectsReturnsProjectViewModels(t *testing.T) {
	app := &App{Name: defaultAppName, Version: defaultAppVersion}
	projects := app.GetProjects()

	if projects == nil {
		t.Error("projects must not be nil")
	}
}

/** 验证 GetAssistantTask 返回助手任务视图模型 */
func TestGetAssistantTaskReturnsAssistantTaskViewModel(t *testing.T) {
	app := &App{Name: defaultAppName, Version: defaultAppVersion}
	task := app.GetAssistantTask()

	if task.ID == "" {
		t.Error("assistant task id must not be empty")
	}
	if task.Status == "" {
		t.Error("assistant task status must not be empty")
	}
}

/** 验证 GetDiagnostics 返回诊断条目视图模型列表 */
func TestGetDiagnosticsReturnsDiagnosticItemViewModels(t *testing.T) {
	app := &App{Name: defaultAppName, Version: defaultAppVersion}
	diags := app.GetDiagnostics()

	if len(diags) == 0 {
		t.Error("diagnostics must not be empty")
	}
}

/** 验证所有视图模型均可 JSON 序列化 */
func TestViewModelsAreJsonSerializable(t *testing.T) {
	models := []any{
		DashboardSnapshot{},
		AgentViewModel{},
		SkillViewModel{},
		StoreItemViewModel{},
		ProjectViewModel{},
		AssistantTaskViewModel{},
		DiagnosticItemViewModel{},
	}
	for _, m := range models {
		if _, err := json.Marshal(m); err != nil {
			t.Errorf("view model %T is not json serializable: %v", m, err)
		}
	}
}

/** 验证 GetSnapshot 聚合所有子视图模型 */
func TestGetSnapshotAggregatesAllViewModels(t *testing.T) {
	app := &App{Name: defaultAppName, Version: defaultAppVersion}
	snapshot := app.GetSnapshot()

	if snapshot.Dashboard.Title == "" {
		t.Error("snapshot dashboard title must not be empty")
	}
	if snapshot.Agents == nil {
		t.Error("snapshot agents must not be nil")
	}
	if snapshot.Assistant.ID == "" {
		t.Error("snapshot assistant id must not be empty")
	}
}

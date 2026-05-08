package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/caojun/agent-skills-manager/internal/ai"
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

type fakeAssistantBridge struct {
	responses map[string]ai.WorkerResponse
	lastReq   ai.WorkerRequest
}

func (f *fakeAssistantBridge) Run(_ context.Context, req ai.WorkerRequest) (ai.WorkerResponse, error) {
	f.lastReq = req
	if resp, ok := f.responses[req.Action]; ok {
		return resp, nil
	}
	return ai.WorkerResponse{Status: "ok", Data: map[string]any{}}, nil
}

func TestSubmitGoalStoresStructuredPlan(t *testing.T) {
	bridge := &fakeAssistantBridge{
		responses: map[string]ai.WorkerResponse{
			"plan": {
				Status: "ok",
				Data: map[string]any{
					"goal": "install a skill",
					"steps": []any{
						map[string]any{"action": "recommend", "label": "推荐技能", "detail": "推荐适合的技能"},
						map[string]any{"action": "execute", "label": "执行安装", "detail": "执行技能安装"},
					},
					"revision": 1,
				},
			},
		},
	}

	app := &App{Name: defaultAppName, Version: defaultAppVersion, bridge: bridge}
	task := app.SubmitGoal("install a skill")

	if task.PlanJSON == "" {
		t.Fatal("plan json must not be empty")
	}
	if len(task.PlanSteps) != 2 {
		t.Fatalf("plan steps mismatch: got %d want 2", len(task.PlanSteps))
	}
	if len(task.ResolvedActions) != 0 {
		t.Fatalf("resolved actions mismatch: got %d want 0", len(task.ResolvedActions))
	}
}

func TestChatAssistantReturnsWorkerReply(t *testing.T) {
	bridge := &fakeAssistantBridge{
		responses: map[string]ai.WorkerResponse{
			"chat": {
				Status: "ok",
				Data: map[string]any{
					"reply": "你好，我是 AI 助手。",
				},
			},
		},
	}

	app := &App{Name: defaultAppName, Version: defaultAppVersion, bridge: bridge}
	response := app.ChatAssistant("你好", []AssistantChatMessageViewModel{
		{Role: "user", Content: "之前的问题"},
	})

	if response.Reply != "你好，我是 AI 助手。" {
		t.Fatalf("reply mismatch: got %q", response.Reply)
	}
	if bridge.lastReq.Action != "chat" {
		t.Fatalf("worker action mismatch: got %q want chat", bridge.lastReq.Action)
	}
}

func TestChatAssistantStripsReasoningFromWorkerReply(t *testing.T) {
	bridge := &fakeAssistantBridge{
		responses: map[string]ai.WorkerResponse{
			"chat": {
				Status: "ok",
				Data: map[string]any{
					"reply": "<think>internal reasoning</think>\n最终答案：你好，我是 AI 助手。",
				},
			},
		},
	}

	app := &App{Name: defaultAppName, Version: defaultAppVersion, bridge: bridge}
	response := app.ChatAssistant("你好", nil)

	if strings.Contains(response.Reply, "internal reasoning") || strings.Contains(response.Reply, "<think>") {
		t.Fatalf("reply leaked reasoning: %q", response.Reply)
	}
	if response.Reply != "你好，我是 AI 助手。" {
		t.Fatalf("reply mismatch: got %q", response.Reply)
	}
}

func TestSaveAISettingsUpdatesBridgeConfig(t *testing.T) {
	bridge := ai.NewLocalBridge("python3", "none", "")
	app := &App{Name: defaultAppName, Version: defaultAppVersion, bridge: bridge}

	result := app.SaveAISettings(AISettingsViewModel{
		Provider: "openai-compatible",
		Model:    "gpt-4.1-mini",
		APIKey:   "secret-key",
		BaseURL:  "https://example.invalid/v1",
	})

	if result != "ok" {
		t.Fatalf("save result mismatch: got %q want %q", result, "ok")
	}
	if bridge.Provider != "openai-compatible" {
		t.Fatalf("provider mismatch: got %q want %q", bridge.Provider, "openai-compatible")
	}
	if bridge.Model != "gpt-4.1-mini" {
		t.Fatalf("model mismatch: got %q want %q", bridge.Model, "gpt-4.1-mini")
	}
	if bridge.APIKey != "secret-key" {
		t.Fatalf("api key mismatch: got %q want %q", bridge.APIKey, "secret-key")
	}
	if bridge.BaseURL != "https://example.invalid/v1" {
		t.Fatalf("base url mismatch: got %q want %q", bridge.BaseURL, "https://example.invalid/v1")
	}
}

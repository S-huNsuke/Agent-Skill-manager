package ai

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

/** 验证 LocalBridge 在 Python 不可用时返回错误而非 panic */
func TestLocalBridgeReturnsErrorWhenPythonUnavailable(t *testing.T) {
	t.Parallel()

	bridge := &LocalBridge{
		PythonPath: "/nonexistent/python3",
		Provider:   "none",
		Timeout:    2 * time.Second,
	}

	_, err := bridge.Run(context.Background(), WorkerRequest{
		Action:  "plan",
		Payload: map[string]any{"goal": "test"},
	})
	if err == nil {
		t.Fatal("expected error when python is unavailable")
	}
}

/** 验证 LocalBridge 在超时时取消请求 */
func TestLocalBridgeRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	bridge := &LocalBridge{
		PythonPath: "python3",
		Provider:   "none",
		Timeout:    5 * time.Second,
	}

	_, err := bridge.Run(ctx, WorkerRequest{
		Action:  "plan",
		Payload: map[string]any{"goal": "test"},
	})
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

/** 验证 WorkerRequest 可正确序列化 */
func TestWorkerRequestSerialization(t *testing.T) {
	t.Parallel()

	req := WorkerRequest{
		Action: "plan",
		Payload: map[string]any{
			"goal": "install python skills",
			"options": map[string]any{
				"dry_run": true,
			},
		},
	}

	if req.Action != "plan" {
		t.Fatalf("action mismatch: got %q want %q", req.Action, "plan")
	}
	if goal, ok := req.Payload["goal"].(string); !ok || goal != "install python skills" {
		t.Fatalf("goal mismatch: got %v want %q", req.Payload["goal"], "install python skills")
	}
}

/** 验证 WorkerResponse 可正确反序列化 */
func TestWorkerResponseDeserialization(t *testing.T) {
	t.Parallel()

	resp := WorkerResponse{
		Status: "ok",
		Data: map[string]any{
			"goal":  "test",
			"steps": []any{"step1", "step2"},
		},
	}

	if resp.Status != "ok" {
		t.Fatalf("status mismatch: got %q want %q", resp.Status, "ok")
	}
	if steps, ok := resp.Data["steps"].([]any); !ok || len(steps) != 2 {
		t.Fatalf("steps mismatch: got %v want 2 items", resp.Data["steps"])
	}
}

/** 验证 NewLocalBridge 设置默认值 */
func TestNewLocalBridgeSetsDefaults(t *testing.T) {
	t.Parallel()

	bridge := NewLocalBridge("", "", "")
	if bridge.PythonPath != "python3" {
		t.Fatalf("python path mismatch: got %q want %q", bridge.PythonPath, "python3")
	}
	if bridge.Provider != "none" {
		t.Fatalf("provider mismatch: got %q want %q", bridge.Provider, "none")
	}
	if bridge.Timeout != 30*time.Second {
		t.Fatalf("timeout mismatch: got %v want %v", bridge.Timeout, 30*time.Second)
	}
}

/** 验证 UpdateConfig 会覆盖旧配置并允许清空敏感字段 */
func TestLocalBridgeUpdateConfigOverwritesPreviousValues(t *testing.T) {
	t.Parallel()

	bridge := &LocalBridge{
		Provider: "openai-compatible",
		Model:    "old-model",
		APIKey:   "old-key",
		BaseURL:  "https://old.invalid/v1",
	}

	bridge.UpdateConfig("", "", "", "")

	if bridge.Provider != "none" {
		t.Fatalf("provider mismatch: got %q want %q", bridge.Provider, "none")
	}
	if bridge.Model != "" {
		t.Fatalf("model mismatch: got %q want empty", bridge.Model)
	}
	if bridge.APIKey != "" {
		t.Fatalf("api key mismatch: got %q want empty", bridge.APIKey)
	}
	if bridge.BaseURL != "" {
		t.Fatalf("base url mismatch: got %q want empty", bridge.BaseURL)
	}
}

/** 验证 LocalBridge 会通过 stdin 传递敏感 worker 配置，并清理子进程环境中的密钥 */
func TestLocalBridgePassesWorkerConfigToChildProcess(t *testing.T) {
	dir := t.TempDir()
	workerDir := filepath.Join(dir, "python")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("create worker dir: %v", err)
	}
	scriptPath := filepath.Join(dir, "fake-python")
	script := `#!/bin/sh
payload="$(cat)"
printf '{"status":"ok","data":{"asm_provider":"%s","asm_model":"%s","asm_api_key":"%s","openai_api_key":"%s","anthropic_api_key":"%s","gemini_api_key":"%s","payload":%s,"arg1":"%s","arg2":"%s","arg3":"%s","arg4":"%s","cwd":"%s"}}\n' \
"$ASM_AI_PROVIDER" "$ASM_AI_MODEL" "$ASM_AI_API_KEY" "$OPENAI_API_KEY" "$ANTHROPIC_API_KEY" "$GEMINI_API_KEY" "$payload" "$1" "$2" "$3" "$4" "$PWD"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python: %v", err)
	}

	t.Setenv("ASM_AI_PROVIDER", "parent-provider")
	t.Setenv("ASM_AI_MODEL", "parent-model")
	t.Setenv("ASM_AI_API_KEY", "parent-secret")
	t.Setenv("OPENAI_API_KEY", "parent-openai-secret")
	t.Setenv("ANTHROPIC_API_KEY", "parent-anthropic-secret")
	t.Setenv("GEMINI_API_KEY", "parent-gemini-secret")

	bridge := &LocalBridge{
		PythonPath: scriptPath,
		Provider:   "openai-compatible",
		Model:      "test-model",
		WorkerDir:  workerDir,
		APIKey:     "secret-key",
		BaseURL:    "https://example.invalid/v1",
		Timeout:    5 * time.Second,
	}

	resp, err := bridge.Run(context.Background(), WorkerRequest{
		Action:  "plan",
		Payload: map[string]any{"goal": "test"},
	})
	if err != nil {
		t.Fatalf("bridge run failed: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status mismatch: got %q want %q", resp.Status, "ok")
	}

	data := resp.Data
	if got := data["asm_provider"]; got != "" {
		t.Fatalf("asm provider leaked into child env: got %v want empty", got)
	}
	if got := data["asm_model"]; got != "" {
		t.Fatalf("asm model leaked into child env: got %v want empty", got)
	}
	if got := data["asm_api_key"]; got != "" {
		t.Fatalf("asm api key leaked into child env: got %v want empty", got)
	}
	if got := data["openai_api_key"]; got != "" {
		t.Fatalf("openai api key leaked into child env: got %v want empty", got)
	}
	if got := data["anthropic_api_key"]; got != "" {
		t.Fatalf("anthropic api key leaked into child env: got %v want empty", got)
	}
	if got := data["gemini_api_key"]; got != "" {
		t.Fatalf("gemini api key leaked into child env: got %v want empty", got)
	}
	if got := data["arg1"]; got != "-m" {
		t.Fatalf("arg1 mismatch: got %v want %q", got, "-m")
	}
	if got := data["arg2"]; got != "worker.main" {
		t.Fatalf("arg2 mismatch: got %v want %q", got, "worker.main")
	}
	gotCWD, _ := data["cwd"].(string)
	realGotCWD, _ := filepath.EvalSymlinks(gotCWD)
	realWorkerDir, _ := filepath.EvalSymlinks(workerDir)
	if realGotCWD != realWorkerDir {
		t.Fatalf("cwd mismatch: got %v want %q", gotCWD, workerDir)
	}

	payload, ok := data["payload"].(map[string]any)
	if !ok {
		t.Fatalf("payload type mismatch: got %T", data["payload"])
	}
	body, ok := payload["payload"].(map[string]any)
	if !ok {
		t.Fatalf("worker body type mismatch: got %T", payload["payload"])
	}
	config, ok := body["config"].(map[string]any)
	if !ok {
		t.Fatalf("worker config type mismatch: got %T", body["config"])
	}
	if got := config["provider"]; got != "openai-compatible" {
		t.Fatalf("provider mismatch: got %v want %q", got, "openai-compatible")
	}
	if got := config["model"]; got != "test-model" {
		t.Fatalf("model mismatch: got %v want %q", got, "test-model")
	}
	if got := config["api_key"]; got != "secret-key" {
		t.Fatalf("api key mismatch: got %v want %q", got, "secret-key")
	}
	if got := config["base_url"]; got != "https://example.invalid/v1" {
		t.Fatalf("base url mismatch: got %v want %q", got, "https://example.invalid/v1")
	}
}

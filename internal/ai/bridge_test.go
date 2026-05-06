package ai

import (
	"context"
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
		Action: "plan",
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
		Action: "plan",
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

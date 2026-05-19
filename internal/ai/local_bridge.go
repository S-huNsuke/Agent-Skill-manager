package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

/** 通过本地 Python 进程调用 AI Worker 的 Bridge 实现 */
type LocalBridge struct {
	PythonPath string
	Provider   string
	Model      string
	APIKey     string
	BaseURL    string
	WorkerDir  string
	Timeout    time.Duration
}

/** 创建默认的 LocalBridge 实例 */
func NewLocalBridge(pythonPath string, provider string, model string) *LocalBridge {
	if pythonPath == "" {
		pythonPath = "python3"
	}
	if provider == "" {
		provider = "none"
	}
	return &LocalBridge{
		PythonPath: pythonPath,
		Provider:   provider,
		Model:      model,
		APIKey:     "",
		BaseURL:    "",
		WorkerDir:  "", // 将在 Run 时设置
		Timeout:    30 * time.Second,
	}
}

/** 执行 Worker 请求，通过 stdin/stdout 与 Python 进程通信 */
func (b *LocalBridge) Run(ctx context.Context, req WorkerRequest) (WorkerResponse, error) {
	body := make(map[string]any, len(req.Payload)+1)
	for key, value := range req.Payload {
		body[key] = value
	}
	body["config"] = map[string]any{
		"provider": b.Provider,
		"model":    b.Model,
		"api_key":  b.APIKey,
		"base_url": b.BaseURL,
	}

	payload := map[string]any{
		"action":  req.Action,
		"payload": body,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return WorkerResponse{Status: "error", Data: map[string]any{"error": err.Error()}}, fmt.Errorf("marshal payload: %w", err)
	}

	args := []string{"-m", "worker.main", "--provider", b.Provider}
	if b.Model != "" {
		args = append(args, "--model", b.Model)
	}

	if b.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, b.PythonPath, args...)
	cmd.Stdin = bytes.NewReader(payloadBytes)
	cmd.Env = filterSensitiveEnv(os.Environ())

	// 设置工作目录为 python，这样模块导入 worker.main 才能正常工作
	if b.WorkerDir != "" {
		cmd.Dir = b.WorkerDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Cancel = func() error {
		return cmd.Process.Kill()
	}

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return WorkerResponse{
			Status: "error",
			Data:   map[string]any{"error": fmt.Sprintf("worker execution failed: %s", errMsg)},
		}, fmt.Errorf("run python worker: %w", err)
	}

	var response WorkerResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return WorkerResponse{
			Status: "error",
			Data:   map[string]any{"error": fmt.Sprintf("invalid worker response: %s", stdout.String())},
		}, fmt.Errorf("unmarshal worker response: %w", err)
	}

	if response.Status != "ok" {
		errMsg := ""
		if response.Data != nil {
			errMsg, _ = response.Data["error"].(string)
		}
		if errMsg == "" {
			errMsg = "worker returned non-ok status"
		}
		return response, fmt.Errorf("%s", errMsg)
	}

	return response, nil
}

/** 动态更新 Bridge 的 Provider、Model 和 API 配置 */
func (b *LocalBridge) UpdateConfig(provider string, model string, apiKey string, baseURL string) {
	if provider == "" {
		provider = "none"
	}
	b.Provider = provider
	b.Model = model
	b.APIKey = apiKey
	b.BaseURL = baseURL
}

func filterSensitiveEnv(env []string) []string {
	result := make([]string, 0, len(env))
	for _, entry := range env {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(name)
		switch {
		case strings.HasPrefix(upper, "ASM_AI_"):
			continue
		case strings.HasSuffix(upper, "_API_KEY"):
			continue
		case strings.HasSuffix(upper, "_TOKEN"):
			continue
		case strings.HasSuffix(upper, "_SECRET"):
			continue
		}
		result = append(result, entry)
	}
	return result
}

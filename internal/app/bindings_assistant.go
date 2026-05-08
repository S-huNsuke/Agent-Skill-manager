package app

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
	"github.com/caojun/agent-skills-manager/internal/ai"
	"github.com/caojun/agent-skills-manager/internal/domain"
)

func (a *App) GetAssistantTask() AssistantTaskViewModel {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()

	if a.activeTask != nil {
		return *a.activeTask
	}

	return AssistantTaskViewModel{
		ID:              "assistant-idle",
		Request:         "",
		Status:          "queued",
		NextStep:        "等待用户输入目标",
		Summary:         "AI 助手待命中，输入目标即可开始规划。",
		Recommendation:  "输入一个目标，让 AI 帮你规划技能安装与修复。",
		Reason:          "",
		Records:         []string{},
		PlanJSON:        "",
		PlanSteps:       []AssistantPlanStep{},
		ResolvedActions: []AssistantResolvedAction{},
	}
}

/** 返回诊断信息列表 */
func assistantPlanStepsFromData(raw any) []AssistantPlanStep {
	stepsData, ok := raw.([]any)
	if !ok {
		return []AssistantPlanStep{}
	}

	steps := make([]AssistantPlanStep, 0, len(stepsData))
	for _, item := range stepsData {
		stepMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		steps = append(steps, AssistantPlanStep{
			Action: assistantStringValue(stepMap["action"]),
			Label:  assistantStringValue(stepMap["label"]),
			Detail: assistantStringValue(stepMap["detail"]),
		})
	}
	return steps
}

func assistantResolvedActionsFromData(raw any) []AssistantResolvedAction {
	actionsData, ok := raw.([]any)
	if !ok {
		return []AssistantResolvedAction{}
	}

	actions := make([]AssistantResolvedAction, 0, len(actionsData))
	for _, item := range actionsData {
		actionMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		skillID := assistantStringValue(actionMap["skill_id"])
		if skillID == "" {
			skillID = assistantStringValue(actionMap["skillId"])
		}
		if skillID == "" {
			skillID = assistantStringValue(actionMap["name"])
		}

		targetAgent := assistantStringValue(actionMap["target_agent"])
		if targetAgent == "" {
			targetAgent = assistantStringValue(actionMap["targetAgent"])
		}

		action := assistantStringValue(actionMap["action"])
		if action == "" {
			continue
		}

		actions = append(actions, AssistantResolvedAction{
			SkillID:     skillID,
			Version:     assistantStringValue(actionMap["version"]),
			TargetAgent: targetAgent,
			Action:      action,
		})
	}

	return actions
}

func assistantPlanDataFromTask(task *AssistantTaskViewModel) map[string]any {
	if task == nil {
		return map[string]any{}
	}

	if task.PlanJSON != "" {
		var data map[string]any
		if err := json.Unmarshal([]byte(task.PlanJSON), &data); err == nil {
			return data
		}
	}

	steps := make([]map[string]any, 0, len(task.PlanSteps))
	for _, step := range task.PlanSteps {
		steps = append(steps, map[string]any{
			"action": step.Action,
			"label":  step.Label,
			"detail": step.Detail,
		})
	}

	return map[string]any{
		"goal":     task.Request,
		"steps":    steps,
		"revision": 1,
	}
}

func assistantStringValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func assistantNewTaskRecords(goal string, planSteps []AssistantPlanStep) []string {
	records := []string{fmt.Sprintf("用户提交目标: %s", goal)}
	for _, step := range planSteps {
		if step.Label == "" {
			continue
		}
		detail := step.Detail
		if detail == "" {
			detail = "无详情"
		}
		records = append(records, fmt.Sprintf("步骤: %s — %s", step.Label, detail))
	}
	return records
}

func assistantDefaultPlanSteps() []AssistantPlanStep {
	return []AssistantPlanStep{
		{Action: "recommend", Label: "推荐技能", Detail: "根据目标推荐合适的技能"},
		{Action: "resolve", Label: "解析依赖", Detail: "解析技能依赖和兼容性"},
		{Action: "execute", Label: "执行安装", Detail: "安装或修复技能"},
		{Action: "verify", Label: "验证结果", Detail: "验证安装结果和完整性"},
	}
}

func firstAssistantPlanLabel(steps []AssistantPlanStep) string {
	for _, step := range steps {
		if step.Label != "" {
			return step.Label
		}
	}
	return "规划完成，等待用户确认"
}

func (a *App) persistAssistantTask(task AssistantTaskViewModel, startedAt time.Time, finishedAt *time.Time) {
	if a.taskRepo == nil {
		return
	}

	data := domain.Task{
		ID:            task.ID,
		TaskType:      "ai_assistant",
		TriggerSource: task.Request,
		Status:        task.Status,
		StatusReason:  task.Blocker,
		PlanJSON:      task.PlanJSON,
		ActionLog:     strings.Join(task.Records, "\n"),
		ResultSummary: task.Summary,
		StartedAt:     &startedAt,
		FinishedAt:    finishedAt,
	}

	if existing, err := a.taskRepo.GetByID(context.Background(), task.ID); err == nil && existing.StartedAt != nil {
		data.StartedAt = existing.StartedAt
	}

	if err := a.taskRepo.Put(context.Background(), data); err != nil && a.logger != nil {
		a.logger.Error("persist assistant task failed", "error", err, "taskID", task.ID)
	}
}

/** 提交 AI 助手目标 */
func (a *App) SubmitGoal(goal string) AssistantTaskViewModel {
	ctx := context.Background()
	now := time.Now()

	if a.logger != nil {
		a.logger.Info("SubmitGoal called", "goal", goal)
	}

	planSteps := assistantDefaultPlanSteps()
	planData := map[string]any{
		"goal":     goal,
		"steps":    planSteps,
		"revision": 1,
	}
	records := assistantNewTaskRecords(goal, planSteps)
	summary := fmt.Sprintf("正在为「%s」规划技能安装方案。", goal)
	recommendation := "AI 助手正在分析您的目标，稍后将给出技能推荐。"
	nextStep := "正在分析目标并规划技能安装步骤"

	if a.bridge != nil {
		agents := a.GetAgents()
		skills := a.GetSkills()
		projects := a.GetProjects()
		storeItems := a.GetStoreItems()

		contextData := map[string]any{
			"agents_count":      len(agents),
			"skills_count":      len(skills),
			"projects_count":    len(projects),
			"store_items_count": len(storeItems),
		}

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

		if a.logger != nil {
			if err != nil {
				a.logger.Error("Python worker failed", "error", err)
			} else {
				a.logger.Info("Python worker success", "status", resp.Status)
			}
		}

		if err == nil && resp.Status == "ok" {
			if dataGoal, ok := resp.Data["goal"].(string); ok && dataGoal != "" {
				goal = dataGoal
			}
			if steps := assistantPlanStepsFromData(resp.Data["steps"]); len(steps) > 0 {
				planSteps = steps
				records = assistantNewTaskRecords(goal, planSteps)
			}
			planData = resp.Data
			if raw, err := json.Marshal(resp.Data); err == nil {
				planJSON := string(raw)
				result := AssistantTaskViewModel{
					ID:              fmt.Sprintf("task-%d", now.UnixMilli()),
					Request:         goal,
					Status:          "planning",
					NextStep:        fmt.Sprintf("下一步: %s", firstAssistantPlanLabel(planSteps)),
					Summary:         fmt.Sprintf("已为「%s」生成执行计划，共 %d 个步骤。", goal, len(planSteps)),
					Recommendation:  "AI 助手已生成执行计划，请查看步骤并确认执行。",
					Reason:          "",
					Records:         records,
					PlanJSON:        planJSON,
					PlanSteps:       planSteps,
					ResolvedActions: []AssistantResolvedAction{},
				}
				a.assistantMu.Lock()
				a.activeTask = &result
				a.assistantMu.Unlock()
				a.persistAssistantTask(result, now, nil)
				if a.logger != nil {
					a.logger.Info("SubmitGoal completed", "taskID", result.ID, "status", result.Status)
				}
				return result
			}
		}
	}

	planJSON, _ := json.Marshal(planData)
	result := AssistantTaskViewModel{
		ID:              fmt.Sprintf("task-%d", now.UnixMilli()),
		Request:         goal,
		Status:          "planning",
		NextStep:        nextStep,
		Summary:         summary,
		Recommendation:  recommendation,
		Reason:          "",
		Records:         records,
		PlanJSON:        string(planJSON),
		PlanSteps:       planSteps,
		ResolvedActions: []AssistantResolvedAction{},
	}

	a.assistantMu.Lock()
	a.activeTask = &result
	a.assistantMu.Unlock()

	if a.logger != nil {
		a.logger.Info("SubmitGoal completed", "taskID", result.ID, "status", result.Status)
	}

	a.persistAssistantTask(result, now, nil)

	return result
}

/** 发送普通 AI 聊天消息 */
func (a *App) ChatAssistant(message string, history []AssistantChatMessageViewModel) AssistantChatResponseViewModel {
	settings := a.GetAISettings()
	if strings.TrimSpace(message) == "" {
		return AssistantChatResponseViewModel{Reply: "请输入要发送的消息。", Provider: settings.Provider, Model: settings.Model}
	}

	historyPayload := make([]map[string]any, 0, len(history))
	for _, item := range history {
		if item.Content == "" {
			continue
		}
		historyPayload = append(historyPayload, map[string]any{
			"role":    item.Role,
			"content": item.Content,
		})
	}

	if a.bridge != nil {
		resp, err := a.bridge.Run(context.Background(), ai.WorkerRequest{
			Action: "chat",
			Payload: map[string]any{
				"message": message,
				"history": historyPayload,
			},
		})
		if err == nil && resp.Status == "ok" {
			reply := sanitizeAssistantReply(assistantStringValue(resp.Data["reply"]))
			if reply == "" {
				reply = "模型没有返回内容。"
			}
			return AssistantChatResponseViewModel{
				Reply:    reply,
				Provider: settings.Provider,
				Model:    settings.Model,
				Error:    assistantStringValue(resp.Data["error"]),
			}
		}
		if err != nil {
			return AssistantChatResponseViewModel{
				Reply:    fmt.Sprintf("AI 请求失败：%s", err.Error()),
				Provider: settings.Provider,
				Model:    settings.Model,
				Error:    err.Error(),
			}
		}
	}

	return AssistantChatResponseViewModel{
		Reply:    fmt.Sprintf("我收到你的消息了：%s\n\n当前没有可用的 AI worker，请检查 AI 配置和本地 Python 环境。", message),
		Provider: settings.Provider,
		Model:    settings.Model,
	}
}

func sanitizeAssistantReply(text string) string {
	reply := strings.TrimSpace(text)
	patterns := []string{
		`(?is)<\s*(?:think|thinking|reasoning|analysis)\s*>.*?<\s*/\s*(?:think|thinking|reasoning|analysis)\s*>`,
		"(?is)^```(?:think|thinking|reasoning|analysis)\\s+.*?```\\s*",
		"(?is)^\\s*(?:思考过程|思考|推理过程|推理|Thinking|Reasoning|Analysis)\\s*[:：].*?\\n\\s*\\n",
		"(?is)^\\s*(?:最终答案|回答)\\s*[:：]\\s*",
	}
	for _, pattern := range patterns {
		reply = regexp.MustCompile(pattern).ReplaceAllString(reply, "")
	}
	for _, marker := range []string{"最终答案：", "最终答案:", "回答：", "回答:"} {
		if idx := strings.LastIndex(reply, marker); idx >= 0 {
			reply = reply[idx+len(marker):]
			break
		}
	}
	return strings.TrimSpace(reply)
}

/** 推进 AI 助手任务到下一阶段 */
func (a *App) AdvanceAssistantTask(taskID string, action string) AssistantTaskViewModel {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()

	if a.activeTask == nil || a.activeTask.ID != taskID {
		return AssistantTaskViewModel{
			ID:              "assistant-idle",
			Status:          "queued",
			NextStep:        "等待用户输入目标",
			Summary:         "AI 助手待命中，输入目标即可开始规划。",
			Recommendation:  "输入一个目标，让 AI 帮你规划技能安装与修复。",
			Records:         []string{},
			PlanJSON:        "",
			PlanSteps:       []AssistantPlanStep{},
			ResolvedActions: []AssistantResolvedAction{},
		}
	}

	task := a.activeTask
	records := make([]string, len(task.Records))
	copy(records, task.Records)
	planData := assistantPlanDataFromTask(task)
	now := time.Now()

	switch action {
	case "resolve":
		task.Status = "resolving"
		task.NextStep = "正在解析依赖关系"
		task.Blocker = ""
		records = append(records, "开始解析依赖关系...")

		if a.bridge != nil {
			resp, err := a.bridge.Run(context.Background(), ai.WorkerRequest{
				Action: "resolve",
				Payload: map[string]any{
					"plan":                planData,
					"has_artifact":        task.PlanJSON != "",
					"adapter_owns_target": a.registry != nil,
				},
			})
			if err == nil && resp.Status == "ok" {
				if status := assistantStringValue(resp.Data["status"]); status == "blocked" {
					task.Status = "blocked"
					task.Blocker = assistantStringValue(resp.Data["summary"])
					if task.Blocker == "" {
						task.Blocker = "任务被前置条件阻塞"
					}
					task.Summary = task.Blocker
					task.NextStep = "任务被阻塞，等待修复前置条件"
				} else {
					records = append(records, "依赖解析完成")
					if detail := assistantStringValue(resp.Data["summary"]); detail != "" {
						records = append(records, fmt.Sprintf("解析结果: %s", detail))
					}
					if actions := assistantResolvedActionsFromData(resp.Data["actions"]); len(actions) > 0 {
						task.ResolvedActions = actions
						records = append(records, fmt.Sprintf("解析出 %d 个可执行动作", len(actions)))
						task.NextStep = fmt.Sprintf("依赖解析完成，已生成 %d 个动作，等待执行确认", len(actions))
					} else {
						task.NextStep = "依赖解析完成，等待执行确认"
					}
					if summary := assistantStringValue(resp.Data["summary"]); summary != "" {
						task.Summary = summary
					} else {
						task.Summary = fmt.Sprintf("已为「%s」完成依赖解析，可以开始执行。", task.Request)
					}
				}
			} else {
				records = append(records, "依赖解析使用本地回退方案")
				task.NextStep = "依赖解析完成（本地模式），等待执行确认"
				task.Summary = fmt.Sprintf("已为「%s」完成依赖解析（本地模式）。", task.Request)
			}
		} else {
			records = append(records, "依赖解析完成（本地模式）")
			task.NextStep = "依赖解析完成，等待执行确认"
			task.Summary = fmt.Sprintf("已为「%s」完成依赖解析（本地模式）。", task.Request)
		}

	case "execute":
		task.Status = "executing"
		task.NextStep = "正在执行安装操作"
		records = append(records, "开始执行安装操作...")

		actions := task.ResolvedActions
		agentID := a.findAgentIDForTask(task)
		executedCount := 0
		if len(actions) == 0 {
			records = append(records, "未解析出可执行动作")
		}
		for _, item := range actions {
			targetAgent := item.TargetAgent
			if targetAgent == "" {
				targetAgent = agentID
			}
			switch item.Action {
			case "install":
				installResult := a.InstallSkill(targetAgent, item.SkillID, "")
				if installResult == "ok" {
					executedCount++
					records = append(records, fmt.Sprintf("✓ 技能 %s 安装成功", item.SkillID))
				} else {
					records = append(records, fmt.Sprintf("✗ 技能 %s 安装失败: %s", item.SkillID, installResult))
				}
			case "update":
				updateResult := a.UpdateSkill(targetAgent, item.SkillID, "")
				if updateResult == "ok" {
					executedCount++
					records = append(records, fmt.Sprintf("✓ 技能 %s 更新成功", item.SkillID))
				} else {
					records = append(records, fmt.Sprintf("✗ 技能 %s 更新失败: %s", item.SkillID, updateResult))
				}
			case "repair":
				if item.SkillID != "" {
					records = append(records, fmt.Sprintf("⚙ 修复代理并重新应用技能 %s", item.SkillID))
				} else {
					records = append(records, "⚙ 修复代理")
				}
				repairResult := a.RepairAgent(targetAgent)
				if repairResult == "ok" && item.SkillID != "" {
					updateResult := a.UpdateSkill(targetAgent, item.SkillID, "")
					if updateResult == "ok" {
						executedCount++
						records = append(records, fmt.Sprintf("✓ 技能 %s 修复并更新成功", item.SkillID))
					} else {
						records = append(records, fmt.Sprintf("✗ 技能 %s 修复后更新失败: %s", item.SkillID, updateResult))
					}
				} else if repairResult != "ok" {
					records = append(records, fmt.Sprintf("✗ 代理修复失败: %s", repairResult))
				}
			default:
				records = append(records, fmt.Sprintf("✗ 未支持的动作: %s", item.Action))
			}
		}

		if executedCount > 0 {
			task.NextStep = fmt.Sprintf("已执行 %d 个动作，等待验证", executedCount)
			task.Summary = fmt.Sprintf("已为「%s」执行 %d 个动作。", task.Request, executedCount)
		} else {
			task.NextStep = "执行完成（无动作），等待验证"
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
			resultPayload := map[string]any{
				"status":         task.Status,
				"goal":           task.Request,
				"summary":        task.Summary,
				"records":        records,
				"plan":           planData,
				"actions":        task.ResolvedActions,
				"recommendation": task.Recommendation,
			}
			resp, err := a.bridge.Run(context.Background(), ai.WorkerRequest{
				Action: "report",
				Payload: map[string]any{
					"result": resultPayload,
				},
			})
			if err == nil && resp.Status == "ok" {
				if summary, ok := resp.Data["summary"].(string); ok && summary != "" {
					task.Summary = summary
					records = append(records, fmt.Sprintf("报告: %s", summary))
				}
				if recommendation, ok := resp.Data["recommendation"].(string); ok && recommendation != "" {
					task.Recommendation = recommendation
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

	if task.Status == "completed" || task.Status == "cancelled" || task.Status == "failed" || task.Status == "blocked" {
		finished := now
		a.persistAssistantTask(*task, now, &finished)
	} else {
		a.persistAssistantTask(*task, now, nil)
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
		ID:              "assistant-idle",
		Request:         "",
		Status:          "queued",
		NextStep:        "等待用户输入目标",
		Summary:         "AI 助手待命中，输入目标即可开始规划。",
		Recommendation:  "输入一个目标，让 AI 帮你规划技能安装与修复。",
		Records:         []string{},
		PlanJSON:        "",
		PlanSteps:       []AssistantPlanStep{},
		ResolvedActions: []AssistantResolvedAction{},
	}
}

func (a *App) GetSuggestionTemplates() []SuggestionTemplate {
	return []SuggestionTemplate{
		{ID: "tpl-1", Category: "Web 开发", Title: "搭建 Web 开发环境", Description: "安装前端开发相关的技能", PromptTemplate: "帮我搭建一个 Web 开发项目，安装前端开发相关的技能", IsBuiltin: true},
		{ID: "tpl-2", Category: "测试", Title: "添加测试技能", Description: "安装单元测试和集成测试技能", PromptTemplate: "为我的项目添加测试技能，包括单元测试和集成测试", IsBuiltin: true},
		{ID: "tpl-3", Category: "数据分析", Title: "数据分析环境", Description: "安装数据处理和分析技能", PromptTemplate: "帮我配置数据分析环境，安装数据处理和可视化相关技能", IsBuiltin: true},
		{ID: "tpl-4", Category: "优化", Title: "优化当前配置", Description: "检查并优化现有技能配置", PromptTemplate: "检查我当前的技能配置，推荐优化方案", IsBuiltin: true},
		{ID: "tpl-5", Category: "DevOps", Title: "DevOps 工具链", Description: "安装 CI/CD 和部署相关技能", PromptTemplate: "帮我配置 DevOps 工具链，安装 CI/CD 和部署相关技能", IsBuiltin: true},
	}
}

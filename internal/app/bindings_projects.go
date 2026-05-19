package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/caojun/agent-skills-manager/internal/agents"
	"github.com/caojun/agent-skills-manager/internal/domain"
	"github.com/caojun/agent-skills-manager/internal/reconcile"
	"github.com/caojun/agent-skills-manager/internal/skillgroups"
)

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
	return "error: 项目不存在"
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
	return "error: 项目不存在"
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
	return "error: 项目不存在"
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
	return "error: 技能组不存在"
}

/** 为技能组添加技能 */
func (a *App) AddSkillToGroup(groupID string, skillName string) string {
	a.projectsMu.Lock()
	defer a.projectsMu.Unlock()

	for i := range a.skillGroups {
		if a.skillGroups[i].ID == groupID {
			for _, existing := range a.skillGroups[i].SkillNames {
				if existing == skillName {
					return "error: 技能已在该组中"
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
	return "error: 技能组不存在"
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
			return "error: 技能不在该组中"
		}
	}
	return "error: 技能组不存在"
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

/** 删除单条任务历史 */
func (a *App) DeleteTaskHistoryItem(taskID string) string {
	if strings.TrimSpace(taskID) == "" {
		return "error: 任务 ID 不能为空"
	}
	if a.taskRepo == nil {
		return "error: 任务存储不可用"
	}
	if err := a.taskRepo.Delete(context.Background(), taskID); err != nil {
		return fmt.Sprintf("error: %s", err.Error())
	}
	return "ok"
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
								ID:           fmt.Sprintf("%s-%s", install.AgentID, name),
								SkillID:      name,
								AgentID:      install.AgentID,
								Version:      "installed",
								InstallState: "installed",
								InstallPath:  filepath.Join(install.SkillsPath, name),
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
		return "error: 协调计划 JSON 格式无效"
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
		return "error: 项目不存在"
	}

	agentID := project.BoundAgentID
	if agentID == "" {
		return "error: 项目未绑定代理"
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

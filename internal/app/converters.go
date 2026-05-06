package app

import (
	"time"

	"github.com/caojun/agent-skills-manager/internal/domain"
)

/** 将 ProjectViewModel 转换为 domain.Project */
func projectVMToDomain(vm ProjectViewModel) domain.Project {
	now := time.Now().UTC()
	return domain.Project{
		ID:               vm.ID,
		Name:             vm.Name,
		Type:             "local",
		Path:             vm.Path,
		Description:      vm.Summary,
		SkillGroupID:     "",
		AutoApplyEnabled: false,
		BoundAgentID:     vm.BoundAgentID,
		BoundAgentName:   vm.BoundAgentName,
		LastActiveAt:     nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

/** 将 domain.Project 转换为 ProjectViewModel */
func projectDomainToVM(p domain.Project) ProjectViewModel {
	stage := "已识别"
	if p.BoundAgentID != "" {
		stage = "已配置"
	}
	return ProjectViewModel{
		ID:              p.ID,
		Name:            p.Name,
		Path:            p.Path,
		Stage:           stage,
		BoundSkillGroup: p.SkillGroupID,
		BoundAgentID:    p.BoundAgentID,
		BoundAgentName:  p.BoundAgentName,
		SkillNames:      make([]string, 0),
		Summary:         p.Description,
		Needs:           make([]string, 0),
		LocalAgents:     make([]string, 0),
		Recent:          make([]string, 0),
		CreatedAt:       p.CreatedAt.Format("2006-01-02"),
	}
}

/** 将 SkillGroupViewModel 转换为 domain.SkillGroup */
func skillGroupVMToDomain(vm SkillGroupViewModel) domain.SkillGroup {
	now := time.Now().UTC()
	return domain.SkillGroup{
		ID:              vm.ID,
		Name:            vm.Name,
		Description:     vm.Description,
		SourceType:      vm.SourceType,
		Version:         "",
		GoalPrompt:      "",
		PreferredAgents: make([]string, 0),
		BoundAgentID:    vm.BoundAgentID,
		BoundAgentName:  vm.BoundAgentName,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

/** 将 domain.SkillGroup 转换为 SkillGroupViewModel */
func skillGroupDomainToVM(g domain.SkillGroup) SkillGroupViewModel {
	return SkillGroupViewModel{
		ID:             g.ID,
		Name:           g.Name,
		Description:    g.Description,
		SourceType:     g.SourceType,
		SkillCount:     0,
		ProjectCount:   0,
		CreatedAt:      g.CreatedAt.Format("2006-01-02 15:04"),
		SkillNames:     make([]string, 0),
		BoundAgentID:   g.BoundAgentID,
		BoundAgentName: g.BoundAgentName,
	}
}

/** 将 CatalogSourceViewModel 转换为 domain.CatalogSource */
func catalogSourceVMToDomain(vm CatalogSourceViewModel) domain.CatalogSource {
	return domain.CatalogSource{
		ID:                        vm.ID,
		Name:                      vm.Name,
		URL:                       vm.URL,
		IsBuiltin:                 vm.IsBuiltin,
		Enabled:                   vm.Enabled,
		LastSyncedAt:              nil,
		LastSyncStatus:            vm.LastSyncStatus,
		LastSyncError:             "",
		CacheExpiresAt:            nil,
		MinSupportedClientVersion: "",
	}
}

/** 将 domain.CatalogSource 转换为 CatalogSourceViewModel */
func catalogSourceDomainToVM(s domain.CatalogSource) CatalogSourceViewModel {
	lastSynced := ""
	if s.LastSyncedAt != nil {
		lastSynced = s.LastSyncedAt.Format("2006-01-02 15:04")
	}
	return CatalogSourceViewModel{
		ID:             s.ID,
		Name:           s.Name,
		URL:            s.URL,
		IsBuiltin:      s.IsBuiltin,
		Enabled:        s.Enabled,
		LastSyncedAt:   lastSynced,
		LastSyncStatus: s.LastSyncStatus,
		SkillCount:     0,
	}
}

/** 将 domain.CatalogSkill 转换为 StoreItemViewModel */
func catalogSkillDomainToVM(s domain.CatalogSkill) StoreItemViewModel {
	status := "available"
	return StoreItemViewModel{
		ID:            s.ID,
		Name:          s.Name,
		Author:        s.Author,
		Source:        s.SourceID,
		Status:        status,
		Summary:       s.Description,
		Installs:      "",
		Impact:        "",
		Compatibility: s.SupportedAgents,
		Homepage:      s.Homepage,
	}
}

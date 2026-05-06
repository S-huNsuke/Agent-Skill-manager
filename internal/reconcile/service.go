package reconcile

import (
	"fmt"

	"github.com/caojun/agent-skills-manager/internal/domain"
	"github.com/caojun/agent-skills-manager/internal/skillgroups"
)

const installStateInstalled = "installed"

type Action struct {
	SkillID string
	Version string
}

type Plan struct {
	Install     []Action
	Update      []Action
	Repair      []Action
	BlockReason string
}

type Service struct{}

func (Service) Plan(desired []skillgroups.DesiredSkill, catalog []domain.CatalogSkill, installed []domain.InstalledSkill) (Plan, error) {
	plan := Plan{
		Install: make([]Action, 0),
		Update:  make([]Action, 0),
		Repair:  make([]Action, 0),
	}

	catalogBySkillID := make(map[string]domain.CatalogSkill, len(catalog))
	for _, item := range catalog {
		catalogBySkillID[item.ID] = item
	}

	for _, wanted := range desired {
		if _, ok := catalogBySkillID[wanted.SkillID]; !ok {
			plan.BlockReason = fmt.Sprintf("missing catalog entry for skill %q", wanted.SkillID)
			return plan, nil
		}
	}

	installedBySkillID := make(map[string]domain.InstalledSkill, len(installed))
	for _, item := range installed {
		if _, exists := installedBySkillID[item.SkillID]; !exists {
			installedBySkillID[item.SkillID] = item
		}
	}

	for _, wanted := range desired {
		catalogEntry := catalogBySkillID[wanted.SkillID]

		current, ok := installedBySkillID[wanted.SkillID]
		if !ok {
			plan.Install = append(plan.Install, Action{
				SkillID: catalogEntry.ID,
				Version: catalogEntry.Version,
			})
			continue
		}

		if current.Version != catalogEntry.Version {
			plan.Update = append(plan.Update, Action{
				SkillID: catalogEntry.ID,
				Version: catalogEntry.Version,
			})
			continue
		}

		if current.InstallState != installStateInstalled || current.ErrorMessage != "" {
			plan.Repair = append(plan.Repair, Action{
				SkillID: catalogEntry.ID,
				Version: catalogEntry.Version,
			})
		}
	}

	return plan, nil
}

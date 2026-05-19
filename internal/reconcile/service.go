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

	resolvedCatalogBySkillID := make(map[string]domain.CatalogSkill, len(desired))
	resolvedActionSkillIDs := make(map[string]string, len(desired))
	for _, wanted := range desired {
		catalogEntry, resolvedSkillID, err := resolveCatalogSkill(catalog, wanted.SkillID)
		if err != nil {
			plan.BlockReason = err.Error()
			return plan, nil
		}
		resolvedCatalogBySkillID[wanted.SkillID] = catalogEntry
		resolvedActionSkillIDs[wanted.SkillID] = resolvedSkillID
	}

	for _, wanted := range desired {
		catalogEntry := resolvedCatalogBySkillID[wanted.SkillID]
		resolvedSkillID := resolvedActionSkillIDs[wanted.SkillID]

		current, ok := lookupInstalledSkill(installed, wanted.SkillID, resolvedSkillID)
		if !ok {
			plan.Install = append(plan.Install, Action{
				SkillID: resolvedSkillID,
				Version: catalogEntry.Version,
			})
			continue
		}

		if current.Version != catalogEntry.Version {
			plan.Update = append(plan.Update, Action{
				SkillID: resolvedSkillID,
				Version: catalogEntry.Version,
			})
			continue
		}

		if current.InstallState != installStateInstalled || current.ErrorMessage != "" {
			plan.Repair = append(plan.Repair, Action{
				SkillID: resolvedSkillID,
				Version: catalogEntry.Version,
			})
		}
	}

	return plan, nil
}

func resolveCatalogSkill(catalog []domain.CatalogSkill, ref string) (domain.CatalogSkill, string, error) {
	for _, item := range catalog {
		if item.ID == ref {
			return item, resolvedCatalogSkillID(item), nil
		}
	}

	for _, item := range catalog {
		if item.Name == ref {
			return item, resolvedCatalogSkillID(item), nil
		}
	}

	return domain.CatalogSkill{}, "", fmt.Errorf("missing catalog entry for skill %q", ref)
}

func resolvedCatalogSkillID(item domain.CatalogSkill) string {
	if item.Name != "" {
		return item.Name
	}
	return item.ID
}

func lookupInstalledSkill(installed []domain.InstalledSkill, refs ...string) (domain.InstalledSkill, bool) {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		for _, item := range installed {
			if item.SkillID == ref || item.ID == ref {
				return item, true
			}
		}
	}
	return domain.InstalledSkill{}, false
}

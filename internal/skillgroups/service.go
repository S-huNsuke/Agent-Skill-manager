package skillgroups

import (
	"fmt"
	"slices"

	"github.com/caojun/agent-skills-manager/internal/domain"
)

type DesiredSkill struct {
	SkillID   string
	Required  bool
	Priority  int
	Reason    string
	GroupID   string
	GroupName string
}

type Service struct{}

func (Service) Validate(group domain.SkillGroup, bindings []domain.SkillGroupSkill) error {
	if group.ID == "" {
		return fmt.Errorf("skill group id is required")
	}

	for _, binding := range bindings {
		if binding.SkillGroupID != group.ID {
			return fmt.Errorf("skill %q is bound to skill group %q, want %q", binding.SkillID, binding.SkillGroupID, group.ID)
		}
		if binding.SkillID == "" {
			return fmt.Errorf("skill group %q contains an empty skill id", group.ID)
		}
	}

	return nil
}

func (svc Service) DesiredSkills(group domain.SkillGroup, bindings []domain.SkillGroupSkill) ([]DesiredSkill, error) {
	if err := svc.Validate(group, bindings); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(bindings))
	desired := make([]DesiredSkill, 0, len(bindings))
	for _, binding := range bindings {
		if _, ok := seen[binding.SkillID]; ok {
			return nil, fmt.Errorf("duplicate desired skill %q in skill group %q", binding.SkillID, group.ID)
		}
		seen[binding.SkillID] = struct{}{}

		desired = append(desired, DesiredSkill{
			SkillID:   binding.SkillID,
			Required:  binding.Required,
			Priority:  binding.Priority,
			Reason:    binding.Reason,
			GroupID:   group.ID,
			GroupName: group.Name,
		})
	}

	slices.SortFunc(desired, func(a, b DesiredSkill) int {
		switch {
		case a.Priority != b.Priority:
			if a.Priority > b.Priority {
				return -1
			}
			return 1
		case a.Required != b.Required:
			if a.Required {
				return -1
			}
			return 1
		case a.SkillID < b.SkillID:
			return -1
		case a.SkillID > b.SkillID:
			return 1
		default:
			return 0
		}
	})

	return desired, nil
}

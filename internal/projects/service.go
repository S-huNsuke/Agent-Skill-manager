package projects

import (
	"fmt"
	"strings"

	"github.com/caojun/agent-skills-manager/internal/domain"
)

const (
	ProjectTypePath    = "path"
	ProjectTypeVirtual = "virtual"
)

type Service struct{}

func (Service) Validate(project domain.Project) error {
	projectType := strings.TrimSpace(project.Type)
	path := strings.TrimSpace(project.Path)
	skillGroupID := strings.TrimSpace(project.SkillGroupID)

	if skillGroupID == "" {
		return fmt.Errorf("project %q must bind exactly one skill group", project.ID)
	}

	switch projectType {
	case ProjectTypePath:
		if path == "" {
			return fmt.Errorf("path project %q must include a path", project.ID)
		}
	case ProjectTypeVirtual:
		if path != "" {
			return fmt.Errorf("virtual project %q must not include a path", project.ID)
		}
		if project.AutoApplyEnabled {
			return fmt.Errorf("virtual project %q must activate manually", project.ID)
		}
	default:
		return fmt.Errorf("project %q must be either path or virtual", project.ID)
	}

	return nil
}

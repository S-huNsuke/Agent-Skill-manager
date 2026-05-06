package projects

import (
	"path/filepath"
	"strings"

	"github.com/caojun/agent-skills-manager/internal/domain"
)

const (
	ActivationStateNone     = "none"
	ActivationStateActivate = "activate"
	ActivationStateBlocked  = "blocked"
)

type ActivationDecision struct {
	State       string
	Project     *domain.Project
	BlockReason string
}

func (svc Service) ResolveAutomaticActivation(currentPath string, projects []domain.Project) ActivationDecision {
	normalizedCurrentPath := normalizePath(currentPath)
	var bestMatch *domain.Project
	bestSpecificity := -1
	tied := false

	for i := range projects {
		project := projects[i]
		if svc.Validate(project) != nil {
			continue
		}
		if strings.TrimSpace(project.Type) != ProjectTypePath || !project.AutoApplyEnabled {
			continue
		}

		projectPath := normalizePath(project.Path)
		if !hasExactPathPrefix(normalizedCurrentPath, projectPath) {
			continue
		}

		specificity := len(projectPath)
		switch {
		case specificity > bestSpecificity:
			bestMatch = &projects[i]
			bestSpecificity = specificity
			tied = false
		case specificity == bestSpecificity:
			tied = true
		}
	}

	if tied {
		return ActivationDecision{
			State:       ActivationStateBlocked,
			BlockReason: "multiple projects matched at the same specificity",
		}
	}

	if bestMatch == nil {
		return ActivationDecision{State: ActivationStateNone}
	}

	return ActivationDecision{
		State:   ActivationStateActivate,
		Project: bestMatch,
	}
}

func (svc Service) ResolveManualActivation(projectID string, projects []domain.Project) ActivationDecision {
	for i := range projects {
		project := projects[i]
		if svc.Validate(project) != nil {
			continue
		}
		if project.ID != projectID {
			continue
		}

		return ActivationDecision{
			State:   ActivationStateActivate,
			Project: &projects[i],
		}
	}

	return ActivationDecision{State: ActivationStateNone}
}

func normalizePath(value string) string {
	cleaned := filepath.Clean(strings.TrimSpace(value))
	return filepath.ToSlash(cleaned)
}

func hasExactPathPrefix(targetPath, prefix string) bool {
	if prefix == "." || prefix == "" {
		return false
	}

	if targetPath == prefix {
		return true
	}

	if prefix == "/" {
		return strings.HasPrefix(targetPath, "/")
	}

	return strings.HasPrefix(targetPath, prefix+"/")
}

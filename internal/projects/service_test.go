package projects

import (
	"testing"
	"time"

	"github.com/caojun/agent-skills-manager/internal/domain"
)

func TestResolveAutomaticActivationPrefersLongestExactPrefixPath(t *testing.T) {
	svc := Service{}
	now := time.Date(2026, time.May, 3, 10, 0, 0, 0, time.UTC)

	decision := svc.ResolveAutomaticActivation("/workspace/monorepo/apps/api/internal", []domain.Project{
		{
			ID:               "root",
			Name:             "root",
			Type:             " path ",
			Path:             "/workspace/monorepo",
			SkillGroupID:     "group-root",
			AutoApplyEnabled: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "api",
			Name:             "api",
			Type:             ProjectTypePath,
			Path:             "/workspace/monorepo/apps/api",
			SkillGroupID:     "group-api",
			AutoApplyEnabled: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "similar",
			Name:             "similar",
			Type:             ProjectTypePath,
			Path:             "/workspace/monorepo/apps/api-client",
			SkillGroupID:     "group-similar",
			AutoApplyEnabled: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	})

	if decision.State != ActivationStateActivate {
		t.Fatalf("expected activate state, got %q", decision.State)
	}

	if decision.Project == nil {
		t.Fatal("expected project to be selected")
	}

	if decision.Project.ID != "api" {
		t.Fatalf("expected api project to win, got %q", decision.Project.ID)
	}
}

func TestValidateRejectsProjectWithoutSkillGroupBinding(t *testing.T) {
	svc := Service{}

	err := svc.Validate(domain.Project{
		ID:   "project-1",
		Type: ProjectTypePath,
		Path: "/workspace/app",
	})
	if err == nil {
		t.Fatal("expected missing skill group binding to be rejected")
	}
}

func TestValidateRejectsPathAndVirtualShapeViolations(t *testing.T) {
	svc := Service{}

	tests := []struct {
		name    string
		project domain.Project
	}{
		{
			name: "path project requires path",
			project: domain.Project{
				ID:           "path-missing",
				Type:         ProjectTypePath,
				SkillGroupID: "group-1",
			},
		},
		{
			name: "virtual project forbids path",
			project: domain.Project{
				ID:           "virtual-with-path",
				Type:         ProjectTypeVirtual,
				Path:         "/workspace/virtual",
				SkillGroupID: "group-1",
			},
		},
		{
			name: "virtual project must activate manually",
			project: domain.Project{
				ID:               "virtual-auto",
				Type:             ProjectTypeVirtual,
				SkillGroupID:     "group-1",
				AutoApplyEnabled: true,
			},
		},
		{
			name: "project type must be path or virtual",
			project: domain.Project{
				ID:           "unknown",
				Type:         "other",
				SkillGroupID: "group-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.Validate(tt.project); err == nil {
				t.Fatalf("expected validation failure for %+v", tt.project)
			}
		})
	}
}

func TestResolveAutomaticActivationBlocksTieAtSameSpecificity(t *testing.T) {
	svc := Service{}
	now := time.Date(2026, time.May, 3, 10, 0, 0, 0, time.UTC)

	decision := svc.ResolveAutomaticActivation("/workspace/app/pkg", []domain.Project{
		{
			ID:               "a",
			Type:             ProjectTypePath,
			Path:             "/workspace/app",
			SkillGroupID:     "group-a",
			AutoApplyEnabled: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "b",
			Type:             ProjectTypePath,
			Path:             "/workspace/app",
			SkillGroupID:     "group-b",
			AutoApplyEnabled: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	})

	if decision.State != ActivationStateBlocked {
		t.Fatalf("expected blocked state, got %q", decision.State)
	}

	if decision.BlockReason == "" {
		t.Fatal("expected block reason for tie")
	}

	if decision.Project != nil {
		t.Fatalf("expected no selected project, got %q", decision.Project.ID)
	}
}

func TestResolveAutomaticActivationMatchesRootScopedProject(t *testing.T) {
	svc := Service{}
	now := time.Date(2026, time.May, 3, 10, 0, 0, 0, time.UTC)

	decision := svc.ResolveAutomaticActivation("/workspace/app", []domain.Project{
		{
			ID:               "root",
			Type:             ProjectTypePath,
			Path:             "/",
			SkillGroupID:     "group-root",
			AutoApplyEnabled: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	})

	if decision.State != ActivationStateActivate {
		t.Fatalf("expected activate state, got %q", decision.State)
	}

	if decision.Project == nil || decision.Project.ID != "root" {
		t.Fatalf("expected root project to match, got %+v", decision.Project)
	}
}

func TestResolveManualActivationAllowsVirtualProject(t *testing.T) {
	svc := Service{}
	now := time.Date(2026, time.May, 3, 10, 0, 0, 0, time.UTC)

	decision := svc.ResolveManualActivation("virtual-1", []domain.Project{
		{
			ID:               "virtual-1",
			Type:             ProjectTypeVirtual,
			SkillGroupID:     "group-virtual",
			AutoApplyEnabled: false,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	})

	if decision.State != ActivationStateActivate {
		t.Fatalf("expected activate state, got %q", decision.State)
	}

	if decision.Project == nil {
		t.Fatal("expected virtual project to be selected manually")
	}

	if decision.Project.Type != ProjectTypeVirtual {
		t.Fatalf("expected virtual project type, got %q", decision.Project.Type)
	}
}

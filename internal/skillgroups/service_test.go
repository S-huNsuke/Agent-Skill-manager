package skillgroups

import (
	"testing"

	"github.com/caojun/agent-skills-manager/internal/domain"
)

func TestValidateRejectsSkillLinkedToDifferentGroup(t *testing.T) {
	svc := Service{}

	err := svc.Validate(domain.SkillGroup{ID: "group-1", Name: "Backend"}, []domain.SkillGroupSkill{
		{SkillGroupID: "group-2", SkillID: "skill-a"},
	})
	if err == nil {
		t.Fatal("expected mismatched skill-group binding to fail validation")
	}
}

func TestDesiredSkillsRejectsDuplicateSkillIDs(t *testing.T) {
	svc := Service{}

	_, err := svc.DesiredSkills(domain.SkillGroup{ID: "group-1", Name: "Backend"}, []domain.SkillGroupSkill{
		{SkillGroupID: "group-1", SkillID: "skill-a", Priority: 10},
		{SkillGroupID: "group-1", SkillID: "skill-a", Priority: 5},
	})
	if err == nil {
		t.Fatal("expected duplicate skill IDs to be rejected")
	}
}

func TestDesiredSkillsReturnsValidatedBindings(t *testing.T) {
	svc := Service{}

	desired, err := svc.DesiredSkills(domain.SkillGroup{ID: "group-1", Name: "Backend"}, []domain.SkillGroupSkill{
		{SkillGroupID: "group-1", SkillID: "skill-b", Priority: 5},
		{SkillGroupID: "group-1", SkillID: "skill-a", Priority: 10, Required: true},
	})
	if err != nil {
		t.Fatalf("expected desired skills, got error: %v", err)
	}

	if len(desired) != 2 {
		t.Fatalf("expected 2 desired skills, got %d", len(desired))
	}

	if desired[0].SkillID != "skill-a" || !desired[0].Required {
		t.Fatalf("expected highest-priority required skill first, got %+v", desired[0])
	}

	if desired[1].SkillID != "skill-b" {
		t.Fatalf("expected second skill to be skill-b, got %+v", desired[1])
	}
}

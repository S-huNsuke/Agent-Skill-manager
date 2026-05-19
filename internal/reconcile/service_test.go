package reconcile

import (
	"testing"

	"github.com/caojun/agent-skills-manager/internal/domain"
	"github.com/caojun/agent-skills-manager/internal/skillgroups"
)

func TestPlanSchedulesInstallUpdateAndRepair(t *testing.T) {
	svc := Service{}

	plan, err := svc.Plan(
		[]skillgroups.DesiredSkill{
			{SkillID: "skill-install"},
			{SkillID: "skill-update"},
			{SkillID: "skill-repair"},
		},
		[]domain.CatalogSkill{
			{ID: "skill-install", Version: "1.0.0"},
			{ID: "skill-update", Version: "2.0.0"},
			{ID: "skill-repair", Version: "1.5.0"},
		},
		[]domain.InstalledSkill{
			{SkillID: "skill-update", Version: "1.0.0", InstallState: "installed"},
			{SkillID: "skill-repair", Version: "1.5.0", InstallState: "error", ErrorMessage: "checksum mismatch"},
		},
	)
	if err != nil {
		t.Fatalf("expected reconcile plan, got error: %v", err)
	}

	if len(plan.Install) != 1 || plan.Install[0].SkillID != "skill-install" {
		t.Fatalf("expected install action for skill-install, got %+v", plan.Install)
	}

	if len(plan.Update) != 1 || plan.Update[0].SkillID != "skill-update" {
		t.Fatalf("expected update action for skill-update, got %+v", plan.Update)
	}

	if len(plan.Repair) != 1 || plan.Repair[0].SkillID != "skill-repair" {
		t.Fatalf("expected repair action for skill-repair, got %+v", plan.Repair)
	}

	if plan.BlockReason != "" {
		t.Fatalf("expected no block reason, got %q", plan.BlockReason)
	}
}

func TestPlanBlocksWhenCatalogEntryIsMissing(t *testing.T) {
	svc := Service{}

	plan, err := svc.Plan(
		[]skillgroups.DesiredSkill{{SkillID: "skill-missing"}},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("expected blocked plan without hard error, got %v", err)
	}

	if plan.BlockReason == "" {
		t.Fatal("expected block reason when catalog entry is missing")
	}

	if len(plan.Install) != 0 || len(plan.Update) != 0 || len(plan.Repair) != 0 {
		t.Fatalf("expected no actions when blocked, got %+v", plan)
	}

	if plan.Install == nil || plan.Update == nil || plan.Repair == nil {
		t.Fatalf("expected blocked plan to keep empty action lists, got %+v", plan)
	}
}

func TestPlanBlocksWithoutReturningPartialActions(t *testing.T) {
	svc := Service{}

	plan, err := svc.Plan(
		[]skillgroups.DesiredSkill{
			{SkillID: "skill-known"},
			{SkillID: "skill-missing"},
		},
		[]domain.CatalogSkill{{ID: "skill-known", Version: "1.0.0"}},
		nil,
	)
	if err != nil {
		t.Fatalf("expected blocked plan without hard error, got %v", err)
	}

	if plan.BlockReason == "" {
		t.Fatal("expected block reason when a later catalog entry is missing")
	}

	if len(plan.Install) != 0 || len(plan.Update) != 0 || len(plan.Repair) != 0 {
		t.Fatalf("expected blocked plan to omit partial actions, got %+v", plan)
	}
}

func TestPlanDoesNotUninstallUnrelatedWorkingSkills(t *testing.T) {
	svc := Service{}

	plan, err := svc.Plan(
		[]skillgroups.DesiredSkill{{SkillID: "skill-a"}},
		[]domain.CatalogSkill{{ID: "skill-a", Version: "1.0.0"}},
		[]domain.InstalledSkill{
			{SkillID: "skill-a", Version: "1.0.0", InstallState: "installed"},
			{SkillID: "skill-unrelated", Version: "9.9.9", InstallState: "installed"},
		},
	)
	if err != nil {
		t.Fatalf("expected reconcile plan, got error: %v", err)
	}

	if len(plan.Install) != 0 || len(plan.Update) != 0 || len(plan.Repair) != 0 {
		t.Fatalf("expected no actions for already-satisfied desired skills, got %+v", plan)
	}

	if plan.BlockReason != "" {
		t.Fatalf("expected no block reason, got %q", plan.BlockReason)
	}
}

func TestPlanMatchesCatalogBySkillName(t *testing.T) {
	svc := Service{}

	plan, err := svc.Plan(
		[]skillgroups.DesiredSkill{{SkillID: "skill-a"}},
		[]domain.CatalogSkill{{ID: "source-one-skill-a", Name: "skill-a", Version: "1.2.3"}},
		nil,
	)
	if err != nil {
		t.Fatalf("expected reconcile plan, got error: %v", err)
	}

	if len(plan.Install) != 1 {
		t.Fatalf("expected one install action, got %+v", plan.Install)
	}

	if plan.Install[0].SkillID != "skill-a" {
		t.Fatalf("expected action to use skill name, got %q", plan.Install[0].SkillID)
	}
}

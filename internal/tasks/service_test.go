package tasks

import "testing"

func TestTaskCannotLoopRecoveryWithoutRetryIncrement(t *testing.T) {
	svc := Service{}

	task := Record{
		ID:                  "task-1",
		Status:              StatusVerifying,
		PlanRevision:        1,
		DefaultRecoveryUsed: true,
	}

	updated := svc.RequestDefaultRecovery(task, RecoveryPrerequisites{
		IsAppManaged:      true,
		HasUsableArtifact: true,
		AdapterOwnsTarget: true,
	})

	if updated.Status != StatusBlocked {
		t.Fatalf("Status = %q, want %q", updated.Status, StatusBlocked)
	}

	if updated.RetryCount != 0 {
		t.Fatalf("RetryCount = %d, want 0", updated.RetryCount)
	}
}

func TestTaskCanRetryRecoveryAfterPlanRevision(t *testing.T) {
	svc := Service{}

	revised := svc.RevisePlan(Record{
		ID:                  "task-1",
		Status:              StatusBlocked,
		PlanRevision:        1,
		RetryCount:          0,
		DefaultRecoveryUsed: true,
	})

	if revised.Status != StatusPlanning {
		t.Fatalf("Status = %q, want %q", revised.Status, StatusPlanning)
	}
	if revised.RetryCount != 1 {
		t.Fatalf("RetryCount = %d, want 1", revised.RetryCount)
	}
	if revised.PlanRevision != 2 {
		t.Fatalf("PlanRevision = %d, want 2", revised.PlanRevision)
	}
	if revised.DefaultRecoveryUsed {
		t.Fatal("DefaultRecoveryUsed = true, want false")
	}

	recovered := svc.RequestDefaultRecovery(Record{
		ID:           revised.ID,
		Status:       StatusVerifying,
		PlanRevision: revised.PlanRevision,
		RetryCount:   revised.RetryCount,
	}, RecoveryPrerequisites{
		IsSafelyReplaceable: true,
		HasUsableArtifact:   true,
		AdapterOwnsTarget:   true,
	})

	if recovered.Status != StatusRecovering {
		t.Fatalf("Status = %q, want %q", recovered.Status, StatusRecovering)
	}
	if !recovered.DefaultRecoveryUsed {
		t.Fatal("DefaultRecoveryUsed = false, want true")
	}
}

func TestTaskBlocksWhenRecoveryPrerequisitesAreUnsatisfied(t *testing.T) {
	svc := Service{}

	updated := svc.RequestDefaultRecovery(Record{
		ID:     "task-1",
		Status: StatusVerifying,
	}, RecoveryPrerequisites{
		IsAppManaged:      true,
		HasUsableArtifact: false,
		AdapterOwnsTarget: true,
	})

	if updated.Status != StatusBlocked {
		t.Fatalf("Status = %q, want %q", updated.Status, StatusBlocked)
	}
	if updated.StatusReason == "" {
		t.Fatal("StatusReason is empty, want recovery block reason")
	}
}

func TestAdvanceRejectsInvalidTerminalTransition(t *testing.T) {
	svc := Service{}

	_, err := svc.Advance(Record{Status: StatusCompleted}, StatusRecovering)
	if err == nil {
		t.Fatal("Advance() error = nil, want invalid transition error")
	}
}

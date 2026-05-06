package tasks

type Status string

const (
	StatusQueued     Status = "queued"
	StatusPlanning   Status = "planning"
	StatusResolving  Status = "resolving"
	StatusExecuting  Status = "executing"
	StatusVerifying  Status = "verifying"
	StatusRecovering Status = "recovering"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusBlocked    Status = "blocked"
	StatusCancelled  Status = "cancelled"
)

type Record struct {
	ID                  string
	Status              Status
	RetryCount          int
	PlanRevision        int
	DefaultRecoveryUsed bool
	StatusReason        string
}

type RecoveryPrerequisites struct {
	IsAppManaged        bool
	IsSafelyReplaceable bool
	HasUsableArtifact   bool
	AdapterOwnsTarget   bool
}

type RecoveryDecision struct {
	Allowed bool
	Reason  string
}

package tasks

import "fmt"

type Service struct{}

func (Service) Advance(task Record, next Status) (Record, error) {
	if !canTransition(task.Status, next) {
		return Record{}, fmt.Errorf("invalid task transition from %q to %q", task.Status, next)
	}

	task.Status = next
	task.StatusReason = ""
	return task, nil
}

func (s Service) RequestDefaultRecovery(task Record, prerequisites RecoveryPrerequisites) Record {
	decision := CanAutoRecover(prerequisites)
	if !decision.Allowed {
		task.Status = StatusBlocked
		task.StatusReason = decision.Reason
		return task
	}

	if task.DefaultRecoveryUsed {
		task.Status = StatusBlocked
		task.StatusReason = "default recovery already used; revise plan and increment retry"
		return task
	}

	task.Status = StatusRecovering
	task.DefaultRecoveryUsed = true
	task.StatusReason = ""
	return task
}

func (Service) RevisePlan(task Record) Record {
	task.Status = StatusPlanning
	task.RetryCount++
	task.PlanRevision++
	task.DefaultRecoveryUsed = false
	task.StatusReason = ""
	return task
}

func canTransition(current, next Status) bool {
	switch current {
	case StatusQueued:
		return next == StatusPlanning || next == StatusCancelled
	case StatusPlanning:
		return next == StatusResolving || next == StatusBlocked || next == StatusFailed || next == StatusCancelled
	case StatusResolving:
		return next == StatusExecuting || next == StatusBlocked || next == StatusFailed || next == StatusCancelled
	case StatusExecuting:
		return next == StatusVerifying || next == StatusRecovering || next == StatusBlocked || next == StatusFailed || next == StatusCancelled
	case StatusVerifying:
		return next == StatusCompleted || next == StatusRecovering || next == StatusBlocked || next == StatusFailed || next == StatusCancelled
	case StatusRecovering:
		return next == StatusVerifying || next == StatusPlanning || next == StatusBlocked || next == StatusFailed || next == StatusCancelled
	case StatusCompleted, StatusFailed, StatusBlocked, StatusCancelled:
		return false
	default:
		return false
	}
}

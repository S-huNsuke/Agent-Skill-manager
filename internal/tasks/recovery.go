package tasks

func CanAutoRecover(prerequisites RecoveryPrerequisites) RecoveryDecision {
	if !prerequisites.IsAppManaged && !prerequisites.IsSafelyReplaceable {
		return RecoveryDecision{
			Allowed: false,
			Reason:  "target is not app-managed or safely replaceable",
		}
	}

	if !prerequisites.HasUsableArtifact {
		return RecoveryDecision{
			Allowed: false,
			Reason:  "no valid package or cached artifact available",
		}
	}

	if !prerequisites.AdapterOwnsTarget {
		return RecoveryDecision{
			Allowed: false,
			Reason:  "adapter cannot prove target path ownership",
		}
	}

	return RecoveryDecision{Allowed: true}
}

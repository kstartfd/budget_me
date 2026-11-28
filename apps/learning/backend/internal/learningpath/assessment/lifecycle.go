package assessment

import "fmt"

// TransitionSection checks a section's next durable state. A terminal result
// cannot be reopened: a repeat assessment is a new attempt, not a mutation of
// old evidence. Quick mirrors its parent test lifecycle into its persisted
// grammar section while old binaries remain in use.
func TransitionSection(current, next string) error {
	if !validSectionStatus(current) || !validSectionStatus(next) {
		return fmt.Errorf("invalid assessment section transition %q -> %q", current, next)
	}
	if current == next {
		return nil // repeated pause/state update is idempotent
	}
	allowed := map[string]map[string]bool{
		Created: {
			InProgress: true, Paused: true, Evaluating: true,
			Completed: true, Partial: true, Failed: true, Cancelled: true,
		},
		InProgress: {
			Paused: true, Evaluating: true,
			Completed: true, Partial: true, Failed: true, Cancelled: true,
		},
		Paused: {
			InProgress: true, Evaluating: true,
			Completed: true, Partial: true, Failed: true, Cancelled: true,
		},
		Evaluating: {
			InProgress: true, Paused: true,
			Completed: true, Partial: true, Failed: true, Cancelled: true,
		},
	}
	if !allowed[current][next] {
		return fmt.Errorf("invalid assessment section transition %q -> %q", current, next)
	}
	return nil
}

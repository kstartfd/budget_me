package commands

import (
	"fmt"
	"time"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/llm"
)

// Budget is what one chat may spend with the learner's agent within a window:
// there is little point in studying a single phrase for hours. It counts every
// token a turn costs (prompt and reply), so a long history weighs as much as it
// is paid for. Clearing a chat does not give it back.
type Budget struct {
	Limit int64
	Used  int64
	// ResetsAt is when the window ends and the chat starts afresh; nil while
	// nothing has been spent in a window.
	ResetsAt *time.Time
}

// BudgetOf is the budget of a chat at `now`. A window that has ended counts as
// untouched, whatever the stored numbers still say.
func BudgetOf(chat db.LearningChat, profile llm.ChatProfile, now time.Time) Budget {
	budget := Budget{Limit: int64(profile.BudgetTokens)}

	if !chat.SpentSince.Valid {
		return budget
	}

	ends := chat.SpentSince.Time.Add(profile.BudgetWindow)
	if !ends.After(now) {
		return budget
	}

	budget.Used = chat.SpentTokens
	budget.ResetsAt = &ends

	return budget
}

// Exhausted: the next question has to wait for the window to end. A turn that
// starts below the limit may end above it, since its size is not known in
// advance, so the limit is soft by one turn.
func (b Budget) Exhausted() bool { return b.Used >= b.Limit }

// After is the budget once a turn costing `spent` has been recorded at `now`.
func (b Budget) After(spent int, now time.Time, window time.Duration) Budget {
	if b.ResetsAt == nil {
		ends := now.Add(window)
		b.ResetsAt = &ends
	}

	b.Used += int64(spent)

	return b
}

// BudgetError is a question refused because its chat has used up its budget.
type BudgetError struct{ ResetsAt time.Time }

func (e *BudgetError) Error() string {
	return fmt.Sprintf("the chat has used up its tokens until %s", e.ResetsAt.UTC().Format(time.RFC3339))
}

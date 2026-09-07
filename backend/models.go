package main

import (
	"database/sql"
	"time"
)

// Chore is a task tracked on a recurring or one-off countdown, mirroring
// how a Roomba tracks remaining hours on a replaceable part.
type Chore struct {
	ID                  int64        `json:"id"`
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	IntervalHours       float64      `json:"interval_hours"`
	Recurring           bool         `json:"recurring"`
	LastCompletedAt     time.Time    `json:"last_completed_at"`
	PreviousCompletedAt sql.NullTime `json:"-"`
	CreatedAt           time.Time    `json:"created_at"`
}

// ChoreView adds the derived, time-based fields the UI renders directly.
type ChoreView struct {
	Chore
	HoursLeft        float64 `json:"hours_left"`
	PercentRemaining float64 `json:"percent_remaining"`
	Overdue          bool    `json:"overdue"`
	CanUndo          bool    `json:"can_undo"`
}

func toView(c Chore) ChoreView {
	// A recurring chore's countdown starts fresh at the local midnight after
	// it was completed, not at the exact moment it was marked done - it's
	// already off the active list until then (see isCompletedNow), so it
	// should reappear with its full interval intact rather than however much
	// had already ticked away since the actual click. A chore that's never
	// been completed counts down from creation as always; this only applies
	// to an actual reset.
	countdownStart := c.LastCompletedAt
	if c.Recurring && hasBeenCompleted(c) {
		countdownStart = startOfNextLocalDay(c.LastCompletedAt)
	}

	elapsed := time.Since(countdownStart).Hours()
	hoursLeft := c.IntervalHours - elapsed

	percent := 0.0
	if c.IntervalHours > 0 {
		percent = (hoursLeft / c.IntervalHours) * 100
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	return ChoreView{
		Chore:            c,
		HoursLeft:        hoursLeft,
		PercentRemaining: percent,
		Overdue:          hoursLeft <= 0,
		CanUndo:          c.PreviousCompletedAt.Valid,
	}
}

// hasBeenCompleted reports whether a chore has ever been marked done. A
// brand-new chore sets last_completed_at equal to created_at, so equality
// means "never completed" without needing a separate column.
func hasBeenCompleted(c Chore) bool {
	return !c.LastCompletedAt.Equal(c.CreatedAt)
}

func sameLocalDay(a, b time.Time) bool {
	a, b = a.Local(), b.Local()
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// startOfNextLocalDay returns local midnight for the calendar day after t.
func startOfNextLocalDay(t time.Time) time.Time {
	t = t.Local()
	y, m, d := t.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, t.Location())
}

// isCompletedNow reports whether a chore currently belongs on the Completed
// tab rather than the active list. A recurring chore stays there until
// local midnight (then it's due again, same as any other active chore); a
// non-recurring chore stays there for a full 24 hours before being cleared
// out entirely by pruneExpiredChores.
func isCompletedNow(c Chore) bool {
	if !hasBeenCompleted(c) {
		return false
	}
	if c.Recurring {
		return sameLocalDay(c.LastCompletedAt, time.Now())
	}
	return time.Since(c.LastCompletedAt) < 24*time.Hour
}

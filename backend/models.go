package main

import (
	"database/sql"
	"time"
)

// Chore is a task tracked on a recurring or one-off countdown, mirroring
// how a Roomba tracks remaining hours on a replaceable part.
//
// LastCompletedAt and CountdownStartedAt look redundant but answer different
// questions. LastCompletedAt is purely "when was Done last clicked" - it
// drives isCompletedNow (the Completed-tab hide-until-midnight/24h window)
// and the "COMPLETED X AGO" label, and nothing else ever touches it.
// CountdownStartedAt is "what point in time is the interval measured from"
// - it's what toView actually uses for hours_left/percent, and unlike
// LastCompletedAt it also gets rebased by editing the chore (see
// updateChore). Keeping them separate means editing a chore's duration can
// never be mistaken for a fresh Done click.
type Chore struct {
	ID                         int64        `json:"id"`
	Name                       string       `json:"name"`
	Description                string       `json:"description"`
	IntervalHours              float64      `json:"interval_hours"`
	Recurring                  bool         `json:"recurring"`
	LastCompletedAt            time.Time    `json:"last_completed_at"`
	PreviousCompletedAt        sql.NullTime `json:"-"`
	CountdownStartedAt         time.Time    `json:"-"`
	PreviousCountdownStartedAt sql.NullTime `json:"-"`
	CreatedAt                  time.Time    `json:"created_at"`
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
	elapsed := time.Since(c.CountdownStartedAt).Hours()
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

// isDailyRecurring reports whether a chore is recurring with an interval of
// a day or more - the cutoff for hiding until local midnight rather than
// just reactivating once its own interval elapses. A chore that recurs
// every couple hours shouldn't vanish until midnight over a single
// completion; only daily-or-longer chores get that treatment.
func isDailyRecurring(c Chore) bool {
	return c.Recurring && c.IntervalHours >= 24
}

// isCompletedNow reports whether a chore currently belongs on the Completed
// tab rather than the active list. A daily-or-longer recurring chore stays
// there until local midnight (then it's due again, same as any other active
// chore); a shorter-cycle recurring chore or a non-recurring one stays
// there until its own interval elapses (capped at 24 hours for non-recurring,
// which otherwise has no interval-driven reason to ever be pruned).
func isCompletedNow(c Chore) bool {
	if !hasBeenCompleted(c) {
		return false
	}
	if isDailyRecurring(c) {
		return sameLocalDay(c.LastCompletedAt, time.Now())
	}
	if c.Recurring {
		return time.Since(c.LastCompletedAt) < time.Duration(c.IntervalHours*float64(time.Hour))
	}
	return time.Since(c.LastCompletedAt) < 24*time.Hour
}

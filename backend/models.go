package main

import "time"

// Chore is a task tracked on a recurring or one-off countdown, mirroring
// how a Roomba tracks remaining hours on a replaceable part.
type Chore struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	IntervalHours   float64   `json:"interval_hours"`
	Recurring       bool      `json:"recurring"`
	LastCompletedAt time.Time `json:"last_completed_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// ChoreView adds the derived, time-based fields the UI renders directly.
type ChoreView struct {
	Chore
	HoursLeft        float64 `json:"hours_left"`
	PercentRemaining float64 `json:"percent_remaining"`
	Overdue          bool    `json:"overdue"`
}

func toView(c Chore) ChoreView {
	elapsed := time.Since(c.LastCompletedAt).Hours()
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
	}
}

// hiddenForToday reports whether a chore was just marked done and should stay
// off the list until local midnight, so completed chores don't linger and
// clutter the view for the rest of the day.
func hiddenForToday(c Chore) bool {
	if !c.Recurring || c.LastCompletedAt.Equal(c.CreatedAt) {
		// Never been completed (a brand-new chore sets both to the same
		// timestamp), so there's nothing to hide.
		return false
	}
	completed := c.LastCompletedAt.Local()
	now := time.Now().Local()
	y1, m1, d1 := completed.Date()
	y2, m2, d2 := now.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

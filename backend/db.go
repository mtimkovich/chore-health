package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	const schema = `
	CREATE TABLE IF NOT EXISTS chores (
		id                              INTEGER PRIMARY KEY AUTOINCREMENT,
		name                            TEXT NOT NULL,
		description                     TEXT NOT NULL DEFAULT '',
		interval_hours                  REAL NOT NULL,
		recurring                       INTEGER NOT NULL DEFAULT 1,
		last_completed_at               DATETIME NOT NULL,
		previous_completed_at           DATETIME,
		countdown_started_at            DATETIME NOT NULL,
		previous_countdown_started_at   DATETIME,
		created_at                      DATETIME NOT NULL
	);`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	// Databases created before these columns existed need them added
	// separately; CREATE TABLE IF NOT EXISTS is a no-op for them.
	for _, migration := range []string{
		`ALTER TABLE chores ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE chores ADD COLUMN previous_completed_at DATETIME`,
		`ALTER TABLE chores ADD COLUMN countdown_started_at DATETIME`,
		`ALTER TABLE chores ADD COLUMN previous_countdown_started_at DATETIME`,
	} {
		if _, err := db.Exec(migration); err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return nil, err
			}
		}
	}

	// Backfill countdown_started_at for rows from before it existed (it
	// can't carry a static DEFAULT since it's derived from another column).
	// last_completed_at is a reasonable stand-in: for a never-completed
	// chore it already equals created_at, and for one that's been completed
	// it just means one fewer "reset to full" cycle at its next midnight,
	// which corrects itself the next time it's completed or edited.
	if _, err := db.Exec(`UPDATE chores SET countdown_started_at = last_completed_at WHERE countdown_started_at IS NULL`); err != nil {
		return nil, err
	}

	return db, nil
}

const choreColumns = `id, name, description, interval_hours, recurring, last_completed_at, previous_completed_at, countdown_started_at, previous_countdown_started_at, created_at`

func scanChore(row interface{ Scan(...any) error }) (Chore, error) {
	var c Chore
	var recurring int
	err := row.Scan(
		&c.ID, &c.Name, &c.Description, &c.IntervalHours, &recurring,
		&c.LastCompletedAt, &c.PreviousCompletedAt,
		&c.CountdownStartedAt, &c.PreviousCountdownStartedAt,
		&c.CreatedAt,
	)
	c.Recurring = recurring != 0
	return c, err
}

func listChores(db *sql.DB) ([]Chore, error) {
	rows, err := db.Query(`SELECT ` + choreColumns + ` FROM chores`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Chore
	for rows.Next() {
		c, err := scanChore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func getChore(db *sql.DB, id int64) (Chore, error) {
	row := db.QueryRow(`SELECT `+choreColumns+` FROM chores WHERE id = ?`, id)
	return scanChore(row)
}

func createChore(db *sql.DB, name, description string, intervalHours float64, recurring bool) (Chore, error) {
	now := time.Now().UTC()
	res, err := db.Exec(
		`INSERT INTO chores (name, description, interval_hours, recurring, last_completed_at, countdown_started_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		name, description, intervalHours, recurring, now, now, now,
	)
	if err != nil {
		return Chore{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Chore{}, err
	}
	return getChore(db, id)
}

// updateChore saves the edited fields. If the chore has been completed at
// least once before AND intervalHours is actually changing, it also rebases
// countdown_started_at to now, so the new interval is measured "from now" -
// otherwise it stays anchored to whenever it was last marked done, which
// goes stale: editing a chore to a different duration would still be
// measured from the wrong starting point. Editing anything else (name,
// description, recurring) leaves countdown_started_at alone, since none of
// that affects hours_left - a rename shouldn't reset the countdown. A chore
// that's never been completed doesn't have this problem (its countdown
// already runs from created_at), so it's left alone regardless.
//
// This deliberately never touches last_completed_at - only completeChore
// does that. Rebasing it here too would make isCompletedNow think the chore
// had just been marked done, hiding it until midnight for no reason.
func updateChore(db *sql.DB, id int64, name, description string, intervalHours float64, recurring bool) error {
	current, err := getChore(db, id)
	if err != nil {
		return err
	}

	if hasBeenCompleted(current) && intervalHours != current.IntervalHours {
		countdownStart := time.Now().UTC()
		if isDailyRecurring(current) && isCompletedNow(current) {
			// Still hidden on the Completed tab pending midnight - keep the
			// same "full interval once it's actually due again" behavior as
			// completing it does, instead of starting the clock immediately.
			countdownStart = startOfNextLocalDay(countdownStart)
		}
		_, err = db.Exec(
			`UPDATE chores SET name = ?, description = ?, interval_hours = ?, recurring = ?, countdown_started_at = ? WHERE id = ?`,
			name, description, intervalHours, recurring, countdownStart, id,
		)
		return err
	}

	_, err = db.Exec(
		`UPDATE chores SET name = ?, description = ?, interval_hours = ?, recurring = ? WHERE id = ?`,
		name, description, intervalHours, recurring, id,
	)
	return err
}

// completeChore stamps last_completed_at with now (saving the old value into
// previous_completed_at so a single undoComplete can reverse it), and resets
// countdown_started_at to match. A daily-or-longer recurring chore's
// countdown starts fresh at the local midnight after completion rather than
// the exact click, since it's already off the active list until then (see
// isCompletedNow) - it should reappear with its full interval intact rather
// than however much had already ticked away since the click. A shorter-cycle
// recurring chore just starts counting down immediately from the click.
func completeChore(db *sql.DB, id int64) error {
	current, err := getChore(db, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	countdownStart := now
	if isDailyRecurring(current) {
		countdownStart = startOfNextLocalDay(now)
	}

	_, err = db.Exec(
		`UPDATE chores SET
			previous_completed_at = last_completed_at, last_completed_at = ?,
			previous_countdown_started_at = countdown_started_at, countdown_started_at = ?
		 WHERE id = ?`,
		now, countdownStart, id,
	)
	return err
}

var errNothingToUndo = errors.New("nothing to undo")

// undoComplete reverses the most recent completeChore call by restoring
// last_completed_at/countdown_started_at from their "previous" columns.
// Only one level of undo is kept - undoing twice in a row without
// completing in between does nothing.
func undoComplete(db *sql.DB, id int64) error {
	res, err := db.Exec(
		`UPDATE chores SET
			last_completed_at = previous_completed_at, previous_completed_at = NULL,
			countdown_started_at = previous_countdown_started_at, previous_countdown_started_at = NULL
		 WHERE id = ? AND previous_completed_at IS NOT NULL`,
		id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errNothingToUndo
	}
	return nil
}

// snoozeChore pushes a chore's countdown out by the given number of hours,
// capped so it can never end up with more than a full interval remaining.
// Unlike completeChore it only touches countdown_started_at - last_completed_at
// (and so isCompletedNow/the Completed tab) is untouched, since snoozing
// isn't marking the chore done, just deferring it.
func snoozeChore(db *sql.DB, id int64, hours float64) error {
	current, err := getChore(db, id)
	if err != nil {
		return err
	}

	countdownStart := current.CountdownStartedAt.Add(time.Duration(hours * float64(time.Hour)))
	if now := time.Now().UTC(); countdownStart.After(now) {
		countdownStart = now
	}

	_, err = db.Exec(`UPDATE chores SET countdown_started_at = ? WHERE id = ?`, countdownStart, id)
	return err
}

func deleteChore(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM chores WHERE id = ?`, id)
	return err
}

// pruneExpiredChores permanently deletes non-recurring chores that have sat
// completed past isCompletedNow's 24-hour window, and returns what's left.
func pruneExpiredChores(db *sql.DB, chores []Chore) ([]Chore, error) {
	remaining := chores[:0]
	for _, c := range chores {
		if !c.Recurring && hasBeenCompleted(c) && !isCompletedNow(c) {
			if err := deleteChore(db, c.ID); err != nil {
				return nil, err
			}
			continue
		}
		remaining = append(remaining, c)
	}
	return remaining, nil
}

func loadPrunedChores(db *sql.DB) ([]Chore, error) {
	chores, err := listChores(db)
	if err != nil {
		return nil, err
	}
	return pruneExpiredChores(db, chores)
}

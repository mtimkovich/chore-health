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
		id                 INTEGER PRIMARY KEY AUTOINCREMENT,
		name               TEXT NOT NULL,
		description        TEXT NOT NULL DEFAULT '',
		interval_hours     REAL NOT NULL,
		recurring          INTEGER NOT NULL DEFAULT 1,
		last_completed_at  DATETIME NOT NULL,
		created_at         DATETIME NOT NULL
	);`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	// Databases created before the description column existed need it added
	// separately; CREATE TABLE IF NOT EXISTS is a no-op for them.
	if _, err := db.Exec(`ALTER TABLE chores ADD COLUMN description TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}

	const settingsSchema = `
	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`
	if _, err := db.Exec(settingsSchema); err != nil {
		return nil, err
	}

	return db, nil
}

// getSetting returns "" if the key has never been set.
func getSetting(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func listChores(db *sql.DB) ([]Chore, error) {
	rows, err := db.Query(`SELECT id, name, description, interval_hours, recurring, last_completed_at, created_at FROM chores`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Chore
	for rows.Next() {
		var c Chore
		var recurring int
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.IntervalHours, &recurring, &c.LastCompletedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Recurring = recurring != 0
		out = append(out, c)
	}
	return out, rows.Err()
}

func getChore(db *sql.DB, id int64) (Chore, error) {
	var c Chore
	var recurring int
	err := db.QueryRow(`SELECT id, name, description, interval_hours, recurring, last_completed_at, created_at FROM chores WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Description, &c.IntervalHours, &recurring, &c.LastCompletedAt, &c.CreatedAt)
	c.Recurring = recurring != 0
	return c, err
}

func createChore(db *sql.DB, name, description string, intervalHours float64, recurring bool) (Chore, error) {
	now := time.Now().UTC()
	res, err := db.Exec(
		`INSERT INTO chores (name, description, interval_hours, recurring, last_completed_at, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		name, description, intervalHours, recurring, now, now,
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

func updateChore(db *sql.DB, id int64, name, description string, intervalHours float64, recurring bool) error {
	_, err := db.Exec(
		`UPDATE chores SET name = ?, description = ?, interval_hours = ?, recurring = ? WHERE id = ?`,
		name, description, intervalHours, recurring, id,
	)
	return err
}

func completeChore(db *sql.DB, id int64) error {
	_, err := db.Exec(`UPDATE chores SET last_completed_at = ? WHERE id = ?`, time.Now().UTC(), id)
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

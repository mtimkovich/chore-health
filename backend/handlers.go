package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
)

type choreRequest struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	IntervalHours float64 `json:"interval_hours"`
	Recurring     bool    `json:"recurring"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func handleListChores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chores, err := loadPrunedChores(db)
		if err != nil {
			log.Println("loadPrunedChores:", err)
			writeError(w, http.StatusInternalServerError, "failed to list chores")
			return
		}

		views := make([]ChoreView, 0, len(chores))
		for _, c := range chores {
			if isCompletedNow(c) {
				continue
			}
			views = append(views, toView(c))
		}
		// Soonest due first. Percent-remaining would rank by how "worn" a
		// chore's own interval is, so a brand-new chore (always near 100%)
		// would sink to the bottom regardless of how soon it's actually due.
		sort.Slice(views, func(i, j int) bool {
			return views[i].HoursLeft < views[j].HoursLeft
		})

		writeJSON(w, http.StatusOK, views)
	}
}

func handleListCompletedChores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chores, err := loadPrunedChores(db)
		if err != nil {
			log.Println("loadPrunedChores:", err)
			writeError(w, http.StatusInternalServerError, "failed to list completed chores")
			return
		}

		views := make([]ChoreView, 0, len(chores))
		for _, c := range chores {
			if !isCompletedNow(c) {
				continue
			}
			views = append(views, toView(c))
		}
		// Most recently completed first.
		sort.Slice(views, func(i, j int) bool {
			return views[i].LastCompletedAt.After(views[j].LastCompletedAt)
		})

		writeJSON(w, http.StatusOK, views)
	}
}

func handleCreateChore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req choreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" || req.IntervalHours <= 0 {
			writeError(w, http.StatusBadRequest, "name and a positive interval_hours are required")
			return
		}

		c, err := createChore(db, req.Name, req.Description, req.IntervalHours, req.Recurring)
		if err != nil {
			log.Println("createChore:", err)
			writeError(w, http.StatusInternalServerError, "failed to create chore")
			return
		}
		writeJSON(w, http.StatusCreated, toView(c))
	}
}

func handleUpdateChore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid chore id")
			return
		}

		var req choreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" || req.IntervalHours <= 0 {
			writeError(w, http.StatusBadRequest, "name and a positive interval_hours are required")
			return
		}

		if err := updateChore(db, id, req.Name, req.Description, req.IntervalHours, req.Recurring); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "chore not found")
				return
			}
			log.Println("updateChore:", err)
			writeError(w, http.StatusInternalServerError, "failed to update chore")
			return
		}

		c, err := getChore(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "chore not found")
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load chore")
			return
		}
		writeJSON(w, http.StatusOK, toView(c))
	}
}

// handleCompleteChore resets a chore's countdown to now. From there,
// isCompletedNow decides how long it stays on the Completed tab: a
// recurring chore until local midnight, a non-recurring one for 24 hours
// before pruneExpiredChores deletes it for good.
func handleCompleteChore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid chore id")
			return
		}

		if err := completeChore(db, id); err != nil {
			log.Println("completeChore:", err)
			writeError(w, http.StatusInternalServerError, "failed to complete chore")
			return
		}

		c, err := getChore(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "chore not found")
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load chore")
			return
		}
		writeJSON(w, http.StatusOK, toView(c))
	}
}

// handleUndoComplete reverses the most recent handleCompleteChore call,
// e.g. for an accidental tap on Done from the Completed tab.
func handleUndoComplete(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid chore id")
			return
		}

		if err := undoComplete(db, id); err != nil {
			if errors.Is(err, errNothingToUndo) {
				writeError(w, http.StatusBadRequest, "nothing to undo")
				return
			}
			log.Println("undoComplete:", err)
			writeError(w, http.StatusInternalServerError, "failed to undo")
			return
		}

		c, err := getChore(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "chore not found")
			return
		} else if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load chore")
			return
		}
		writeJSON(w, http.StatusOK, toView(c))
	}
}

func handleDeleteChore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid chore id")
			return
		}
		if err := deleteChore(db, id); err != nil {
			log.Println("deleteChore:", err)
			writeError(w, http.StatusInternalServerError, "failed to delete chore")
			return
		}
		writeJSON(w, http.StatusNoContent, nil)
	}
}

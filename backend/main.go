package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dbPath := os.Getenv("CHORE_TIMER_DB_PATH")
	if dbPath == "" {
		dbPath = "chores.db"
	}
	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// An env var always wins on startup, so it doubles as a way to force a
	// known password (e.g. from a launch script) regardless of whatever was
	// last set through the app.
	if pw := os.Getenv("CHORE_TIMER_PASSWORD"); pw != "" {
		if err := setSetting(db, "password", pw); err != nil {
			log.Fatalf("failed to apply CHORE_TIMER_PASSWORD: %v", err)
		}
	}

	sessions := newSessionStore()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/chores", handleListChores(db))
	mux.HandleFunc("GET /api/chores/completed", handleListCompletedChores(db))
	mux.HandleFunc("POST /api/chores", handleCreateChore(db))
	mux.HandleFunc("PUT /api/chores/{id}", handleUpdateChore(db))
	mux.HandleFunc("POST /api/chores/{id}/complete", handleCompleteChore(db))
	mux.HandleFunc("POST /api/chores/{id}/undo", handleUndoComplete(db))
	mux.HandleFunc("DELETE /api/chores/{id}", handleDeleteChore(db))

	mux.HandleFunc("GET /api/auth/status", handleAuthStatus(db, sessions))
	mux.HandleFunc("POST /api/auth/login", handleLogin(db, sessions))
	mux.HandleFunc("POST /api/auth/logout", handleLogout(sessions))

	// Only populated by the Docker build (see static/.gitkeep); in local dev
	// this serves nothing because Vite handles the frontend directly and
	// nothing ever routes here.
	mux.Handle("/", http.FileServer(http.FS(staticFS())))

	addr := ":8080"
	log.Printf("chore-timer backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, requireAuth(db, sessions, mux)))
}

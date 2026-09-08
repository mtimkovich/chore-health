package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dbPath := os.Getenv("CHORE_HEALTH_DB_PATH")
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
	if pw := os.Getenv("CHORE_HEALTH_PASSWORD"); pw != "" {
		if err := setSetting(db, "password", pw); err != nil {
			log.Fatalf("failed to apply CHORE_HEALTH_PASSWORD: %v", err)
		}
	}

	a := &app{db: db, sessions: newSessionStore()}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/chores", a.handleListChores)
	mux.HandleFunc("GET /api/chores/completed", a.handleListCompletedChores)
	mux.HandleFunc("POST /api/chores", a.handleCreateChore)
	mux.HandleFunc("PUT /api/chores/{id}", a.handleUpdateChore)
	mux.HandleFunc("POST /api/chores/{id}/complete", a.handleCompleteChore)
	mux.HandleFunc("POST /api/chores/{id}/undo", a.handleUndoComplete)
	mux.HandleFunc("DELETE /api/chores/{id}", a.handleDeleteChore)

	mux.HandleFunc("GET /api/auth/status", a.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", a.handleLogout)

	// Only populated by the Docker build (see static/.gitkeep); in local dev
	// this serves nothing because Vite handles the frontend directly and
	// nothing ever routes here.
	mux.Handle("/", http.FileServer(http.FS(staticFS())))

	addr := ":8080"
	log.Printf("chore-health backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, a.requireAuth(mux)))
}

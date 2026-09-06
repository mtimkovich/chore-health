package main

import (
	"log"
	"net/http"
)

func main() {
	db, err := openDB("chores.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/chores", handleListChores(db))
	mux.HandleFunc("GET /api/chores/completed", handleListCompletedChores(db))
	mux.HandleFunc("POST /api/chores", handleCreateChore(db))
	mux.HandleFunc("PUT /api/chores/{id}", handleUpdateChore(db))
	mux.HandleFunc("POST /api/chores/{id}/complete", handleCompleteChore(db))
	mux.HandleFunc("DELETE /api/chores/{id}", handleDeleteChore(db))

	addr := ":8080"
	log.Printf("chore-timer backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

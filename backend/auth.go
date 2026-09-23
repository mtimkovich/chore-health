package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

const sessionCookieName = "chore_health_session"

// sessionMaxAge matches the session cookie's own MaxAge - a session past
// this age is treated as expired even if its row is still in the database.
const sessionMaxAge = 90 * 24 * time.Hour

// sessionStore checks logins against the sessions table instead of an
// in-memory map, so they survive a server restart instead of forcing
// everyone to log back in.
type sessionStore struct {
	db *sql.DB
}

// newSessionStore also clears out any sessions that have already aged past
// sessionMaxAge, so the table doesn't just grow forever.
func newSessionStore(db *sql.DB) (*sessionStore, error) {
	if _, err := db.Exec(`DELETE FROM sessions WHERE created_at < ?`, time.Now().UTC().Add(-sessionMaxAge)); err != nil {
		return nil, err
	}
	return &sessionStore{db: db}, nil
}

func (s *sessionStore) create() (string, error) {
	token := randomToken()
	_, err := s.db.Exec(`INSERT INTO sessions (token, created_at) VALUES (?, ?)`, token, time.Now().UTC())
	return token, err
}

func (s *sessionStore) valid(token string) bool {
	var createdAt time.Time
	if err := s.db.QueryRow(`SELECT created_at FROM sessions WHERE token = ?`, token).Scan(&createdAt); err != nil {
		return false
	}
	return time.Since(createdAt) < sessionMaxAge
}

func (s *sessionStore) revoke(token string) {
	s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionMaxAge.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (a *app) requestAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return a.sessions.valid(cookie.Value)
}

// requireAuth gates only /api/chores* behind the configured password - not
// /api/auth/* (that's the login flow itself) and not the static frontend
// (its JS is what renders the login screen in the first place, so it can't
// be behind the same gate it's presenting). If no password is configured,
// this is a no-op and the app behaves exactly as it did before auth
// existed.
func (a *app) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/chores") {
			next.ServeHTTP(w, r)
			return
		}

		if a.password == "" || a.requestAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}

		writeError(w, http.StatusUnauthorized, "login required")
	})
}

func (a *app) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"password_set":  a.password != "",
		"authenticated": a.password == "" || a.requestAuthenticated(r),
	})
}

type loginRequest struct {
	Password string `json:"password"`
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if a.password == "" || req.Password != a.password {
		writeError(w, http.StatusUnauthorized, "incorrect password")
		return
	}

	token, err := a.sessions.create()
	if err != nil {
		log.Println("sessions.create:", err)
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *app) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		a.sessions.revoke(cookie.Value)
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

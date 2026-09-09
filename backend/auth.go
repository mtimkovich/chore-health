package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "chore_health_session"

// sessionStore tracks logged-in sessions in memory. That's plenty for a
// single-user app on a local network: no persistence needed, and a server
// restart just means logging in again.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]time.Time)}
}

func (s *sessionStore) create() string {
	token := randomToken()
	s.mu.Lock()
	s.sessions[token] = time.Now()
	s.mu.Unlock()
	return token
}

func (s *sessionStore) valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[token]
	return ok
}

func (s *sessionStore) revoke(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
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
		MaxAge:   30 * 24 * 60 * 60, // 30 days - this is a low-stakes local-network login
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

	setSessionCookie(w, a.sessions.create())
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *app) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		a.sessions.revoke(cookie.Value)
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

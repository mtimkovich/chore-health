package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "chore_timer_session"

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

func requestAuthenticated(r *http.Request, sessions *sessionStore) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return sessions.valid(cookie.Value)
}

// requireAuth gates every route except /api/auth/* behind the stored
// password. If no password has ever been set, this is a no-op and the app
// behaves exactly as it did before auth existed.
func requireAuth(db *sql.DB, sessions *sessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		password, err := getSetting(db, "password")
		if err != nil {
			log.Println("getSetting password:", err)
			writeError(w, http.StatusInternalServerError, "failed to check auth")
			return
		}
		if password == "" || requestAuthenticated(r, sessions) {
			next.ServeHTTP(w, r)
			return
		}

		writeError(w, http.StatusUnauthorized, "login required")
	})
}

func handleAuthStatus(db *sql.DB, sessions *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		password, err := getSetting(db, "password")
		if err != nil {
			log.Println("getSetting password:", err)
			writeError(w, http.StatusInternalServerError, "failed to check auth")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{
			"password_set":  password != "",
			"authenticated": password == "" || requestAuthenticated(r, sessions),
		})
	}
}

type loginRequest struct {
	Password string `json:"password"`
}

func handleLogin(db *sql.DB, sessions *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		password, err := getSetting(db, "password")
		if err != nil {
			log.Println("getSetting password:", err)
			writeError(w, http.StatusInternalServerError, "failed to check auth")
			return
		}
		if password == "" || req.Password != password {
			writeError(w, http.StatusUnauthorized, "incorrect password")
			return
		}

		setSessionCookie(w, sessions.create())
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

func handleLogout(sessions *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			sessions.revoke(cookie.Value)
		}
		clearSessionCookie(w)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

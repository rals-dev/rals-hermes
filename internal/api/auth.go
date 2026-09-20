package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rals-dev/rals-hermes/internal/config"
)

// sessionCookieName is the browser session cookie (ADR-010).
const sessionCookieName = "bff_session"

// Login rate limiting: after loginBurst failures from one address within
// loginWindow, every attempt from that address is refused until the window
// passes — including attempts with the correct key, so the limiter is not an
// oracle.
const (
	loginBurst  = 5
	loginWindow = time.Minute
)

// sessions is the in-memory session store. A BFF restart forgets every
// session; for one operator that is a login, not an incident.
type sessions struct {
	key config.Secret
	ttl time.Duration
	now func() time.Time

	mu       sync.Mutex
	tokens   map[string]time.Time // token → expiry
	failures map[string][]time.Time
}

func newSessions(key config.Secret, ttl time.Duration, now func() time.Time) *sessions {
	return &sessions{key: key, ttl: ttl, now: now, tokens: map[string]time.Time{}, failures: map[string][]time.Time{}}
}

// login validates the operator key and mints a session token.
func (s *sessions) login(w http.ResponseWriter, r *http.Request) {
	addr := clientAddr(r)
	if s.limited(addr) {
		writeError(w, http.StatusTooManyRequests, "too_many_attempts", "too many failed attempts; try again later")
		return
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "body must be JSON: {\"key\": \"...\"}")
		return
	}
	if subtle.ConstantTimeCompare([]byte(body.Key), []byte(s.key.Reveal())) != 1 {
		s.recordFailure(addr)
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid key")
		return
	}
	s.clearFailures(addr)

	token := newToken()
	s.mu.Lock()
	s.tokens[token] = s.now().Add(s.ttl)
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure is deliberately off: plain HTTP inside WireGuard (ADR-010)
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(s.ttl / time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		// Secure is deliberately off: Traefik serves plain HTTP inside the
		// WireGuard-encrypted tailnet (ADR-010).
	})
	w.WriteHeader(http.StatusNoContent)
}

// logout revokes the presented session and clears the cookie.
func (s *sessions) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		s.mu.Lock()
		delete(s.tokens, c.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode}) //nolint:gosec // see login
	w.WriteHeader(http.StatusNoContent)
}

// valid reports whether the request carries a live session cookie.
func (s *sessions) valid(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.tokens[c.Value]
	if !ok {
		return false
	}
	if !s.now().Before(exp) {
		delete(s.tokens, c.Value)
		return false
	}
	return true
}

// require wraps next so it only runs with a valid session.
func (s *sessions) require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.valid(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "a valid session is required")
			return
		}
		next(w, r)
	}
}

func (s *sessions) limited(addr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.prune(addr)) >= loginBurst
}

func (s *sessions) recordFailure(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures[addr] = append(s.prune(addr), s.now())
}

func (s *sessions) clearFailures(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.failures, addr)
}

// prune drops failures older than the window; caller holds mu.
func (s *sessions) prune(addr string) []time.Time {
	cutoff := s.now().Add(-loginWindow)
	kept := s.failures[addr][:0]
	for _, t := range s.failures[addr] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(s.failures, addr)
		return nil
	}
	s.failures[addr] = kept
	return kept
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// clientAddr keys the limiter. Behind Traefik the real client is in
// X-Forwarded-For; the BFF is never reachable without Traefik, so trusting
// the first hop is acceptable here.
func clientAddr(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

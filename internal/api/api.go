// Package api exposes the BFF's HTTP surface: the read-only JSON API under
// /api, the unauthenticated liveness and metrics endpoints, and (later) the
// embedded web UI.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/rals-dev/rals-hermes/internal/cache"
	"github.com/rals-dev/rals-hermes/internal/config"
	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// Deps carries everything the handlers need.
type Deps struct {
	Logger  *slog.Logger
	Version string
	// Profiles are the configured Hermes clients, in configuration order.
	// That order is the display order everywhere.
	Profiles []*hermes.Client
	// CacheTTL bounds how long upstream responses are reused.
	CacheTTL time.Duration
	// AuthKey is the operator secret accepted by POST /api/auth/session.
	AuthKey config.Secret
	// SessionTTL is the lifetime of a login cookie.
	SessionTTL time.Duration

	// now overrides the clock in tests.
	now func() time.Time
}

// handlers holds the per-process state behind the routes.
type handlers struct {
	log      *slog.Logger
	version  string
	profiles []*hermes.Client
	byName   map[string]*hermes.Client
	health   *cache.Cache[*hermes.HealthDetailed]
	sessions *sessions
}

// NewHandler builds the root handler with all routes registered.
func NewHandler(d Deps) http.Handler {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	if d.CacheTTL <= 0 {
		d.CacheTTL = 3 * time.Second
	}
	if d.SessionTTL <= 0 {
		d.SessionTTL = 24 * time.Hour
	}
	if d.now == nil {
		d.now = time.Now
	}
	h := &handlers{
		log:      d.Logger,
		version:  d.Version,
		profiles: d.Profiles,
		byName:   make(map[string]*hermes.Client, len(d.Profiles)),
		health:   cache.New[*hermes.HealthDetailed](d.CacheTTL),
		sessions: newSessions(d.AuthKey, d.SessionTTL, d.now),
	}
	for _, c := range d.Profiles {
		h.byName[c.Name()] = c
	}

	r := newRouter()
	r.handle("GET /healthz", healthz(d.Version))
	r.handle("POST /api/auth/session", h.sessions.login)
	r.handle("DELETE /api/auth/session", h.sessions.logout)

	// Everything else under /api needs a session (T-106). /healthz and
	// /metrics stay open by design.
	auth := h.sessions.require
	r.handle("GET /api/overview", auth(h.overview))
	r.handle("GET /api/agents", auth(h.agents))
	r.handle("GET /api/agents/{profile}", auth(h.agentDetail))
	return requestLog(d.Logger, r)
}

func healthz(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	}
}

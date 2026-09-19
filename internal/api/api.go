// Package api exposes the BFF's HTTP surface: the read-only JSON API under
// /api, the unauthenticated liveness and metrics endpoints, and (later) the
// embedded web UI.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/rals-dev/rals-hermes/internal/activity"
	"github.com/rals-dev/rals-hermes/internal/cache"
	"github.com/rals-dev/rals-hermes/internal/config"
	"github.com/rals-dev/rals-hermes/internal/hermes"
	"github.com/rals-dev/rals-hermes/internal/observ"
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

	// Metrics is optional; when nil, /metrics is not served.
	Metrics *observ.Metrics
	// Activity bounds the session poller behind /activity/stream.
	Activity activity.Config
	// StreamKeepalive is how often BFF-generated streams send a comment
	// line while idle. Zero means 15 s.
	StreamKeepalive time.Duration

	// now overrides the clock in tests.
	now func() time.Time
}

// handlers holds the per-process state behind the routes.
type handlers struct {
	log       *slog.Logger
	version   string
	profiles  []*hermes.Client
	byName    map[string]*hermes.Client
	health    *cache.Cache[*hermes.HealthDetailed]
	sessions  *sessions
	metrics   *observ.Metrics
	activity  *activity.Hub
	keepalive time.Duration
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
	if d.StreamKeepalive <= 0 {
		d.StreamKeepalive = 15 * time.Second
	}
	sources := make([]activity.Source, 0, len(d.Profiles))
	for _, c := range d.Profiles {
		sources = append(sources, c)
	}
	h := &handlers{
		log:       d.Logger,
		version:   d.Version,
		profiles:  d.Profiles,
		byName:    make(map[string]*hermes.Client, len(d.Profiles)),
		health:    cache.New[*hermes.HealthDetailed](d.CacheTTL),
		sessions:  newSessions(d.AuthKey, d.SessionTTL, d.now),
		metrics:   d.Metrics,
		activity:  activity.NewHub(d.Activity, sources, d.Logger),
		keepalive: d.StreamKeepalive,
	}
	if h.metrics != nil {
		h.metrics.RegisterCache("health", h.health.Stats)
		h.activity.SetGauge(h.metrics)
	}
	for _, c := range d.Profiles {
		h.byName[c.Name()] = c
	}

	r := newRouter()
	r.handle("GET /healthz", healthz(d.Version))
	if h.metrics != nil {
		// Unauthenticated by design: reachable only through the Traefik LAN
		// entrypoint with an IP allow-list (ADR-008).
		r.handle("GET /metrics", h.metrics.Handler().ServeHTTP)
	}
	r.handle("POST /api/auth/session", h.sessions.login)
	r.handle("DELETE /api/auth/session", h.sessions.logout)

	// Everything else under /api needs a session (T-106). /healthz and
	// /metrics stay open by design.
	auth := h.sessions.require
	r.handle("GET /api/overview", auth(h.overview))
	r.handle("GET /api/agents", auth(h.agents))
	r.handle("GET /api/agents/{profile}", auth(h.agentDetail))
	r.handle("GET /api/agents/{profile}/sessions", auth(h.listSessions))
	r.handle("GET /api/agents/{profile}/sessions/{id}", auth(h.sessionDetail))
	r.handle("GET /api/agents/{profile}/runs/{run_id}", auth(h.run))
	r.handle("GET /api/agents/{profile}/runs/{run_id}/stream", auth(h.runStream))
	r.handle("GET /api/agents/{profile}/activity/stream", auth(h.activityStream))
	return requestLog(d.Logger, h.metrics, r)
}

func healthz(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	}
}

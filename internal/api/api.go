// Package api exposes the BFF's HTTP surface: the read-only JSON API under
// /api, the unauthenticated liveness and metrics endpoints, and (later) the
// embedded web UI.
package api

import (
	"log/slog"
	"net/http"
)

// Deps carries everything the handlers need. Fields are added as milestones
// land; the zero value is only useful in tests.
type Deps struct {
	Logger  *slog.Logger
	Version string
}

// NewHandler builds the root handler with all routes registered.
func NewHandler(d Deps) http.Handler {
	if d.Logger == nil {
		d.Logger = slog.Default()
	}
	r := newRouter()
	r.handle("GET /healthz", healthz(d.Version))
	return r
}

func healthz(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	}
}

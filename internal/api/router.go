package api

import (
	"net/http"
	"strings"
)

// router wraps http.ServeMux so that every miss is answered in the BFF's JSON
// error shape: 404 for unknown paths and 405 for known paths with the wrong
// method. Go's mux would otherwise fall through to the "/" catch-all and
// report 404 for a wrong method.
type router struct {
	*http.ServeMux
	paths map[string]struct{}
}

// newRouter builds the mux with fallback as the "/" catch-all. Every
// registered pattern is more specific than "/", so the fallback only sees
// paths nothing else claimed — the JSON 404 by default, the SPA when a UI
// bundle is present.
func newRouter(fallback http.Handler) *router {
	r := &router{ServeMux: http.NewServeMux(), paths: map[string]struct{}{}}
	if fallback == nil {
		fallback = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusNotFound, "not_found", "no such route")
		})
	}
	r.Handle("/", fallback)
	return r
}

// handle registers pattern ("METHOD /path") and, once per path, a
// method-agnostic fallback that returns 405. The wrapper records the matched
// pattern and {profile} value for the access log and metrics.
func (r *router) handle(pattern string, h http.HandlerFunc) {
	r.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
		info := infoFrom(req.Context())
		info.route = pattern
		info.profile = req.PathValue("profile")
		h(w, req)
	})
	_, path, ok := strings.Cut(pattern, " ")
	if !ok {
		return
	}
	if _, seen := r.paths[path]; seen {
		return
	}
	r.paths[path] = struct{}{}
	r.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})
}

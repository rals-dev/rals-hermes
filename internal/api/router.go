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

func newRouter() *router {
	r := &router{ServeMux: http.NewServeMux(), paths: map[string]struct{}{}}
	r.ServeMux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "no such route")
	})
	return r
}

// handle registers pattern ("METHOD /path") and, once per path, a
// method-agnostic fallback that returns 405.
func (r *router) handle(pattern string, h http.HandlerFunc) {
	r.ServeMux.HandleFunc(pattern, h)
	_, path, ok := strings.Cut(pattern, " ")
	if !ok {
		return
	}
	if _, seen := r.paths[path]; seen {
		return
	}
	r.paths[path] = struct{}{}
	r.ServeMux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})
}

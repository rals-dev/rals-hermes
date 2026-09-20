package api

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// uiHandler serves the embedded single-page app (ADR-011).
//
// Real files are served as-is; hashed build assets under /assets are
// immutable-cacheable; every other path falls back to index.html so the Vue
// router owns the URL space. /api, /healthz and /metrics are registered
// with higher precedence on the mux and never reach this handler.
type uiHandler struct {
	fsys fs.FS
}

func (u uiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	// Unknown /api paths must stay JSON even when a UI bundle is present.
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "not_found", "no such route")
		return
	}
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "index.html"
	}
	if f, err := u.fsys.Open(name); err == nil {
		defer f.Close()
		if st, err := f.Stat(); err == nil && !st.IsDir() {
			u.serveFile(w, name, f)
			return
		}
	}
	// Asset misses are genuine 404s; anything else is a client-side route.
	if strings.HasPrefix(name, "assets/") || strings.Contains(path.Base(name), ".") && name != "index.html" {
		writeError(w, http.StatusNotFound, "not_found", "no such file")
		return
	}
	f, err := u.fsys.Open("index.html")
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "the web UI is not bundled into this build")
		return
	}
	defer f.Close()
	u.serveFile(w, "index.html", f)
}

func (uiHandler) serveFile(w http.ResponseWriter, name string, f fs.File) {
	ct := mime.TypeByExtension(path.Ext(name))
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

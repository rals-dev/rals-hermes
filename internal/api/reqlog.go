package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// reqInfo is filled in by the matched route so the access log can report the
// route pattern and profile without parsing the path again.
type reqInfo struct {
	route   string
	profile string
}

type reqInfoKey struct{}

func infoFrom(ctx context.Context) *reqInfo {
	if v, ok := ctx.Value(reqInfoKey{}).(*reqInfo); ok {
		return v
	}
	return &reqInfo{}
}

// statusWriter captures status and size for the access log.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Flush lets SSE handlers stream through the wrapper.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		if w.status == 0 {
			w.status = http.StatusOK
		}
		f.Flush()
	}
}

// requestLog emits one JSON line per request (T-109). Headers, cookies and
// query strings are never logged: the query may carry nothing sensitive
// today, but the rule is simpler to keep than to audit.
func requestLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		info := &reqInfo{}
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r.WithContext(context.WithValue(r.Context(), reqInfoKey{}, info)))
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"route", info.route,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", sw.bytes,
			"remote", clientAddr(r),
		}
		if info.profile != "" {
			attrs = append(attrs, "profile", info.profile)
		}
		log.Info("request", attrs...)
	})
}

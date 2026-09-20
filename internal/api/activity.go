package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rals-dev/rals-hermes/internal/activity"
)

// activityStream serves the BFF-generated feed for one profile as SSE
// (T-205). Each event is "event: <type>\ndata: <json>\n\n"; a ": keepalive"
// comment is sent while idle so proxies and browsers keep the connection.
func (h *handlers) activityStream(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.profileFromPath(w, r); !ok {
		return
	}
	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		writeError(w, http.StatusInternalServerError, "internal_error", "streaming unsupported")
		return
	}
	events, cancel, err := h.activity.Subscribe(r.PathValue("profile"))
	if err != nil {
		if errors.Is(err, activity.ErrUnknownProfile) {
			writeMappedError(w, ErrProfileNotFound, "not_found")
			return
		}
		writeMappedError(w, err, "not_found")
		return
	}
	defer cancel()

	hd := w.Header()
	hd.Set("Content-Type", "text/event-stream; charset=utf-8")
	hd.Set("Cache-Control", "no-cache")
	hd.Set("Connection", "keep-alive")
	hd.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	// A first comment line makes proxies release the headers and lets the
	// browser's EventSource fire onopen before any real event exists.
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	keepalive := time.NewTicker(h.keepalive)
	defer keepalive.Stop()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-keepalive.C:
			if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case e := <-events:
			payload, err := json.Marshal(e)
			if err != nil {
				continue
			}
			if _, err := w.Write([]byte("event: " + e.Type + "\ndata: " + string(payload) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

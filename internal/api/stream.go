package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// sseLineLimit caps one upstream line. Hermes previews are cut at 500
// characters, so real lines are small; the cap only guards the relay.
const sseLineLimit = 1 << 20

// runStream relays GET /v1/runs/{id}/events to the browser (T-204).
//
//   - Lines are forwarded verbatim and flushed at every blank line (end of
//     event) and for every comment line, so ": keepalive" reaches the browser
//     without waiting for a real event.
//   - The upstream body is tied to the request context: when the browser
//     leaves, the context is cancelled and the upstream connection closes.
//   - An upstream read error mid-stream is reported as a named "error" event
//     with the uniform error body, then the response ends cleanly.
//   - Nothing is buffered or replayed; reconnect is the frontend's job.
func (h *handlers) runStream(w http.ResponseWriter, r *http.Request) {
	c, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	flusher, canFlush := w.(http.Flusher)
	if !canFlush {
		writeError(w, http.StatusInternalServerError, "internal_error", "streaming unsupported")
		return
	}
	stream, err := c.StreamRunEvents(r.Context(), r.PathValue("run_id"))
	if err != nil {
		writeMappedError(w, err, "run_not_found")
		return
	}
	defer stream.Close()

	hd := w.Header()
	hd.Set("Content-Type", "text/event-stream; charset=utf-8")
	hd.Set("Cache-Control", "no-cache")
	hd.Set("Connection", "keep-alive")
	hd.Set("X-Accel-Buffering", "no") // never let a proxy buffer the stream
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	relaySSE(w, flusher, stream.Body)
}

// relaySSE copies an event stream line by line. It is separated from the
// handler so the flush policy can be tested against an io.Reader.
func relaySSE(w io.Writer, flusher http.Flusher, upstream io.Reader) {
	sc := bufio.NewScanner(upstream)
	sc.Buffer(make([]byte, 0, 64*1024), sseLineLimit)
	for sc.Scan() {
		line := sc.Bytes()
		if _, err := w.Write(append(line, '\n')); err != nil {
			return // browser gone; the deferred Close cancels upstream
		}
		if len(line) == 0 || line[0] == ':' {
			flusher.Flush()
		}
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		// Upstream broke mid-stream. Tell the browser in-band, then end.
		_, body := mapError(&hermes.UpstreamError{Kind: hermes.KindUnreachable}, "run_not_found")
		payload, _ := json.Marshal(body)
		_, _ = w.Write([]byte("event: error\ndata: " + string(payload) + "\n\n"))
	}
	flusher.Flush()
}

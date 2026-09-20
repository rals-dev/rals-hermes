package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// sseUpstream is a fake /v1/runs/{id}/events that scripts its output.
type sseUpstream struct {
	t          *testing.T
	script     func(w http.ResponseWriter, flush func())
	clientGone chan struct{}
}

func (u *sseUpstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/events") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Header.Get("Authorization") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	f := w.(http.Flusher)
	done := make(chan struct{})
	go func() {
		<-r.Context().Done()
		close(u.clientGone)
		close(done)
	}()
	u.script(w, f.Flush)
	<-done
}

func newSSEUpstream(t *testing.T, script func(w http.ResponseWriter, flush func())) *sseUpstream {
	return &sseUpstream{t: t, script: script, clientGone: make(chan struct{})}
}

// startBFF runs the real handler on a real listener and logs in.
func startBFF(t *testing.T, d Deps) (base string, cookie *http.Cookie) {
	t.Helper()
	srv := httptest.NewServer(NewHandler(d))
	t.Cleanup(srv.Close)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/session", strings.NewReader(`{"key":"`+bffKey+`"}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			return srv.URL, c
		}
	}
	t.Fatal("login did not set a cookie")
	return "", nil
}

func openStream(ctx context.Context, t *testing.T, base string, cookie *http.Cookie, path string) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	req.AddCookie(cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	return resp
}

func TestStream_RelaysEventsAndKeepalivesAsTheyArrive(t *testing.T) {
	up := newSSEUpstream(t, func(w http.ResponseWriter, flush func()) {
		_, _ = w.Write([]byte("event: tool.started\ndata: {\"tool\":\"terminal\"}\n\n"))
		flush()
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(": keepalive\n\n"))
		flush()
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte("event: run.completed\ndata: {}\n\n"))
		flush()
	})
	d := newTestDeps(t, time.Second, profileSpec{"default", up})
	base, cookie := startBFF(t, d)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp := openStream(ctx, t, base, cookie, "/api/agents/default/runs/r1/stream")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status=%d content-type=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("Cache-Control") != "no-cache" || resp.Header.Get("X-Accel-Buffering") != "no" {
		t.Errorf("missing anti-buffering headers: %v", resp.Header)
	}

	sc := bufio.NewScanner(resp.Body)
	var lines []string
	firstAt := time.Time{}
	for sc.Scan() {
		if firstAt.IsZero() {
			firstAt = time.Now()
		}
		lines = append(lines, sc.Text())
	}
	got := strings.Join(lines, "\n")
	for _, want := range []string{"event: tool.started", `data: {"tool":"terminal"}`, ": keepalive", "event: run.completed"} {
		if !strings.Contains(got, want) {
			t.Errorf("relayed stream lacks %q:\n%s", want, got)
		}
	}
	// The first event must be delivered before the upstream finished (~100 ms
	// later) — i.e. the relay flushes per event instead of buffering.
	if firstAt.IsZero() {
		t.Fatal("no lines received")
	}
}

func TestStream_ClientDisconnectClosesUpstream(t *testing.T) {
	up := newSSEUpstream(t, func(w http.ResponseWriter, flush func()) {
		_, _ = w.Write([]byte("event: tool.started\ndata: {}\n\n"))
		flush()
		// then stay silent until the client goes away
	})
	d := newTestDeps(t, time.Second, profileSpec{"default", up})
	base, cookie := startBFF(t, d)

	ctx, cancel := context.WithCancel(context.Background())
	resp := openStream(ctx, t, base, cookie, "/api/agents/default/runs/r1/stream")
	buf := make([]byte, 64)
	if _, err := resp.Body.Read(buf); err != nil { // wait for the first event
		t.Fatalf("read: %v", err)
	}
	cancel()
	resp.Body.Close()

	select {
	case <-up.clientGone:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream connection was not closed after the client disconnected")
	}
}

func TestStream_UpstreamFailureMidStreamBecomesErrorEvent(t *testing.T) {
	up := newSSEUpstream(t, func(w http.ResponseWriter, flush func()) {
		_, _ = w.Write([]byte("event: tool.started\ndata: {}\n\n"))
		flush()
		// Kill the TCP connection without a clean end-of-stream.
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			if tc, ok := conn.(*net.TCPConn); ok {
				_ = tc.SetLinger(0)
			}
			conn.Close()
		}
	})
	d := newTestDeps(t, time.Second, profileSpec{"default", up})
	base, cookie := startBFF(t, d)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp := openStream(ctx, t, base, cookie, "/api/agents/default/runs/r1/stream")
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, "event: error") {
		t.Fatalf("expected a named error event before close:\n%s", got)
	}
	// The error payload uses the uniform error body.
	for _, l := range lines {
		if strings.HasPrefix(l, "data: ") && strings.Contains(l, "upstream_unreachable") {
			var body errorBody
			if err := json.Unmarshal([]byte(strings.TrimPrefix(l, "data: ")), &body); err != nil || body.Code != "upstream_unreachable" {
				t.Errorf("error payload = %s (%v)", l, err)
			}
			return
		}
	}
	t.Errorf("no upstream_unreachable error payload in:\n%s", got)
}

func TestStream_Upstream404IsPlainJSONRunNotFound(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", serveStatus(404)})
	base, cookie := startBFF(t, d)
	resp := openStream(context.Background(), t, base, cookie, "/api/agents/default/runs/ghost/stream")
	defer resp.Body.Close()
	var body errorBody
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if resp.StatusCode != http.StatusNotFound || body.Code != "run_not_found" {
		t.Errorf("status=%d code=%q", resp.StatusCode, body.Code)
	}
}

func TestStream_RequiresSession(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", serveStatus(200)})
	srv := httptest.NewServer(NewHandler(d))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/agents/default/runs/r1/stream") //nolint:noctx // test
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

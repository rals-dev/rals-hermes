package hermes

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rals-dev/rals-hermes/internal/config"
)

const testKey = "k-secret-value"

func fixture(t *testing.T, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", rel))
	if err != nil {
		t.Fatalf("fixture %s: %v", rel, err)
	}
	return b
}

func newClient(t *testing.T, baseURL string, timeout time.Duration) *Client {
	t.Helper()
	return NewClient(config.Profile{Name: "coder-agent", BaseURL: baseURL, Key: config.Secret(testKey)},
		Options{Timeout: timeout, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
}

func TestClient_SendsBearerAndJoinsPrefixedBaseURL(t *testing.T) {
	var gotAuth, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath, gotQuery = r.Header.Get("Authorization"), r.URL.Path, r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "coder-agent/sessions.json"))
	}))
	defer srv.Close()

	c := newClient(t, srv.URL+"/p/coder-agent", time.Second)
	_, err := c.Sessions(context.Background(), SessionsQuery{Limit: 5, Offset: 10, Source: "kanban", IncludeChildren: true})
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}
	if gotAuth != "Bearer "+testKey {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotPath != "/p/coder-agent/api/sessions" {
		t.Errorf("path = %q, want /p/coder-agent/api/sessions", gotPath)
	}
	for _, want := range []string{"limit=5", "offset=10", "source=kanban", "include_children=true"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q lacks %s", gotQuery, want)
		}
	}
}

func TestClient_TimeoutIsEnforcedWithoutRetry(t *testing.T) {
	var hits atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	c := newClient(t, srv.URL, 100*time.Millisecond)
	start := time.Now()
	_, err := c.HealthDetailed(context.Background())
	elapsed := time.Since(start)

	var ue *UpstreamError
	if !errors.As(err, &ue) || ue.Kind != KindUnreachable {
		t.Fatalf("err = %v, want *UpstreamError{Kind: KindUnreachable}", err)
	}
	if elapsed > 400*time.Millisecond {
		t.Errorf("took %v; timeout of 100ms not enforced", elapsed)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("upstream hit %d times, want exactly 1 (no retries)", n)
	}
}

func TestClient_ConnectionRefusedIsUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens on url any more

	_, err := newClient(t, url, time.Second).HealthDetailed(context.Background())
	var ue *UpstreamError
	if !errors.As(err, &ue) || ue.Kind != KindUnreachable {
		t.Fatalf("err = %v, want KindUnreachable", err)
	}
}

func TestClient_MapsHTTPStatusesToKinds(t *testing.T) {
	cases := []struct {
		status int
		kind   Kind
	}{
		{401, KindUnauthorized},
		{403, KindUnauthorized},
		{404, KindNotFound},
		{429, KindBusy},
		{500, KindServerError},
		{503, KindServerError},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":{"message":"upstream says: ` + testKey + `"}}`))
			}))
			defer srv.Close()

			_, err := newClient(t, srv.URL, time.Second).Run(context.Background(), "run-1")
			var ue *UpstreamError
			if !errors.As(err, &ue) {
				t.Fatalf("err = %T %v, want *UpstreamError", err, err)
			}
			if ue.Kind != tc.kind || ue.Status != tc.status || ue.Profile != "coder-agent" {
				t.Errorf("got %+v, want kind=%v status=%d profile=coder-agent", ue, tc.kind, tc.status)
			}
			if strings.Contains(err.Error(), testKey) {
				t.Errorf("error text leaks upstream body containing the key: %q", err.Error())
			}
		})
	}
}

func TestClient_ErrorsNeverContainTheKey(t *testing.T) {
	// Even a malformed base URL or a transport failure must not echo the key.
	c := newClient(t, "http://127.0.0.1:1", 200*time.Millisecond)
	_, err := c.HealthDetailed(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), testKey) {
		t.Errorf("error leaks key: %q", err.Error())
	}
}

func TestClient_DecodesHealthDetailedFixture(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "default/health_detailed.json"))
	}))
	defer srv.Close()

	h, err := newClient(t, srv.URL, time.Second).HealthDetailed(context.Background())
	if err != nil {
		t.Fatalf("HealthDetailed: %v", err)
	}
	if h.Status != "ok" || h.Version != "0.21.2" || h.GatewayState != "running" {
		t.Errorf("top-level = %+v", *h)
	}
	if h.Readiness.Checks.Disk.UsedPercent != 10.9 || h.Readiness.Checks.Gateway.ConnectedPlatforms != 4 {
		t.Errorf("checks = %+v", h.Readiness.Checks)
	}
	if q := h.Readiness.Checks.BackgroundQueues; q.ActiveAPIRuns != 0 || q.ActiveDelegations != 0 {
		t.Errorf("queues = %+v", q)
	}
	if p, ok := h.Platforms["coder-agent:telegram"]; !ok || p.State != "connected" {
		t.Errorf("platforms[coder-agent:telegram] = %+v ok=%v", p, ok)
	}
	if h.ActiveAgents != 0 || h.UpdatedAt.IsZero() {
		t.Errorf("active_agents=%d updated_at=%v", h.ActiveAgents, h.UpdatedAt)
	}
}

func TestClient_DecodesSessionsAndMessagesFixtures(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "default/sessions.json"))
	})
	mux.HandleFunc("/api/sessions/{id}/messages", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "default/session_messages.json"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := newClient(t, srv.URL, time.Second)

	list, err := c.Sessions(context.Background(), SessionsQuery{})
	if err != nil {
		t.Fatalf("Sessions: %v", err)
	}
	if !list.HasMore || list.Limit != 5 || len(list.Data) != 5 {
		t.Errorf("list meta = has_more=%v limit=%d n=%d", list.HasMore, list.Limit, len(list.Data))
	}
	s := list.Data[0]
	if s.ID != "20260913_163024_5d910738" || s.Source != "telegram" || s.MessageCount != 427 || s.ToolCallCount != 152 {
		t.Errorf("session[0] = %+v", s)
	}
	if s.InputTokens != 1552865 || s.EstimatedCostUSD == nil || s.ActualCostUSD != nil {
		t.Errorf("usage/cost = in=%d est=%v act=%v", s.InputTokens, s.EstimatedCostUSD, s.ActualCostUSD)
	}
	if s.EndedAt != nil || s.EndReason != nil {
		t.Errorf("open session should have nil ended_at/end_reason: %+v %+v", s.EndedAt, s.EndReason)
	}
	if got := s.LastActive.Time().UTC().Format(time.RFC3339); got != "2026-09-19T12:14:10Z" {
		t.Errorf("last_active = %s, want 2026-09-19T12:14:10Z", got)
	}
	if list.Data[1].EndReason == nil || *list.Data[1].EndReason != "agent_close" {
		t.Errorf("session[1].end_reason = %v", list.Data[1].EndReason)
	}

	msgs, err := c.Messages(context.Background(), s.ID, MessagesQuery{Limit: 20})
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if msgs.SessionID != s.ID || msgs.Pagination.Returned != 20 || msgs.Pagination.Order != "oldest" {
		t.Errorf("messages meta = %+v", msgs.Pagination)
	}
	if m := msgs.Data[0]; m.ID != 194 || m.Role != "user" {
		t.Errorf("message[0] = %+v", m)
	}
	var sawToolCall, sawToolResult bool
	for _, m := range msgs.Data {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			sawToolCall = true
		}
		if m.Role == "tool" && m.ToolName != "" && m.ToolCallID != "" {
			sawToolResult = true
		}
	}
	if !sawToolCall || !sawToolResult {
		t.Errorf("expected both an assistant tool_calls message and a tool result message (got %v %v)", sawToolCall, sawToolResult)
	}
}

func TestClient_RunNotFoundIsNotFoundKind(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"run not found"}}`))
	}))
	defer srv.Close()
	_, err := newClient(t, srv.URL, time.Second).Run(context.Background(), "other-profile-run")
	var ue *UpstreamError
	if !errors.As(err, &ue) || ue.Kind != KindNotFound {
		t.Fatalf("err = %v, want KindNotFound", err)
	}
}

func TestClient_NonJSONBodyIsDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	defer srv.Close()
	_, err := newClient(t, srv.URL, time.Second).Capabilities(context.Background())
	var ue *UpstreamError
	if !errors.As(err, &ue) || ue.Kind != KindBadResponse {
		t.Fatalf("err = %v, want KindBadResponse", err)
	}
}

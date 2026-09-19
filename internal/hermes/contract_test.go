package hermes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestContract_AllProfileFixturesDecode replays every captured fixture through
// the typed client methods. When a Hermes upgrade changes a payload shape,
// re-running scripts/collect-fixtures.sh makes this test fail before any
// handler does.
func TestContract_AllProfileFixturesDecode(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "fixtures")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		profile := e.Name()
		t.Run(profile, func(t *testing.T) {
			dir := filepath.Join(root, profile)
			// Each fixture file is served at its own name; the client paths are
			// rewritten to those names through a tiny mux.
			mux := http.NewServeMux()
			serve := func(path, file string) {
				mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
					b, err := os.ReadFile(filepath.Join(dir, file))
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(b)
				})
			}
			serve("/health/detailed", "health_detailed.json")
			serve("/v1/capabilities", "capabilities.json")
			serve("/v1/models", "models.json")
			serve("/v1/toolsets", "toolsets.json")
			serve("/api/jobs", "jobs.json")
			serve("/api/sessions", "sessions.json")
			serve("/api/sessions/{id}", "session_detail.json")
			serve("/api/sessions/{id}/messages", "session_messages.json")
			fsrv := httptest.NewServer(mux)
			defer fsrv.Close()

			c := newClient(t, fsrv.URL, time.Second)
			ctx := context.Background()

			if h, err := c.HealthDetailed(ctx); err != nil || h.Version == "" {
				t.Errorf("HealthDetailed: %v", err)
			}
			if cp, err := c.Capabilities(ctx); err != nil || len(cp.Features) == 0 {
				t.Errorf("Capabilities: %v", err)
			}
			if _, err := c.Models(ctx); err != nil {
				t.Errorf("Models: %v", err)
			}
			if ts, err := c.Toolsets(ctx); err != nil || len(ts.Data) == 0 {
				t.Errorf("Toolsets: %v", err)
			}
			if _, err := c.Jobs(ctx); err != nil {
				t.Errorf("Jobs: %v", err)
			}
			list, err := c.Sessions(ctx, SessionsQuery{})
			if err != nil {
				t.Fatalf("Sessions: %v", err)
			}
			if len(list.Data) == 0 {
				return // product-agent has no sessions yet
			}
			if _, err := c.Session(ctx, list.Data[0].ID); err != nil {
				t.Errorf("Session: %v", err)
			}
			if m, err := c.Messages(ctx, list.Data[0].ID, MessagesQuery{}); err != nil || len(m.Data) == 0 {
				t.Errorf("Messages: %v", err)
			}
		})
	}
}

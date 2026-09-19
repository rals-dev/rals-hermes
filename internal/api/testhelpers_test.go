package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rals-dev/rals-hermes/internal/config"
	"github.com/rals-dev/rals-hermes/internal/hermes"
	"github.com/rals-dev/rals-hermes/internal/observ"
)

func fixture(t *testing.T, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", rel))
	if err != nil {
		t.Fatalf("fixture %s: %v", rel, err)
	}
	return b
}

// serveFixture answers every request with the named fixture file.
func serveFixture(t *testing.T, rel string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, rel))
	})
}

func serveStatus(status int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
	})
}

// hang blocks until the client gives up.
func hang() http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
}

// profileSpec pairs a profile name with the handler standing in for Hermes.
// A nil handler means "nothing listens" (connection refused).
type profileSpec struct {
	name    string
	handler http.Handler
}

// newTestDeps wires real hermes.Clients against httptest servers so the
// handlers are exercised end to end, transport included.
func newTestDeps(t *testing.T, timeout time.Duration, specs ...profileSpec) Deps {
	t.Helper()
	metrics := observ.New()
	clients := make([]*hermes.Client, 0, len(specs))
	for _, s := range specs {
		var base string
		if s.handler == nil {
			srv := httptest.NewServer(http.NotFoundHandler())
			base = srv.URL
			srv.Close()
		} else {
			srv := httptest.NewServer(s.handler)
			t.Cleanup(srv.Close)
			base = srv.URL
		}
		p := config.Profile{Name: s.name, BaseURL: base, Key: config.Secret("key-" + s.name)}
		clients = append(clients, hermes.NewClient(p, hermes.Options{
			Timeout:  timeout,
			Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			Observer: metrics,
		}))
	}
	return Deps{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version:    "test",
		Profiles:   clients,
		CacheTTL:   3 * time.Second,
		AuthKey:    config.Secret(bffKey),
		SessionTTL: time.Hour,
		Metrics:    metrics,
	}
}

// authedGet performs GET path with a freshly minted session cookie.
func authedGet(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	c := sessionCookie(t, login(t, h, bffKey))
	return getWithCookie(h, path, c)
}

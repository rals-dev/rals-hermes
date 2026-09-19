package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestRun_ServesUntilContextCancelledThenShutsDownGracefully(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	// A handler that takes a moment, so we can prove in-flight requests finish
	// during shutdown instead of being dropped.
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusTeapot)
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, ln, h, Options{
			Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
			ReadTimeout:     time.Second,
			ShutdownTimeout: 2 * time.Second,
		})
	}()

	url := "http://" + ln.Addr().String() + "/"
	resp, err := http.Get(url) //nolint:noctx // test helper
	if err != nil {
		t.Fatalf("GET before shutdown: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", resp.StatusCode)
	}

	// Start an in-flight request, then cancel while it is still running.
	inflight := make(chan int, 1)
	go func() {
		r, err := http.Get(url) //nolint:noctx // test helper
		if err != nil {
			inflight <- -1
			return
		}
		r.Body.Close()
		inflight <- r.StatusCode
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
	if code := <-inflight; code != http.StatusTeapot {
		t.Errorf("in-flight request got %d, want 418 (graceful drain)", code)
	}

	if _, err := http.Get(url); err == nil { //nolint:noctx,bodyclose // expecting failure
		t.Error("server still accepting connections after Run returned")
	}
}

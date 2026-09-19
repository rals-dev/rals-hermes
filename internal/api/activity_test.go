package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rals-dev/rals-hermes/internal/activity"
)

func TestActivityStream_EmitsSnapshotsThenDeltasAndKeepalives(t *testing.T) {
	// Fixture-backed profile: sessions.json has sessions whose last_active
	// is in 2026-09; the window in this test is huge so they are tracked.
	d := newTestDeps(t, time.Second, profileSpec{"default", hermesFixtureMux(t, "default", nil)})
	d.Activity = activity.Config{PollInterval: 20 * time.Millisecond, Window: 24 * 365 * time.Hour, MaxSessions: 2, IdleStopAfter: 50 * time.Millisecond}
	d.StreamKeepalive = 30 * time.Millisecond
	base, cookie := startBFF(t, d)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp := openStream(t, ctx, base, cookie, "/api/agents/default/activity/stream")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status=%d ct=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}

	sc := bufio.NewScanner(resp.Body)
	var events []string
	var sawKeepalive bool
	var snapshot map[string]any
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) && sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			events = append(events, strings.TrimPrefix(line, "event: "))
		case strings.HasPrefix(line, ": keepalive"):
			sawKeepalive = true
		case strings.HasPrefix(line, "data: ") && snapshot == nil:
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &snapshot); err != nil {
				t.Fatalf("data is not JSON: %v", err)
			}
		}
		if sawKeepalive && len(events) >= 2 {
			break
		}
	}
	if len(events) < 2 || events[0] != "session.snapshot" {
		t.Errorf("events = %v, want snapshots first (MaxSessions=2)", events)
	}
	if !sawKeepalive {
		t.Error("no keepalive comment within 1.5s")
	}
	if snapshot["profile"] != "default" || snapshot["session_id"] == "" || snapshot["type"] != "session.snapshot" {
		t.Errorf("snapshot payload = %v", snapshot)
	}
}

func TestActivityStream_UnknownProfileIs404AndDisconnectReleasesSubscription(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", hermesFixtureMux(t, "default", nil)})
	d.Activity = activity.Config{PollInterval: 20 * time.Millisecond, Window: 24 * 365 * time.Hour, MaxSessions: 2, IdleStopAfter: 50 * time.Millisecond}
	base, cookie := startBFF(t, d)

	resp := openStream(t, context.Background(), base, cookie, "/api/agents/ghost/activity/stream")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown profile: status = %d", resp.StatusCode)
	}

	ctx, cancel := context.WithCancel(context.Background())
	resp = openStream(t, ctx, base, cookie, "/api/agents/default/activity/stream")
	buf := make([]byte, 16)
	_, _ = resp.Body.Read(buf)
	cancel()
	resp.Body.Close()
	// The hub behind this server is private to it; assert through metrics.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		r, err := http.Get(base + "/metrics") //nolint:noctx // test
		if err == nil {
			b, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if strings.Contains(string(b), `bff_activity_subscribers{profile="default"} 0`) {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Error("subscriber gauge did not return to 0 after disconnect")
}

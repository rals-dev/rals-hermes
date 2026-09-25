package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// wireSession/wireSessionList mirror the real Hermes wire shape (T-003 § 3):
// timestamps are raw float seconds, not hermes.UnixTime's own MarshalJSON —
// that method renders RFC 3339 instead, for the BFF's *outbound* responses,
// and does not round-trip with UnixTime.UnmarshalJSON's "parse a float"
// expectation. Encoding a hermes.Session directly would silently send a
// payload the client itself cannot parse.
type wireSession struct {
	ID               string   `json:"id"`
	Source           string   `json:"source"`
	Model            string   `json:"model"`
	StartedAt        float64  `json:"started_at"`
	LastActive       float64  `json:"last_active"`
	InputTokens      int64    `json:"input_tokens"`
	OutputTokens     int64    `json:"output_tokens"`
	EstimatedCostUSD *float64 `json:"estimated_cost_usd"`
	ActualCostUSD    *float64 `json:"actual_cost_usd"`
}

type wireSessionList struct {
	Object  string        `json:"object"`
	Data    []wireSession `json:"data"`
	HasMore bool          `json:"has_more"`
}

// sessionsPageHandler answers every request with the same wireSessionList,
// counting how many times it was hit — the tool that lets these tests assert
// on pagination behaviour (or its absence) without a real Hermes.
type sessionsPageHandler struct {
	t    *testing.T
	list wireSessionList
	hits int
}

func (h *sessionsPageHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.hits++
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.list); err != nil {
		h.t.Fatalf("encode sessions page: %v", err)
	}
}

func sess(id string, lastActive time.Time, in, out int64, estCost, actCost *float64) wireSession {
	return wireSession{
		ID: id, Source: "cli", Model: "x",
		StartedAt: float64(lastActive.Add(-time.Minute).Unix()), LastActive: float64(lastActive.Unix()),
		InputTokens: in, OutputTokens: out, EstimatedCostUSD: estCost, ActualCostUSD: actCost,
	}
}

func f64(v float64) *float64 { return &v }

type usageBody struct {
	GeneratedAt time.Time `json:"generated_at"`
	WindowHours int       `json:"window_hours"`
	Profiles    []struct {
		Profile      string `json:"profile"`
		SessionCount int    `json:"session_count"`
		Usage        struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
		EstimatedCostUSD *float64 `json:"estimated_cost_usd"`
		ActualCostUSD    *float64 `json:"actual_cost_usd"`
		Truncated        bool     `json:"truncated"`
	} `json:"profiles"`
	Errors []profileError `json:"errors"`
}

func getUsage(t *testing.T, d Deps) (int, usageBody) {
	t.Helper()
	rec := authedGet(t, NewHandler(d), "/api/usage")
	var body usageBody
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v (%s)", err, rec.Body)
		}
	}
	return rec.Code, body
}

func TestUsage_SumsWithinWindowAndToleratesADeadProfile(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	inWindow := &sessionsPageHandler{t: t, list: wireSessionList{
		Object: "list",
		Data: []wireSession{
			sess("s1", now.Add(-1*time.Hour), 100, 10, f64(0.01), nil),
			sess("s2", now.Add(-23*time.Hour), 50, 5, f64(0.02), nil),
			sess("s3", now.Add(-25*time.Hour), 999, 999, f64(9.99), nil), // outside the 24h window
		},
	}}
	d := newTestDeps(t, time.Second,
		profileSpec{"default", inWindow},
		profileSpec{"tester-agent", nil}, // dead: connection refused
	)
	d.now = func() time.Time { return now }

	code, body := getUsage(t, d)
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if body.WindowHours != 24 {
		t.Errorf("window_hours = %d, want 24", body.WindowHours)
	}
	if len(body.Profiles) != 1 {
		t.Fatalf("profiles = %+v, want exactly 'default'", body.Profiles)
	}
	p := body.Profiles[0]
	if p.Profile != "default" || p.SessionCount != 2 {
		t.Errorf("default summary = %+v, want session_count=2 (s3 is outside the window)", p)
	}
	if p.Usage.InputTokens != 150 || p.Usage.OutputTokens != 15 {
		t.Errorf("token sums = %+v, want 150/15 (s1+s2 only)", p.Usage)
	}
	if p.EstimatedCostUSD == nil || *p.EstimatedCostUSD != 0.03 {
		t.Errorf("estimated_cost_usd = %v, want 0.03", p.EstimatedCostUSD)
	}
	if p.ActualCostUSD != nil {
		t.Errorf("actual_cost_usd = %v, want null (no session reported one)", *p.ActualCostUSD)
	}
	if len(body.Errors) != 1 || body.Errors[0].Profile != "tester-agent" || body.Errors[0].Error.Code != "upstream_unreachable" {
		t.Errorf("errors = %+v, want one upstream_unreachable for tester-agent", body.Errors)
	}
}

func TestUsage_StopsPagingAsSoonAsASessionAgesOutOfTheWindow(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	// Hermes orders by last_active descending; the third entry is already
	// outside the window, so a second page must never be requested even
	// though has_more says there is one.
	h := &sessionsPageHandler{t: t, list: wireSessionList{
		Object:  "list",
		HasMore: true,
		Data: []wireSession{
			sess("s1", now.Add(-1*time.Hour), 1, 1, nil, nil),
			sess("s2", now.Add(-2*time.Hour), 1, 1, nil, nil),
			sess("s3", now.Add(-30*time.Hour), 1, 1, nil, nil),
		},
	}}
	d := newTestDeps(t, time.Second, profileSpec{"default", h})
	d.now = func() time.Time { return now }

	code, body := getUsage(t, d)
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if h.hits != 1 {
		t.Errorf("upstream hits = %d, want exactly 1 (must not fetch a second page)", h.hits)
	}
	if len(body.Profiles) != 1 || body.Profiles[0].SessionCount != 2 {
		t.Errorf("profiles = %+v, want session_count=2", body.Profiles)
	}
}

func TestUsage_CapsRunawayPaginationAndFlagsTruncated(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	// Every page reports one fresh (in-window) session and claims there's
	// more — a pathological upstream that never lets the window cutoff end
	// the scan naturally. The safety cap must still stop it.
	h := &sessionsPageHandler{t: t, list: wireSessionList{
		Object: "list", HasMore: true,
		Data: []wireSession{sess("s", now.Add(-time.Minute), 1, 1, nil, nil)},
	}}
	d := newTestDeps(t, time.Second, profileSpec{"default", h})
	d.now = func() time.Time { return now }

	code, body := getUsage(t, d)
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if h.hits != usageMaxPages {
		t.Errorf("upstream hits = %d, want exactly usageMaxPages (%d)", h.hits, usageMaxPages)
	}
	if len(body.Profiles) != 1 || !body.Profiles[0].Truncated {
		t.Errorf("profiles = %+v, want truncated=true", body.Profiles)
	}
}

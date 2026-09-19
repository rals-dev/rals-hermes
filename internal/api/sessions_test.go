package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessions_ForwardsPaginationAndReturnsList(t *testing.T) {
	var gotQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write(fixture(t, "coder-agent/sessions.json"))
	})
	d := newTestDeps(t, time.Second, profileSpec{"coder-agent", mux})
	rec := authedGet(t, NewHandler(d), "/api/agents/coder-agent/sessions?limit=5&offset=10&source=kanban")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	for _, want := range []string{"limit=5", "offset=10", "source=kanban", "include_children=true"} {
		if !contains(gotQuery, want) {
			t.Errorf("upstream query %q lacks %s", gotQuery, want)
		}
	}
	var body struct {
		Profile string `json:"profile"`
		HasMore bool   `json:"has_more"`
		Limit   int    `json:"limit"`
		Offset  int    `json:"offset"`
		Data    []struct {
			ID         string `json:"id"`
			Source     string `json:"source"`
			LastActive string `json:"last_active"`
			Usage      struct {
				Input  int64 `json:"input_tokens"`
				Output int64 `json:"output_tokens"`
			} `json:"usage"`
			EstimatedCostUSD *float64 `json:"estimated_cost_usd"`
			Open             bool     `json:"open"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Profile != "coder-agent" || len(body.Data) != 5 || body.Data[0].Source != "kanban" {
		t.Errorf("body = %+v", body)
	}
	if body.Data[0].LastActive == "" || body.Data[0].Usage.Input == 0 {
		t.Errorf("session projection missing last_active/usage: %+v", body.Data[0])
	}
	if body.Data[0].Open { // fixture sessions of coder-agent all ended with cli_close
		t.Errorf("session[0].open = true, want false (end_reason set)")
	}
}

func TestSessions_RejectsOutOfRangeParameters(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", hermesFixtureMux(t, "default", nil)})
	h := NewHandler(d)
	for _, q := range []string{"limit=0", "limit=201", "limit=abc", "offset=-1", "source=../etc"} {
		rec := authedGet(t, h, "/api/agents/default/sessions?"+q)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", q, rec.Code)
		}
		var body errorBody
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body.Code != "bad_request" {
			t.Errorf("%s: code = %q", q, body.Code)
		}
	}
}

func TestSessionDetail_MergesMetadataAndMessages(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/api/sessions/{id}", serveFixture(t, "default/session_detail.json"))
	mux.Handle("/api/sessions/{id}/messages", serveFixture(t, "default/session_messages.json"))
	d := newTestDeps(t, time.Second, profileSpec{"default", mux})
	rec := authedGet(t, NewHandler(d), "/api/agents/default/sessions/20260913_163024_5d910738?limit=20")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Profile string `json:"profile"`
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
		Messages struct {
			Pagination struct {
				Returned int `json:"returned"`
			} `json:"pagination"`
			Data []struct {
				Role     string `json:"role"`
				ToolName string `json:"tool_name,omitempty"`
			} `json:"data"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Profile != "default" || body.Session.ID != "20260913_163024_5d910738" || body.Messages.Pagination.Returned != 20 || len(body.Messages.Data) != 20 {
		t.Errorf("body = %+v", body)
	}
}

func TestSessionDetail_UnknownSessionIs404SessionNotFound(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", serveStatus(404)})
	rec := authedGet(t, NewHandler(d), "/api/agents/default/sessions/nope")
	var body errorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusNotFound || body.Code != "session_not_found" {
		t.Errorf("status=%d code=%q", rec.Code, body.Code)
	}
}

func TestRun_StatusAndCrossProfile404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/runs/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") != "run-abc" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"run not found"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"run-abc","status":"running","object":"run"}`))
	})
	d := newTestDeps(t, time.Second, profileSpec{"coder-agent", mux}, profileSpec{"tester-agent", mux})
	h := NewHandler(d)

	rec := authedGet(t, h, "/api/agents/coder-agent/runs/run-abc")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Profile string         `json:"profile"`
		Status  string         `json:"status"`
		Run     map[string]any `json:"run"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Profile != "coder-agent" || body.Status != "running" || body.Run["object"] != "run" {
		t.Errorf("body = %+v", body)
	}

	// Acceptance test 6: a run id owned by another profile is 404 run_not_found.
	rec = authedGet(t, h, "/api/agents/tester-agent/runs/run-owned-by-coder")
	var eb errorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &eb)
	if rec.Code != http.StatusNotFound || eb.Code != "run_not_found" {
		t.Errorf("cross-profile: status=%d code=%q, want 404 run_not_found", rec.Code, eb.Code)
	}
	_ = httptest.NewRecorder
}

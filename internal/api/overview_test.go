package api

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type overviewResp struct {
	GeneratedAt time.Time `json:"generated_at"`
	Gateway     *struct {
		Version           string `json:"version"`
		State             string `json:"state"`
		ActiveAgents      int    `json:"active_agents"`
		ActiveAPIRuns     int    `json:"active_api_runs"`
		ActiveDelegations int    `json:"active_delegations"`
		Readiness         string `json:"readiness"`
		SourceProfile     string `json:"source_profile"`
	} `json:"gateway"`
	Agents []cardView `json:"agents"`
}

type cardView struct {
	Profile   string `json:"profile"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Error     *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Platforms map[string]struct {
		State string `json:"state"`
	} `json:"platforms"`
}

func getOverview(t *testing.T, h http.Handler) (int, overviewResp) {
	t.Helper()
	rec := authedGet(t, h, "/api/overview")
	var body overviewResp
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v: %s", err, rec.Body)
		}
	}
	return rec.Code, body
}

func TestOverview_MixedHealthyUnauthorizedAndDeadProfiles(t *testing.T) {
	deps := newTestDeps(t, time.Second,
		profileSpec{"default", serveFixture(t, "default/health_detailed.json")},
		profileSpec{"coder-agent", serveFixture(t, "coder-agent/health_detailed.json")},
		profileSpec{"tester-agent", serveStatus(401)},
		profileSpec{"product-agent", nil},
	)
	code, body := getOverview(t, NewHandler(deps))
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even with dead profiles", code)
	}

	want := map[string]string{"default": "healthy", "coder-agent": "healthy", "tester-agent": "unauthorized", "product-agent": "unreachable"}
	if len(body.Agents) != 4 {
		t.Fatalf("agents = %d, want 4 in config order", len(body.Agents))
	}
	for i, name := range []string{"default", "coder-agent", "tester-agent", "product-agent"} {
		a := body.Agents[i]
		if a.Profile != name || a.Status != want[name] {
			t.Errorf("agents[%d] = %s/%s, want %s/%s", i, a.Profile, a.Status, name, want[name])
		}
		if a.LatencyMS < 0 {
			t.Errorf("agents[%d].latency_ms = %d", i, a.LatencyMS)
		}
	}
	if e := body.Agents[2].Error; e == nil || e.Code != "upstream_unauthorized" {
		t.Errorf("tester-agent error = %+v, want upstream_unauthorized", e)
	}
	if e := body.Agents[3].Error; e == nil || e.Code != "upstream_unreachable" {
		t.Errorf("product-agent error = %+v, want upstream_unreachable", e)
	}
	if body.Agents[2].Platforms != nil || body.Agents[3].Platforms != nil {
		t.Error("failed profiles must not carry platform data")
	}

	// Gateway-wide block comes from the first healthy profile.
	if body.Gateway == nil {
		t.Fatal("gateway block missing")
	}
	if body.Gateway.Version != "0.21.2" || body.Gateway.State != "running" || body.Gateway.SourceProfile != "default" {
		t.Errorf("gateway = %+v", *body.Gateway)
	}
	// Per-profile platform entries are filtered from the gateway-wide map.
	if p, ok := body.Agents[1].Platforms["telegram"]; !ok || p.State != "connected" {
		t.Errorf("coder-agent platforms = %+v, want telegram=connected (from coder-agent:telegram)", body.Agents[1].Platforms)
	}
	if _, leak := body.Agents[1].Platforms["tester-agent:telegram"]; leak {
		t.Error("coder-agent card leaks tester-agent platform entry")
	}
	if p, ok := body.Agents[0].Platforms["telegram"]; !ok || p.State != "connected" {
		t.Errorf("default platforms = %+v, want telegram=connected", body.Agents[0].Platforms)
	}
	if body.GeneratedAt.IsZero() {
		t.Error("generated_at missing")
	}
}

func TestOverview_AllProfilesDeadStillReturns200WithNoGateway(t *testing.T) {
	deps := newTestDeps(t, 200*time.Millisecond, profileSpec{"a", nil}, profileSpec{"b", nil})
	code, body := getOverview(t, NewHandler(deps))
	if code != http.StatusOK || body.Gateway != nil || len(body.Agents) != 2 {
		t.Errorf("code=%d gateway=%v agents=%d", code, body.Gateway, len(body.Agents))
	}
}

func TestOverview_PerProfileTimeoutDoesNotHoldThePage(t *testing.T) {
	deps := newTestDeps(t, 150*time.Millisecond,
		profileSpec{"fast", serveFixture(t, "default/health_detailed.json")},
		profileSpec{"slow", hang()},
	)
	start := time.Now()
	code, body := getOverview(t, NewHandler(deps))
	elapsed := time.Since(start)
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if elapsed > 600*time.Millisecond {
		t.Errorf("overview took %v with a hanging profile; timeout not enforced", elapsed)
	}
	if body.Agents[0].Status != "healthy" || body.Agents[1].Status != "unreachable" {
		t.Errorf("statuses = %s/%s", body.Agents[0].Status, body.Agents[1].Status)
	}
}

func TestOverview_DegradedWhenReadinessNotOK(t *testing.T) {
	degraded := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","readiness":{"status":"degraded","checks":{"disk":{"status":"warn","used_percent":97}}},"version":"0.21.2","gateway_state":"running","platforms":{}}`))
	})
	deps := newTestDeps(t, time.Second, profileSpec{"default", degraded})
	_, body := getOverview(t, NewHandler(deps))
	if body.Agents[0].Status != "degraded" {
		t.Errorf("status = %s, want degraded (HTTP 200 must not imply healthy)", body.Agents[0].Status)
	}
	if body.Gateway == nil || body.Gateway.Readiness != "degraded" {
		t.Errorf("gateway readiness = %+v", body.Gateway)
	}
}

func TestOverview_SecondCallWithinTTLDoesNotHitUpstream(t *testing.T) {
	var hits atomic.Int32
	counting := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(fixture(t, "default/health_detailed.json"))
	})
	deps := newTestDeps(t, time.Second, profileSpec{"default", counting}, profileSpec{"coder-agent", counting})
	h := NewHandler(deps)
	getOverview(t, h)
	getOverview(t, h)
	if n := hits.Load(); n != 2 {
		t.Errorf("upstream hit %d times for two overview calls, want 2 (one per profile)", n)
	}
}

func TestAgents_ListsProfilesInConfigOrderWithStatus(t *testing.T) {
	deps := newTestDeps(t, time.Second,
		profileSpec{"default", serveFixture(t, "default/health_detailed.json")},
		profileSpec{"tester-agent", serveStatus(401)},
	)
	rec := authedGet(t, NewHandler(deps), "/api/agents")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Agents []cardView `json:"agents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Agents) != 2 || body.Agents[0].Profile != "default" || body.Agents[0].Status != "healthy" ||
		body.Agents[1].Profile != "tester-agent" || body.Agents[1].Status != "unauthorized" {
		t.Errorf("agents = %+v", body.Agents)
	}
}

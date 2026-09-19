package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// hermesFixtureMux serves the captured fixtures of one profile on their real paths.
func hermesFixtureMux(t *testing.T, profile string, overrides map[string]http.Handler) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	routes := map[string]string{
		"/health/detailed": "health_detailed.json",
		"/v1/capabilities": "capabilities.json",
		"/v1/models":       "models.json",
		"/v1/toolsets":     "toolsets.json",
		"/v1/skills":       "skills.json",
		"/api/jobs":        "jobs.json",
		"/api/sessions":    "sessions.json",
	}
	for path, file := range routes {
		if h, ok := overrides[path]; ok {
			mux.Handle(path, h)
			continue
		}
		mux.Handle(path, serveFixture(t, profile+"/"+file))
	}
	return mux
}

func TestAgentDetail_MergesHealthCapabilitiesToolsetsAndModels(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", hermesFixtureMux(t, "default", map[string]http.Handler{
		"/v1/skills": serveStatus(500), // exactly what 0.21.2 does
	})})
	rec := authedGet(t, NewHandler(d), "/api/agents/default")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Profile      string          `json:"profile"`
		Status       string          `json:"status"`
		Capabilities json.RawMessage `json:"capabilities"`
		Toolsets     []struct {
			Name  string   `json:"name"`
			Tools []string `json:"tools"`
		} `json:"toolsets"`
		Models    []string        `json:"models"`
		Skills    json.RawMessage `json:"skills"`
		Warnings  []string        `json:"warnings"`
		Platforms map[string]any  `json:"platforms"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Profile != "default" || body.Status != "healthy" {
		t.Errorf("profile/status = %s/%s", body.Profile, body.Status)
	}
	if len(body.Capabilities) == 0 || len(body.Toolsets) == 0 || body.Toolsets[0].Name == "" || len(body.Models) == 0 {
		t.Errorf("merged sections missing: caps=%d toolsets=%d models=%d", len(body.Capabilities), len(body.Toolsets), len(body.Models))
	}
	if string(body.Skills) != "null" {
		t.Errorf("skills should be null when upstream fails, got %s", body.Skills)
	}
	if len(body.Warnings) != 1 || body.Warnings[0] != "skills_unavailable" {
		t.Errorf("warnings = %v, want [skills_unavailable]", body.Warnings)
	}
	if _, ok := body.Platforms["telegram"]; !ok {
		t.Errorf("platforms = %v, want telegram entry", body.Platforms)
	}
}

func TestAgentDetail_UnknownProfileIs404(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", hermesFixtureMux(t, "default", nil)})
	rec := authedGet(t, NewHandler(d), "/api/agents/ghost")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var body errorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Code != "profile_not_found" {
		t.Errorf("code = %q", body.Code)
	}
}

func TestAgentDetail_UnreachableProfileIs503NotCrash(t *testing.T) {
	d := newTestDeps(t, 200*time.Millisecond, profileSpec{"default", nil})
	rec := authedGet(t, NewHandler(d), "/api/agents/default")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", rec.Code, rec.Body)
	}
	var body errorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Code != "upstream_unreachable" {
		t.Errorf("code = %q", body.Code)
	}
}

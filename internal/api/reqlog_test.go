package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestLog_OneJSONLinePerRequestWithoutCredentials(t *testing.T) {
	var buf bytes.Buffer
	d := newTestDeps(t, time.Second, profileSpec{"default", serveFixture(t, "default/health_detailed.json")})
	d.Logger = slog.New(slog.NewJSONHandler(&buf, nil))
	h := NewHandler(d)

	c := sessionCookie(t, login(t, h, bffKey))
	buf.Reset()

	req := httptest.NewRequest(http.MethodGet, "/api/agents?limit=3", nil)
	req.AddCookie(c)
	req.Header.Set("Authorization", "Bearer should-never-be-logged")
	req.Header.Set("X-Forwarded-For", "100.64.0.9")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var entry map[string]any
	found := 0
	for _, l := range lines {
		var m map[string]any
		if json.Unmarshal([]byte(l), &m) == nil && m["msg"] == "request" {
			found++
			entry = m
		}
	}
	if found != 1 {
		t.Fatalf("want exactly one request log line, got %d:\n%s", found, buf.String())
	}
	for _, k := range []string{"method", "path", "status", "duration_ms", "bytes", "remote"} {
		if _, ok := entry[k]; !ok {
			t.Errorf("request log lacks %q: %v", k, entry)
		}
	}
	if entry["path"] != "/api/agents" || entry["status"] != float64(200) || entry["method"] != "GET" {
		t.Errorf("entry = %v", entry)
	}
	for _, forbidden := range []string{c.Value, "should-never-be-logged", bffKey} {
		if strings.Contains(buf.String(), forbidden) {
			t.Errorf("request log leaks %q", forbidden)
		}
	}
}

func TestRequestLog_CarriesProfileFromRoute(t *testing.T) {
	var buf bytes.Buffer
	d := newTestDeps(t, time.Second, profileSpec{"default", serveFixture(t, "default/health_detailed.json")})
	d.Logger = slog.New(slog.NewJSONHandler(&buf, nil))
	h := NewHandler(d)
	c := sessionCookie(t, login(t, h, bffKey))
	buf.Reset()

	req := httptest.NewRequest(http.MethodGet, "/api/agents/default", nil)
	req.AddCookie(c)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !strings.Contains(buf.String(), `"profile":"default"`) {
		t.Errorf("request log for a profile route must carry profile=default:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), `"route":"GET /api/agents/{profile}"`) {
		t.Errorf("request log must carry the matched route pattern:\n%s", buf.String())
	}
}

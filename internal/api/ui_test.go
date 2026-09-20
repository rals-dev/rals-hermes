package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func uiFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":        {Data: []byte("<!doctype html><title>Hermes</title><div id=app></div>")},
		"assets/app-1a.js":  {Data: []byte("console.log('app')")},
		"assets/app-1a.css": {Data: []byte("body{}")},
		"favicon.svg":       {Data: []byte("<svg/>")},
	}
}

func TestUI_ServesIndexAssetsAndSPAFallback(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", serveFixture(t, "default/health_detailed.json")})
	d.UI = uiFS()
	h := NewHandler(d)

	cases := []struct {
		path     string
		wantCT   string
		wantBody string
		wantCode int
	}{
		{"/", "text/html", "<div id=app>", 200},
		{"/agents/default", "text/html", "<div id=app>", 200},                 // SPA route
		{"/agents/default/sessions/2026_x", "text/html", "<div id=app>", 200}, // deep SPA route
		{"/assets/app-1a.js", "text/javascript", "console.log", 200},
		{"/assets/app-1a.css", "text/css", "body{}", 200},
		{"/favicon.svg", "image/svg+xml", "<svg/>", 200},
		{"/assets/missing.js", "application/json", "not_found", 404}, // real asset misses are 404, not index
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.wantCode, rec.Body)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, tc.wantCT) {
				t.Errorf("Content-Type = %q, want prefix %q", ct, tc.wantCT)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Errorf("body %q lacks %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestUI_HashedAssetsAreCacheableIndexIsNot(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", serveFixture(t, "default/health_detailed.json")})
	d.UI = uiFS()
	h := NewHandler(d)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app-1a.js", nil))
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("asset Cache-Control = %q, want immutable", cc)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
		t.Errorf("index Cache-Control = %q, want no-cache", cc)
	}
}

func TestUI_APIRoutesAreNeverShadowedByTheSPA(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", serveFixture(t, "default/health_detailed.json")})
	d.UI = uiFS()
	h := NewHandler(d)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/nope", nil))
	var body errorBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusNotFound || body.Code != "not_found" {
		t.Errorf("/api/nope → %d %q, want JSON 404 not index.html", rec.Code, body.Code)
	}
}

func TestUI_WithoutBundleRootIsJSON404(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"default", serveFixture(t, "default/health_detailed.json")})
	rec := httptest.NewRecorder()
	NewHandler(d).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "not_found") {
		t.Errorf("without a UI bundle: %d %s", rec.Code, rec.Body)
	}
}

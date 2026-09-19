package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsEndpoint_IsOpenAndReflectsTraffic(t *testing.T) {
	d := newTestDeps(t, time.Second,
		profileSpec{"default", serveFixture(t, "default/health_detailed.json")},
		profileSpec{"product-agent", nil},
	)
	h := NewHandler(d)

	if rec := authedGet(t, h, "/api/overview"); rec.Code != http.StatusOK {
		t.Fatalf("overview: %d", rec.Code)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil)) // no cookie on purpose
	if rec.Code != http.StatusOK {
		t.Fatalf("/metrics must be reachable without a session, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`bff_profile_up{profile="default"} 1`,
		`bff_profile_up{profile="product-agent"} 0`,
		`bff_gateway_active_agents 0`,
		`bff_upstream_errors_total{kind="unreachable",profile="product-agent"} 1`,
		`bff_http_requests_total{route="GET /api/overview",status="200"} 1`,
		`bff_cache_misses_total{cache="health"} 2`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics lack %q", want)
		}
	}
}

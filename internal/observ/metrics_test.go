package observ

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

func scrape(t *testing.T, m *Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("scrape status = %d", rec.Code)
	}
	return rec.Body.String()
}

func mustContain(t *testing.T, body string, lines ...string) {
	t.Helper()
	for _, l := range lines {
		if !strings.Contains(body, l) {
			t.Errorf("metrics output lacks %q", l)
		}
	}
}

func TestMetrics_UpstreamObservationsAreLabelledByProfileOpAndOutcome(t *testing.T) {
	m := New()
	m.ObserveUpstream("default", "GET /health/detailed", 200, 0, 42*time.Millisecond)
	m.ObserveUpstream("default", "GET /health/detailed", 200, 0, 40*time.Millisecond)
	m.ObserveUpstream("tester-agent", "GET /health/detailed", 401, hermes.KindUnauthorized, 5*time.Millisecond)
	m.ObserveUpstream("product-agent", "GET /health/detailed", 0, hermes.KindUnreachable, 2*time.Second)

	body := scrape(t, m)
	mustContain(t, body,
		`bff_upstream_request_duration_seconds_count{op="GET /health/detailed",outcome="ok",profile="default"} 2`,
		`bff_upstream_request_duration_seconds_count{op="GET /health/detailed",outcome="error",profile="tester-agent"} 1`,
		`bff_upstream_errors_total{kind="unauthorized",profile="tester-agent"} 1`,
		`bff_upstream_errors_total{kind="unreachable",profile="product-agent"} 1`,
	)
}

func TestMetrics_ProfileAndGatewayGauges(t *testing.T) {
	m := New()
	m.SetProfileUp("default", true)
	m.SetProfileUp("product-agent", false)
	m.SetGateway(1, 0, 2)

	mustContain(t, scrape(t, m),
		`bff_profile_up{profile="default"} 1`,
		`bff_profile_up{profile="product-agent"} 0`,
		`bff_gateway_active_agents 1`,
		`bff_gateway_active_api_runs 0`,
		`bff_gateway_active_delegations 2`,
	)
}

func TestMetrics_HTTPRequestsByRouteAndStatus(t *testing.T) {
	m := New()
	m.ObserveHTTP("GET /api/overview", 200, 12*time.Millisecond)
	m.ObserveHTTP("GET /api/overview", 200, 8*time.Millisecond)
	m.ObserveHTTP("GET /api/agents/{profile}", 404, time.Millisecond)

	mustContain(t, scrape(t, m),
		`bff_http_requests_total{route="GET /api/overview",status="200"} 2`,
		`bff_http_requests_total{route="GET /api/agents/{profile}",status="404"} 1`,
		`bff_http_request_duration_seconds_count{route="GET /api/overview"} 2`,
	)
}

func TestMetrics_CacheStatsAreReadLazily(t *testing.T) {
	m := New()
	hits, misses := uint64(0), uint64(0)
	m.RegisterCache("health", func() (uint64, uint64) { return hits, misses })
	hits, misses = 7, 3
	mustContain(t, scrape(t, m),
		`bff_cache_hits_total{cache="health"} 7`,
		`bff_cache_misses_total{cache="health"} 3`,
	)
}

func TestMetrics_GoRuntimeMetricsArePresent(t *testing.T) {
	body := scrape(t, New())
	mustContain(t, body, "go_goroutines")
	// The process collector reads /proc on Linux; on Darwin it needs cgo,
	// which pure-Go builds (the production configuration) do not have.
	if runtime.GOOS == "linux" {
		mustContain(t, body, "process_resident_memory_bytes")
	}
}

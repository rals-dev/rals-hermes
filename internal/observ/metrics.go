// Package observ holds the Prometheus instrumentation of the BFF.
//
// Everything is registered on a private registry so tests can build
// independent instances, and the /metrics handler serves exactly this
// registry plus the Go runtime and process collectors.
package observ

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// Metrics owns every collector the BFF exposes.
type Metrics struct {
	reg *prometheus.Registry

	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec

	upstreamDuration *prometheus.HistogramVec
	upstreamErrors   *prometheus.CounterVec

	profileUp         *prometheus.GaugeVec
	activeAgents      prometheus.Gauge
	activeAPIRuns     prometheus.Gauge
	activeDelegations prometheus.Gauge

	activitySubscribers *prometheus.GaugeVec
}

// New builds and registers all collectors.
func New() *Metrics {
	m := &Metrics{
		reg: prometheus.NewRegistry(),
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "bff_http_requests_total", Help: "BFF HTTP requests by matched route and status.",
		}, []string{"route", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "bff_http_request_duration_seconds", Help: "BFF HTTP request latency by route.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"route"}),
		upstreamDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "bff_upstream_request_duration_seconds", Help: "Latency of calls to Hermes by profile, operation and outcome.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2, 5},
		}, []string{"profile", "op", "outcome"}),
		upstreamErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "bff_upstream_errors_total", Help: "Failed calls to Hermes by profile and error kind.",
		}, []string{"profile", "kind"}),
		profileUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "bff_profile_up", Help: "1 when the profile's last health probe was healthy, else 0.",
		}, []string{"profile"}),
		activeAgents: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bff_gateway_active_agents", Help: "Gateway-wide active agents from the last successful health probe.",
		}),
		activeAPIRuns: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bff_gateway_active_api_runs", Help: "Gateway-wide active API runs from the last successful health probe.",
		}),
		activeDelegations: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "bff_gateway_active_delegations", Help: "Gateway-wide active delegations from the last successful health probe.",
		}),
		activitySubscribers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "bff_activity_subscribers", Help: "Open activity-stream subscriptions by profile.",
		}, []string{"profile"}),
	}
	m.reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.httpRequests, m.httpDuration,
		m.upstreamDuration, m.upstreamErrors,
		m.profileUp, m.activeAgents, m.activeAPIRuns, m.activeDelegations,
		m.activitySubscribers,
	)
	return m
}

// Handler serves the Prometheus exposition format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

// ObserveHTTP records one BFF request.
func (m *Metrics) ObserveHTTP(route string, status int, d time.Duration) {
	m.httpRequests.WithLabelValues(route, strconv.Itoa(status)).Inc()
	m.httpDuration.WithLabelValues(route).Observe(d.Seconds())
}

// ObserveUpstream records one call to Hermes. kind is zero on success. It
// satisfies hermes.Observer.
func (m *Metrics) ObserveUpstream(profile, op string, _ int, kind hermes.Kind, d time.Duration) {
	outcome := "ok"
	if kind != 0 {
		outcome = "error"
		m.upstreamErrors.WithLabelValues(profile, kind.String()).Inc()
	}
	m.upstreamDuration.WithLabelValues(profile, op, outcome).Observe(d.Seconds())
}

// SetProfileUp records the result of a profile health probe.
func (m *Metrics) SetProfileUp(profile string, up bool) {
	v := 0.0
	if up {
		v = 1
	}
	m.profileUp.WithLabelValues(profile).Set(v)
}

// SetGateway records the gateway-wide counters.
func (m *Metrics) SetGateway(activeAgents, activeAPIRuns, activeDelegations int) {
	m.activeAgents.Set(float64(activeAgents))
	m.activeAPIRuns.Set(float64(activeAPIRuns))
	m.activeDelegations.Set(float64(activeDelegations))
}

// SetActivitySubscribers records the number of open feed subscriptions.
func (m *Metrics) SetActivitySubscribers(profile string, n int) {
	m.activitySubscribers.WithLabelValues(profile).Set(float64(n))
}

// RegisterCache exposes a cache's counters, read at scrape time.
func (m *Metrics) RegisterCache(name string, stats func() (hits, misses uint64)) {
	labels := prometheus.Labels{"cache": name}
	m.reg.MustRegister(
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "bff_cache_hits_total", Help: "Cache hits.", ConstLabels: labels,
		}, func() float64 { h, _ := stats(); return float64(h) }),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Name: "bff_cache_misses_total", Help: "Cache misses.", ConstLabels: labels,
		}, func() float64 { _, mi := stats(); return float64(mi) }),
	)
}

package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// Agent status values (PRD § 5).
const (
	statusHealthy      = "healthy"
	statusDegraded     = "degraded"
	statusUnreachable  = "unreachable"
	statusUnauthorized = "unauthorized"
)

// agentCard is one profile's entry in /api/overview and /api/agents.
type agentCard struct {
	Profile   string                          `json:"profile"`
	Status    string                          `json:"status"`
	LatencyMS int64                           `json:"latency_ms"`
	Error     *errorBody                      `json:"error,omitempty"`
	Platforms map[string]hermes.PlatformState `json:"platforms,omitempty"`
}

// gatewayBlock is the gateway-wide part of /health/detailed, reported once.
// /health/detailed is identical on every profile prefix (T-003 § 2), so the
// first healthy answer is authoritative.
type gatewayBlock struct {
	SourceProfile      string            `json:"source_profile"`
	Version            string            `json:"version"`
	State              string            `json:"state"`
	Readiness          string            `json:"readiness"`
	ActiveAgents       int               `json:"active_agents"`
	ActiveAPIRuns      int               `json:"active_api_runs"`
	ActiveDelegations  int               `json:"active_delegations"`
	ProcessCompletions int               `json:"process_completions"`
	Busy               bool              `json:"busy"`
	Disk               hermes.DiskCheck  `json:"disk"`
	Checks             map[string]string `json:"checks"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type overviewResponse struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Gateway     *gatewayBlock `json:"gateway"`
	Agents      []agentCard   `json:"agents"`
}

type agentsResponse struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Agents      []agentCard `json:"agents"`
}

// probeResult is one profile's health fetch.
type probeResult struct {
	card   agentCard
	health *hermes.HealthDetailed
}

func (h *handlers) overview(w http.ResponseWriter, r *http.Request) {
	results := h.probeAll(r.Context())
	resp := overviewResponse{GeneratedAt: time.Now().UTC(), Agents: make([]agentCard, 0, len(results))}
	for _, res := range results {
		resp.Agents = append(resp.Agents, res.card)
		if resp.Gateway == nil && res.health != nil {
			resp.Gateway = newGatewayBlock(res.card.Profile, res.health)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) agents(w http.ResponseWriter, r *http.Request) {
	results := h.probeAll(r.Context())
	resp := agentsResponse{GeneratedAt: time.Now().UTC(), Agents: make([]agentCard, 0, len(results))}
	for _, res := range results {
		res.card.Platforms = nil // summary only
		resp.Agents = append(resp.Agents, res.card)
	}
	writeJSON(w, http.StatusOK, resp)
}

// probeAll fetches /health/detailed for every profile in parallel. Each call
// is bounded by the client's own timeout, so one slow profile never delays
// the others; results keep configuration order.
func (h *handlers) probeAll(ctx context.Context) []probeResult {
	results := make([]probeResult, len(h.profiles))
	var wg sync.WaitGroup
	for i, c := range h.profiles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = h.probe(ctx, c)
		}()
	}
	wg.Wait()
	return results
}

func (h *handlers) probe(ctx context.Context, c *hermes.Client) probeResult {
	start := time.Now()
	health, _, err := h.health.Get(ctx, "health/"+c.Name(), c.HealthDetailed)
	card := agentCard{Profile: c.Name(), LatencyMS: time.Since(start).Milliseconds()}
	if err != nil {
		h.log.Warn("profile health probe failed", "profile", c.Name(), "err", err)
		card.Status = statusUnreachable
		var ue *hermes.UpstreamError
		if errors.As(err, &ue) && ue.Kind == hermes.KindUnauthorized {
			card.Status = statusUnauthorized
		}
		_, body := mapError(err, "not_found")
		card.Error = &body
		return probeResult{card: card}
	}
	card.Status = statusDegraded
	if health.Status == "ok" && health.Readiness.Status == "ok" {
		card.Status = statusHealthy
	}
	card.Platforms = platformsFor(c.Name(), health.Platforms)
	return probeResult{card: card, health: health}
}

// platformsFor extracts this profile's entries from the gateway-wide map.
// Named profiles are keyed "<profile>:<platform>"; the default profile's
// entries carry no prefix. The api_server entry is gateway-level and omitted.
func platformsFor(profile string, all map[string]hermes.PlatformState) map[string]hermes.PlatformState {
	out := map[string]hermes.PlatformState{}
	for key, st := range all {
		owner, platform, prefixed := strings.Cut(key, ":")
		switch {
		case prefixed && owner == profile:
			out[platform] = st
		case !prefixed && profile == "default" && key != "api_server":
			out[key] = st
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func newGatewayBlock(profile string, hd *hermes.HealthDetailed) *gatewayBlock {
	ch := hd.Readiness.Checks
	return &gatewayBlock{
		SourceProfile:      profile,
		Version:            hd.Version,
		State:              hd.GatewayState,
		Readiness:          hd.Readiness.Status,
		ActiveAgents:       hd.ActiveAgents,
		ActiveAPIRuns:      ch.BackgroundQueues.ActiveAPIRuns,
		ActiveDelegations:  ch.BackgroundQueues.ActiveDelegations,
		ProcessCompletions: ch.BackgroundQueues.ProcessCompletions,
		Busy:               hd.GatewayBusy,
		Disk:               ch.Disk,
		Checks: map[string]string{
			"state_db":          ch.StateDB.Status,
			"session_store":     ch.SessionStore.Status,
			"config":            ch.Config.Status,
			"model":             ch.Model.Status,
			"disk":              ch.Disk.Status,
			"gateway":           ch.Gateway.Status,
			"background_queues": ch.BackgroundQueues.Status,
		},
		UpdatedAt: hd.UpdatedAt,
	}
}

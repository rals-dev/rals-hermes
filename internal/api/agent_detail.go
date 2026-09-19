package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// agentDetailResponse is GET /api/agents/{profile}: the profile's card plus
// the static facts the agent-detail page shows. Sections that fail on
// upstream become null and are listed in warnings, so one broken endpoint
// (like /v1/skills on 0.21.2) never blanks the page.
type agentDetailResponse struct {
	agentCard
	GeneratedAt  time.Time        `json:"generated_at"`
	Model        string           `json:"model,omitempty"`
	Models       []string         `json:"models"`
	Capabilities json.RawMessage  `json:"capabilities"`
	Toolsets     []hermes.Toolset `json:"toolsets"`
	Skills       json.RawMessage  `json:"skills"`
	Warnings     []string         `json:"warnings"`
}

// profileFromPath resolves {profile} to a client or writes 404.
func (h *handlers) profileFromPath(w http.ResponseWriter, r *http.Request) (*hermes.Client, bool) {
	c, ok := h.byName[r.PathValue("profile")]
	if !ok {
		writeMappedError(w, ErrProfileNotFound, "not_found")
		return nil, false
	}
	return c, true
}

func (h *handlers) agentDetail(w http.ResponseWriter, r *http.Request) {
	c, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	ctx := r.Context()

	// Health decides whether the profile is usable at all; the remaining
	// sections are fetched in parallel and are individually optional.
	probe := h.probe(ctx, c)
	if probe.health == nil {
		status, body := mapError(probe.err, "not_found")
		writeJSON(w, status, body)
		return
	}

	resp := agentDetailResponse{agentCard: probe.card, GeneratedAt: time.Now().UTC(), Warnings: []string{}}
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	warn := func(name string, err error) {
		h.log.Warn("agent detail section failed", "profile", c.Name(), "section", name, "err", err)
		mu.Lock()
		resp.Warnings = append(resp.Warnings, name+"_unavailable")
		mu.Unlock()
	}
	wg.Add(4)
	go func() {
		defer wg.Done()
		caps, err := c.Capabilities(ctx)
		if err != nil {
			warn("capabilities", err)
			return
		}
		mu.Lock()
		resp.Capabilities, resp.Model = caps.Raw, caps.Model
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		ts, err := c.Toolsets(ctx)
		if err != nil {
			warn("toolsets", err)
			return
		}
		mu.Lock()
		resp.Toolsets = ts.Data
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		ml, err := c.Models(ctx)
		if err != nil {
			warn("models", err)
			return
		}
		names := make([]string, 0, len(ml.Data))
		for _, m := range ml.Data {
			names = append(names, m.ID)
		}
		mu.Lock()
		resp.Models = names
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		sk, err := c.Skills(ctx)
		if err != nil {
			warn("skills", err)
			return
		}
		mu.Lock()
		resp.Skills = sk
		mu.Unlock()
	}()
	wg.Wait()

	if resp.Models == nil {
		resp.Models = []string{}
	}
	if resp.Toolsets == nil {
		resp.Toolsets = []hermes.Toolset{}
	}
	writeJSON(w, http.StatusOK, resp)
}

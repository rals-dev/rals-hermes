package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
	"github.com/rals-dev/rals-hermes/internal/view"
)

// usageWindow bounds how far back /api/usage sums session tokens and cost.
// Hermes has no date filter on GET /api/sessions and no known retention
// limit (T-003 § 3), so summing "everything" would mean paginating an
// unbounded, ever-growing history on every cache miss. A rolling window
// keeps the aggregation bounded by recent activity instead — see ADR-021.
const usageWindow = 24 * time.Hour

// usagePageLimit and usageMaxPages bound how many sessions one profile's
// aggregation will ever fetch in a single request, in case last_active data
// is missing, unsorted, or an upstream bug means has_more never turns
// false — the window cutoff below is the normal exit, this is the backstop.
const (
	usagePageLimit = 100
	usageMaxPages  = 10 // 1,000 sessions/profile, worst case
)

// usageCacheTTL is longer than the general CacheTTL: this aggregation can
// mean several upstream calls per profile, so it's refreshed less eagerly
// than a single health probe.
const usageCacheTTL = 60 * time.Second

// usageProfile is one profile's token/cost totals for the window.
type usageProfile struct {
	Profile          string     `json:"profile"`
	SessionCount     int        `json:"session_count"`
	Usage            view.Usage `json:"usage"`
	EstimatedCostUSD *float64   `json:"estimated_cost_usd"`
	ActualCostUSD    *float64   `json:"actual_cost_usd"`
	// Truncated is true if usageMaxPages was hit before a session aged out
	// of the window — the totals are a lower bound, not the whole window.
	Truncated bool `json:"truncated"`
}

type usageResponse struct {
	GeneratedAt time.Time      `json:"generated_at"`
	WindowHours int            `json:"window_hours"`
	Profiles    []usageProfile `json:"profiles"`
	Errors      []profileError `json:"errors"`
}

// usage aggregates GET /api/usage across every profile: token and cost
// totals over the trailing usageWindow, on the fly, with no persisted
// state (ADR-007, ADR-021). A dead profile contributes an entry to errors
// instead of failing the response, matching jobs/overview.
func (h *handlers) usage(w http.ResponseWriter, r *http.Request) {
	type result struct {
		profile string
		summary usageProfile
		err     error
	}
	ctx := r.Context()
	cutoff := h.now().Add(-usageWindow)
	results := make([]result, len(h.profiles))
	var wg sync.WaitGroup
	for i, c := range h.profiles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			summary, _, err := h.usageCache.Get(ctx, "usage/"+c.Name(), func(ctx context.Context) (usageProfile, error) {
				return sumUsage(ctx, c, cutoff)
			})
			results[i] = result{profile: c.Name(), summary: summary, err: err}
		}()
	}
	wg.Wait()

	resp := usageResponse{
		GeneratedAt: h.now().UTC(), WindowHours: int(usageWindow.Hours()),
		Profiles: make([]usageProfile, 0, len(results)), Errors: []profileError{},
	}
	for _, res := range results {
		if res.err != nil {
			h.log.Warn("usage fetch failed", "profile", res.profile, "err", res.err)
			_, body := mapError(res.err, "not_found")
			resp.Errors = append(resp.Errors, profileError{Profile: res.profile, Error: body})
			continue
		}
		resp.Profiles = append(resp.Profiles, res.summary)
	}
	writeJSON(w, http.StatusOK, resp)
}

// sumUsage pages through one profile's sessions, newest first (Hermes
// orders by last_active descending — T-003 § 3), summing token and cost
// fields for sessions inside the window. The first session older than
// cutoff ends the scan without requesting another page.
func sumUsage(ctx context.Context, c *hermes.Client, cutoff time.Time) (usageProfile, error) {
	out := usageProfile{Profile: c.Name()}
	var estSum, actSum float64
	var sawEst, sawAct bool

	offset := 0
	for page := 0; page < usageMaxPages; page++ {
		list, err := c.Sessions(ctx, hermes.SessionsQuery{Limit: usagePageLimit, Offset: offset})
		if err != nil {
			return usageProfile{}, err
		}
		agedOut := false
		for _, s := range list.Data {
			if s.LastActive.Time().Before(cutoff) {
				agedOut = true
				break
			}
			out.SessionCount++
			out.Usage.InputTokens += s.InputTokens
			out.Usage.OutputTokens += s.OutputTokens
			out.Usage.CacheReadTokens += s.CacheReadTokens
			out.Usage.CacheWriteTokens += s.CacheWriteTokens
			out.Usage.ReasoningTokens += s.ReasoningTokens
			if s.EstimatedCostUSD != nil {
				estSum += *s.EstimatedCostUSD
				sawEst = true
			}
			if s.ActualCostUSD != nil {
				actSum += *s.ActualCostUSD
				sawAct = true
			}
		}
		if agedOut || !list.HasMore || len(list.Data) == 0 {
			return finishUsage(out, estSum, actSum, sawEst, sawAct, false), nil
		}
		offset += usagePageLimit
	}
	return finishUsage(out, estSum, actSum, sawEst, sawAct, true), nil
}

func finishUsage(out usageProfile, estSum, actSum float64, sawEst, sawAct, truncated bool) usageProfile {
	if sawEst {
		out.EstimatedCostUSD = &estSum
	}
	if sawAct {
		out.ActualCostUSD = &actSum
	}
	out.Truncated = truncated
	return out
}

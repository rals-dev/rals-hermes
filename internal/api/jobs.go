package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// jobView is a scheduled job tagged with the profile it belongs to.
type jobView struct {
	Profile string `json:"profile"`
	hermes.Job
}

type profileError struct {
	Profile string    `json:"profile"`
	Error   errorBody `json:"error"`
}

type jobsResponse struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Jobs        []jobView      `json:"jobs"`
	Errors      []profileError `json:"errors"`
}

// jobs aggregates GET /api/jobs across every profile (T-206). A dead
// profile contributes an entry to errors instead of failing the response.
func (h *handlers) jobs(w http.ResponseWriter, r *http.Request) {
	type result struct {
		profile string
		list    *hermes.JobList
		err     error
	}
	results := make([]result, len(h.profiles))
	var wg sync.WaitGroup
	for i, c := range h.profiles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			list, _, err := h.jobsCache.Get(r.Context(), "jobs/"+c.Name(), c.Jobs)
			results[i] = result{profile: c.Name(), list: list, err: err}
		}()
	}
	wg.Wait()

	resp := jobsResponse{GeneratedAt: time.Now().UTC(), Jobs: []jobView{}, Errors: []profileError{}}
	for _, res := range results {
		if res.err != nil {
			h.log.Warn("jobs fetch failed", "profile", res.profile, "err", res.err)
			_, body := mapError(res.err, "not_found")
			resp.Errors = append(resp.Errors, profileError{Profile: res.profile, Error: body})
			continue
		}
		for _, j := range res.list.Jobs {
			resp.Jobs = append(resp.Jobs, jobView{Profile: res.profile, Job: j})
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

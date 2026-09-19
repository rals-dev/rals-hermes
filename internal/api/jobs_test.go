package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestJobs_AggregatesAcrossProfilesAndToleratesDeadOnes(t *testing.T) {
	d := newTestDeps(t, 300*time.Millisecond,
		profileSpec{"default", hermesFixtureMux(t, "default", nil)},
		profileSpec{"coder-agent", hermesFixtureMux(t, "coder-agent", nil)},
		profileSpec{"tester-agent", nil},
	)
	rec := authedGet(t, NewHandler(d), "/api/jobs")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Jobs []struct {
			Profile  string `json:"profile"`
			ID       string `json:"id"`
			Name     string `json:"name"`
			Schedule struct {
				Expr string `json:"expr"`
			} `json:"schedule"`
			State      string  `json:"state"`
			LastStatus *string `json:"last_status"`
		} `json:"jobs"`
		Errors []struct {
			Profile string    `json:"profile"`
			Error   errorBody `json:"error"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Jobs) == 0 {
		t.Fatal("no jobs aggregated")
	}
	for _, j := range body.Jobs {
		if j.Profile == "" || j.ID == "" || j.Schedule.Expr == "" {
			t.Errorf("job lacks profile/id/schedule: %+v", j)
		}
	}
	if len(body.Errors) != 1 || body.Errors[0].Profile != "tester-agent" || body.Errors[0].Error.Code != "upstream_unreachable" {
		t.Errorf("errors = %+v, want one upstream_unreachable for tester-agent", body.Errors)
	}
}

func TestJobs_EmptyWhenNoProfileHasJobs(t *testing.T) {
	d := newTestDeps(t, time.Second, profileSpec{"product-agent", hermesFixtureMux(t, "product-agent", nil)})
	rec := authedGet(t, NewHandler(d), "/api/jobs")
	var body struct {
		Jobs   []any `json:"jobs"`
		Errors []any `json:"errors"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusOK || body.Jobs == nil || len(body.Jobs) != 0 || body.Errors == nil {
		t.Errorf("status=%d body=%s (jobs and errors must be [] not null)", rec.Code, rec.Body)
	}
}

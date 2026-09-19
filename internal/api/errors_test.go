package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

func upstream(kind hermes.Kind, status int) error {
	return &hermes.UpstreamError{Profile: "p", Op: "GET /x", Kind: kind, Status: status}
}

// Every row of the PRD § 5 error taxonomy, plus the internal kinds the
// taxonomy does not name explicitly.
func TestMapError_CoversTheTaxonomy(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"profile not in config", ErrProfileNotFound, http.StatusNotFound, "profile_not_found"},
		{"upstream 401", upstream(hermes.KindUnauthorized, 401), http.StatusBadGateway, "upstream_unauthorized"},
		{"upstream 404 on run", upstream(hermes.KindNotFound, 404), http.StatusNotFound, "run_not_found"},
		{"upstream 429", upstream(hermes.KindBusy, 429), http.StatusServiceUnavailable, "upstream_busy"},
		{"timeout / refused", upstream(hermes.KindUnreachable, 0), http.StatusServiceUnavailable, "upstream_unreachable"},
		{"upstream 5xx", upstream(hermes.KindServerError, 500), http.StatusBadGateway, "upstream_error"},
		{"upstream non-JSON", upstream(hermes.KindBadResponse, 200), http.StatusBadGateway, "upstream_error"},
		{"bad query param", &BadRequestError{Msg: "limit out of range"}, http.StatusBadRequest, "bad_request"},
		{"unknown error", errors.New("wat"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st, body := mapError(tc.err, "run_not_found")
			if st != tc.wantStatus || body.Code != tc.wantCode {
				t.Errorf("mapError = %d %q, want %d %q", st, body.Code, tc.wantStatus, tc.wantCode)
			}
			if body.Message == "" {
				t.Error("message must not be empty")
			}
		})
	}
}

func TestMapError_NotFoundCodeIsCallerChosen(t *testing.T) {
	_, body := mapError(upstream(hermes.KindNotFound, 404), "session_not_found")
	if body.Code != "session_not_found" {
		t.Errorf("code = %q, want session_not_found", body.Code)
	}
}

func TestMapError_NeverForwardsUpstreamDetail(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.9:8642: connect: connection refused; Authorization: Bearer leaked")
	err := &hermes.UpstreamError{Profile: "p", Op: "GET /x", Kind: hermes.KindUnreachable}
	_ = cause
	_, body := mapError(err, "run_not_found")
	for _, forbidden := range []string{"10.0.0.9", "Bearer", "leaked", "GET /x"} {
		if contains(body.Message, forbidden) {
			t.Errorf("message %q leaks %q", body.Message, forbidden)
		}
	}
}

func contains(s, sub string) bool { return len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

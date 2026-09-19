package api

import (
	"errors"
	"net/http"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// ErrProfileNotFound is returned when the path names a profile that is not in
// the configuration.
var ErrProfileNotFound = errors.New("profile not found")

// BadRequestError rejects a client-supplied parameter. Msg is safe to show.
type BadRequestError struct{ Msg string }

func (e *BadRequestError) Error() string { return "bad request: " + e.Msg }

// mapError translates any error into the BFF's HTTP status and uniform error
// body (PRD § 5, "Error taxonomy"). notFoundCode names the resource for an
// upstream 404 — "run_not_found", "session_not_found" — because the same
// upstream kind means different things on different routes.
//
// Messages are fixed strings. Upstream text, URLs and operations never pass
// through here; they stay in server logs.
func mapError(err error, notFoundCode string) (int, errorBody) {
	var br *BadRequestError
	if errors.As(err, &br) {
		return http.StatusBadRequest, errorBody{Code: "bad_request", Message: br.Msg}
	}
	if errors.Is(err, ErrProfileNotFound) {
		return http.StatusNotFound, errorBody{Code: "profile_not_found", Message: "no such profile is configured"}
	}
	var ue *hermes.UpstreamError
	if errors.As(err, &ue) {
		switch ue.Kind {
		case hermes.KindUnauthorized:
			return http.StatusBadGateway, errorBody{Code: "upstream_unauthorized", Message: "the profile key was rejected by Hermes"}
		case hermes.KindNotFound:
			return http.StatusNotFound, errorBody{Code: notFoundCode, Message: "not found on this profile"}
		case hermes.KindBusy:
			return http.StatusServiceUnavailable, errorBody{Code: "upstream_busy", Message: "Hermes is at its concurrency limit"}
		case hermes.KindUnreachable:
			return http.StatusServiceUnavailable, errorBody{Code: "upstream_unreachable", Message: "Hermes did not answer in time"}
		case hermes.KindServerError, hermes.KindBadResponse:
			return http.StatusBadGateway, errorBody{Code: "upstream_error", Message: "Hermes returned an unexpected response"}
		}
	}
	return http.StatusInternalServerError, errorBody{Code: "internal_error", Message: "internal error"}
}

// writeMappedError logs the full error server-side and sends the sanitised
// body to the client.
func writeMappedError(w http.ResponseWriter, err error, notFoundCode string) {
	status, body := mapError(err, notFoundCode)
	writeJSON(w, status, body)
}

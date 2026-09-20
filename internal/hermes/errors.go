package hermes

import (
	"fmt"
	"net/http"
)

// Kind classifies an upstream failure independently of transport details so
// the API layer can map it to the BFF error taxonomy (PRD § 5).
type Kind int

const (
	// KindUnreachable covers dial failures, resets and deadline exceeded.
	KindUnreachable Kind = iota + 1
	// KindUnauthorized covers 401 and 403: wrong or missing profile key.
	KindUnauthorized
	// KindNotFound covers 404, including a run_id owned by another profile.
	KindNotFound
	// KindBusy covers 429: the profile hit max_concurrent_runs.
	KindBusy
	// KindServerError covers any other 4xx/5xx.
	KindServerError
	// KindBadResponse covers a 2xx whose body is not the expected JSON.
	KindBadResponse
)

func (k Kind) String() string {
	switch k {
	case KindUnreachable:
		return "unreachable"
	case KindUnauthorized:
		return "unauthorized"
	case KindNotFound:
		return "not_found"
	case KindBusy:
		return "busy"
	case KindServerError:
		return "server_error"
	case KindBadResponse:
		return "bad_response"
	default:
		return fmt.Sprintf("kind(%d)", int(k))
	}
}

// UpstreamError is the only error type the client returns for failed calls.
// Its text is deliberately terse: it names the profile, the operation and the
// classification, never the request URL, headers, or the upstream body —
// those may carry credentials or free text. The wrapped cause is available
// through errors.Unwrap for logging at debug level.
type UpstreamError struct {
	Profile string
	Op      string // e.g. "GET /health/detailed"
	Kind    Kind
	Status  int // HTTP status when one was received, else 0
	cause   error
}

func (e *UpstreamError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("hermes %s: %s: %s (http %d)", e.Profile, e.Op, e.Kind, e.Status)
	}
	return fmt.Sprintf("hermes %s: %s: %s", e.Profile, e.Op, e.Kind)
}

// Unwrap exposes the transport-level cause. Callers must not forward it to
// clients; it may contain the upstream URL.
func (e *UpstreamError) Unwrap() error { return e.cause }

func kindForStatus(status int) Kind {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return KindUnauthorized
	case http.StatusNotFound:
		return KindNotFound
	case http.StatusTooManyRequests:
		return KindBusy
	default:
		return KindServerError
	}
}

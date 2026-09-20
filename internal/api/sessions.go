package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/rals-dev/rals-hermes/internal/hermes"
	"github.com/rals-dev/rals-hermes/internal/view"
)

// Query parameter bounds. Hermes accepts more, but the dashboard never needs
// it and a bound keeps one click from asking for thousands of rows.
const (
	defaultLimit = 20
	maxLimit     = 200
)

var sourcePattern = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

type sessionListResponse struct {
	Profile string         `json:"profile"`
	Data    []view.Session `json:"data"`
	HasMore bool           `json:"has_more"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

type sessionDetailResponse struct {
	Profile  string              `json:"profile"`
	Session  view.Session        `json:"session"`
	Messages *hermes.MessageList `json:"messages"`
}

type runResponse struct {
	Profile string          `json:"profile"`
	RunID   string          `json:"run_id"`
	Status  string          `json:"status"`
	Run     json.RawMessage `json:"run"`
}

// parsePaging validates limit/offset and returns them with defaults applied.
func parsePaging(r *http.Request) (limit, offset int, err error) {
	limit = defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, perr := strconv.Atoi(v)
		if perr != nil || n < 1 || n > maxLimit {
			return 0, 0, &BadRequestError{Msg: "limit must be an integer between 1 and " + strconv.Itoa(maxLimit)}
		}
		limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, perr := strconv.Atoi(v)
		if perr != nil || n < 0 {
			return 0, 0, &BadRequestError{Msg: "offset must be a non-negative integer"}
		}
		offset = n
	}
	return limit, offset, nil
}

func (h *handlers) listSessions(w http.ResponseWriter, r *http.Request) {
	c, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	limit, offset, err := parsePaging(r)
	if err != nil {
		writeMappedError(w, err, "not_found")
		return
	}
	source := r.URL.Query().Get("source")
	if source != "" && !sourcePattern.MatchString(source) {
		writeMappedError(w, &BadRequestError{Msg: "source must match [a-z0-9_-]{1,32}"}, "not_found")
		return
	}
	list, err := c.Sessions(r.Context(), hermes.SessionsQuery{Limit: limit, Offset: offset, Source: source, IncludeChildren: true})
	if err != nil {
		writeMappedError(w, err, "not_found")
		return
	}
	resp := sessionListResponse{Profile: c.Name(), Data: make([]view.Session, 0, len(list.Data)), HasMore: list.HasMore, Limit: list.Limit, Offset: list.Offset}
	for _, s := range list.Data {
		resp.Data = append(resp.Data, view.FromSession(s))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handlers) sessionDetail(w http.ResponseWriter, r *http.Request) {
	c, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	limit, offset, err := parsePaging(r)
	if err != nil {
		writeMappedError(w, err, "session_not_found")
		return
	}
	id := r.PathValue("id")
	detail, err := c.Session(r.Context(), id)
	if err != nil {
		writeMappedError(w, err, "session_not_found")
		return
	}
	msgs, err := c.Messages(r.Context(), id, hermes.MessagesQuery{Limit: limit, Offset: offset, Order: r.URL.Query().Get("order")})
	if err != nil {
		writeMappedError(w, err, "session_not_found")
		return
	}
	writeJSON(w, http.StatusOK, sessionDetailResponse{Profile: c.Name(), Session: view.FromSession(detail.Session), Messages: msgs})
}

func (h *handlers) run(w http.ResponseWriter, r *http.Request) {
	c, ok := h.profileFromPath(w, r)
	if !ok {
		return
	}
	id := r.PathValue("run_id")
	run, err := c.Run(r.Context(), id)
	if err != nil {
		writeMappedError(w, err, "run_not_found")
		return
	}
	writeJSON(w, http.StatusOK, runResponse{Profile: c.Name(), RunID: id, Status: run.Status, Run: run.Raw})
}

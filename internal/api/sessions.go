package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// Query parameter bounds. Hermes accepts more, but the dashboard never needs
// it and a bound keeps one click from asking for thousands of rows.
const (
	defaultLimit = 20
	maxLimit     = 200
)

var sourcePattern = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

// sessionView is the dashboard's projection of a Hermes session: usage is
// grouped, timestamps are RFC 3339, and "open" is derived so the UI never
// re-implements the end_reason rule.
type sessionView struct {
	ID               string     `json:"id"`
	Source           string     `json:"source"`
	Model            string     `json:"model"`
	Title            *string    `json:"title"`
	Preview          *string    `json:"preview"`
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	EndReason        *string    `json:"end_reason"`
	Open             bool       `json:"open"`
	LastActive       time.Time  `json:"last_active"`
	MessageCount     int        `json:"message_count"`
	ToolCallCount    int        `json:"tool_call_count"`
	APICallCount     int        `json:"api_call_count"`
	ParentSessionID  *string    `json:"parent_session_id"`
	Usage            usageView  `json:"usage"`
	EstimatedCostUSD *float64   `json:"estimated_cost_usd"`
	ActualCostUSD    *float64   `json:"actual_cost_usd"`
}

type usageView struct {
	InputTokens      int64 `json:"input_tokens"`
	OutputTokens     int64 `json:"output_tokens"`
	CacheReadTokens  int64 `json:"cache_read_tokens"`
	CacheWriteTokens int64 `json:"cache_write_tokens"`
	ReasoningTokens  int64 `json:"reasoning_tokens"`
}

func toSessionView(s hermes.Session) sessionView {
	v := sessionView{
		ID: s.ID, Source: s.Source, Model: s.Model, Title: s.Title, Preview: s.Preview,
		StartedAt: s.StartedAt.Time(), EndReason: s.EndReason, Open: s.EndReason == nil && s.EndedAt == nil,
		LastActive: s.LastActive.Time(), MessageCount: s.MessageCount, ToolCallCount: s.ToolCallCount,
		APICallCount: s.APICallCount, ParentSessionID: s.ParentSessionID,
		Usage: usageView{
			InputTokens: s.InputTokens, OutputTokens: s.OutputTokens, CacheReadTokens: s.CacheReadTokens,
			CacheWriteTokens: s.CacheWriteTokens, ReasoningTokens: s.ReasoningTokens,
		},
		EstimatedCostUSD: s.EstimatedCostUSD, ActualCostUSD: s.ActualCostUSD,
	}
	if s.EndedAt != nil {
		t := s.EndedAt.Time()
		v.EndedAt = &t
	}
	return v
}

type sessionListResponse struct {
	Profile string        `json:"profile"`
	Data    []sessionView `json:"data"`
	HasMore bool          `json:"has_more"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
}

type sessionDetailResponse struct {
	Profile  string              `json:"profile"`
	Session  sessionView         `json:"session"`
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
	resp := sessionListResponse{Profile: c.Name(), Data: make([]sessionView, 0, len(list.Data)), HasMore: list.HasMore, Limit: list.Limit, Offset: list.Offset}
	for _, s := range list.Data {
		resp.Data = append(resp.Data, toSessionView(s))
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
	writeJSON(w, http.StatusOK, sessionDetailResponse{Profile: c.Name(), Session: toSessionView(detail.Session), Messages: msgs})
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

package hermes

import (
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// UnixTime decodes Hermes' float "seconds since epoch" timestamps.
type UnixTime float64

// Time converts to time.Time (UTC).
func (u UnixTime) Time() time.Time {
	sec, frac := math.Modf(float64(u))
	return time.Unix(int64(sec), int64(frac*1e9)).UTC()
}

// MarshalJSON renders the timestamp as RFC 3339 for the BFF's own API.
func (u UnixTime) MarshalJSON() ([]byte, error) {
	if u == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(u.Time().Format(time.RFC3339Nano))
}

// UnmarshalJSON accepts a JSON number.
func (u *UnixTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*u = 0
		return nil
	}
	f, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	*u = UnixTime(f)
	return nil
}

// HealthDetailed mirrors GET /health/detailed (gateway-wide; see
// docs/t003-findings.md § 2).
type HealthDetailed struct {
	Status           string                   `json:"status"`
	Readiness        Readiness                `json:"readiness"`
	Platform         string                   `json:"platform"`
	Version          string                   `json:"version"`
	GatewayState     string                   `json:"gateway_state"`
	Platforms        map[string]PlatformState `json:"platforms"`
	ActiveAgents     int                      `json:"active_agents"`
	GatewayBusy      bool                     `json:"gateway_busy"`
	GatewayDrainable bool                     `json:"gateway_drainable"`
	ExitReason       *string                  `json:"exit_reason"`
	UpdatedAt        time.Time                `json:"updated_at"`
	PID              int                      `json:"pid"`
}

// Readiness groups the individual checks.
type Readiness struct {
	Status string `json:"status"`
	Checks Checks `json:"checks"`
}

// Checks are the named readiness checks observed on Hermes 0.21.2.
type Checks struct {
	StateDB          Check        `json:"state_db"`
	SessionStore     Check        `json:"session_store"`
	Config           Check        `json:"config"`
	Model            Check        `json:"model"`
	Disk             DiskCheck    `json:"disk"`
	Gateway          GatewayCheck `json:"gateway"`
	BackgroundQueues QueueCheck   `json:"background_queues"`
}

// Check is a plain status check.
type Check struct {
	Status string `json:"status"`
}

// DiskCheck adds disk usage to Check.
type DiskCheck struct {
	Status      string  `json:"status"`
	UsedPercent float64 `json:"used_percent"`
	FreeBytes   int64   `json:"free_bytes"`
}

// GatewayCheck adds gateway state to Check.
type GatewayCheck struct {
	Status             string `json:"status"`
	State              string `json:"state"`
	ConnectedPlatforms int    `json:"connected_platforms"`
	Platforms          int    `json:"platforms"`
}

// QueueCheck carries the gateway-wide work counters.
type QueueCheck struct {
	Status             string `json:"status"`
	ActiveAPIRuns      int    `json:"active_api_runs"`
	ProcessCompletions int    `json:"process_completions"`
	ActiveDelegations  int    `json:"active_delegations"`
}

// PlatformState is one entry of HealthDetailed.Platforms, keyed by
// "<profile>:<platform>" (or just "<platform>" for the default profile).
type PlatformState struct {
	State          string    `json:"state"`
	ErrorCode      *string   `json:"error_code"`
	ErrorMessage   *string   `json:"error_message"`
	UpdatedAt      time.Time `json:"updated_at"`
	NeedsAttention *bool     `json:"needs_attention,omitempty"`
}

// Capabilities mirrors GET /v1/capabilities. Only the parts the BFF gates on
// are typed; the rest is kept raw for the agent-detail page.
type Capabilities struct {
	Platform string          `json:"platform"`
	Model    string          `json:"model"`
	Features map[string]any  `json:"features"`
	Raw      json.RawMessage `json:"-"`
}

// SessionList mirrors GET /api/sessions.
type SessionList struct {
	Object  string    `json:"object"`
	Data    []Session `json:"data"`
	HasMore bool      `json:"has_more"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
}

// Session mirrors one entry of /api/sessions (docs/t003-findings.md § 3).
type Session struct {
	ID               string    `json:"id"`
	Source           string    `json:"source"`
	UserID           *string   `json:"user_id"`
	Model            string    `json:"model"`
	Title            *string   `json:"title"`
	StartedAt        UnixTime  `json:"started_at"`
	EndedAt          *UnixTime `json:"ended_at"`
	EndReason        *string   `json:"end_reason"`
	MessageCount     int       `json:"message_count"`
	ToolCallCount    int       `json:"tool_call_count"`
	InputTokens      int64     `json:"input_tokens"`
	OutputTokens     int64     `json:"output_tokens"`
	CacheReadTokens  int64     `json:"cache_read_tokens"`
	CacheWriteTokens int64     `json:"cache_write_tokens"`
	ReasoningTokens  int64     `json:"reasoning_tokens"`
	EstimatedCostUSD *float64  `json:"estimated_cost_usd"`
	ActualCostUSD    *float64  `json:"actual_cost_usd"`
	APICallCount     int       `json:"api_call_count"`
	ParentSessionID  *string   `json:"parent_session_id"`
	LastActive       UnixTime  `json:"last_active"`
	Preview          *string   `json:"preview"`
	Pinned           bool      `json:"pinned"`
	Archived         bool      `json:"archived"`
	Hidden           bool      `json:"hidden"`
}

// SessionDetail mirrors GET /api/sessions/{id}.
type SessionDetail struct {
	Object  string  `json:"object"`
	Session Session `json:"session"`
}

// MessageList mirrors GET /api/sessions/{id}/messages.
type MessageList struct {
	Object     string     `json:"object"`
	SessionID  string     `json:"session_id"`
	Pagination Pagination `json:"pagination"`
	Data       []Message  `json:"data"`
}

// Pagination is the messages endpoint's paging block.
type Pagination struct {
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Order    string `json:"order"`
	Returned int    `json:"returned"`
}

// Message is one session message. Tool calls appear as `tool_calls` on an
// assistant message; their results as role "tool" with tool_call_id.
type Message struct {
	ID               int64           `json:"id"`
	SessionID        string          `json:"session_id"`
	Role             string          `json:"role"`
	Content          *string         `json:"content"`
	ToolCallID       string          `json:"tool_call_id"`
	ToolCalls        []ToolCall      `json:"tool_calls"`
	ToolName         string          `json:"tool_name"`
	Timestamp        UnixTime        `json:"timestamp"`
	TokenCount       *int            `json:"token_count"`
	FinishReason     *string         `json:"finish_reason"`
	DisplayKind      *string         `json:"display_kind"`
	Reasoning        json.RawMessage `json:"reasoning,omitempty"`
	ReasoningContent json.RawMessage `json:"reasoning_content,omitempty"`
}

// ToolCall is one entry of Message.ToolCalls (OpenAI-style shape).
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Run mirrors GET /v1/runs/{run_id}. Kept loosely typed until a live run
// fixture exists; Status is the field the UI keys on.
type Run struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Raw    json.RawMessage `json:"-"`
}

// JobList mirrors GET /api/jobs.
type JobList struct {
	Jobs []Job `json:"jobs"`
}

// Job mirrors one scheduled job (fields the dashboard shows).
type Job struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Schedule         JobSchedule     `json:"schedule"`
	ScheduleDisplay  string          `json:"schedule_display"`
	Enabled          bool            `json:"enabled"`
	State            string          `json:"state"`
	CreatedAt        *time.Time      `json:"created_at"`
	NextRunAt        *time.Time      `json:"next_run_at"`
	LastRunAt        *time.Time      `json:"last_run_at"`
	LastStatus       *string         `json:"last_status"`
	LastError        *string         `json:"last_error"`
	FailureStreak    int             `json:"failure_streak"`
	Deliver          *string         `json:"deliver"`
	ModelSnapshot    *string         `json:"model_snapshot"`
	ProviderSnapshot *string         `json:"provider_snapshot"`
	LatestExecution  json.RawMessage `json:"latest_execution,omitempty"`
}

// JobSchedule is the schedule block of a job.
type JobSchedule struct {
	Kind    string `json:"kind"`
	Expr    string `json:"expr"`
	Display string `json:"display"`
}

// ToolsetList mirrors GET /v1/toolsets.
type ToolsetList struct {
	Object string    `json:"object"`
	Data   []Toolset `json:"data"`
}

// Toolset is one configured toolset with its concrete tools.
type Toolset struct {
	Name       string   `json:"name"`
	Label      string   `json:"label"`
	Enabled    bool     `json:"enabled"`
	Configured bool     `json:"configured"`
	Tools      []string `json:"tools"`
}

// ModelList mirrors GET /v1/models (OpenAI shape).
type ModelList struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

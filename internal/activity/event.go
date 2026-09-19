// Package activity derives a live activity feed from Hermes sessions.
//
// Hermes has no endpoint that lists runs, and work started from Telegram or
// by delegation never appears in /v1/runs (docs/t003-findings.md). Sessions
// and their messages are the only cross-source signal, so each profile gets
// a poller that tails recent sessions incrementally and turns new messages
// into events. Pollers are lazy: they run only while someone is subscribed
// (ADR-018).
package activity

import (
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// Event types emitted on a profile's stream.
const (
	// TypeSnapshot describes a tracked session's current state. Sent for
	// every tracked session when a poller starts, and again whenever a
	// session's counters change. No message history is replayed.
	TypeSnapshot = "session.snapshot"
	// TypeSessionStarted marks a session first seen while the poller runs.
	TypeSessionStarted = "session.started"
	// TypeSessionEnded marks a tracked session that gained an end_reason.
	TypeSessionEnded = "session.ended"
	// TypeSubagentStart / TypeSubagentComplete are the child-session variants
	// of started/ended: the session carries parent_session_id.
	TypeSubagentStart    = "subagent.start"
	TypeSubagentComplete = "subagent.complete"
	// TypeToolStarted is an assistant message carrying a tool call.
	TypeToolStarted = "tool.started"
	// TypeToolCompleted is a tool-result message.
	TypeToolCompleted = "tool.completed"
	// TypeMessage is a user or assistant text message.
	TypeMessage = "message"
	// TypeError reports an upstream failure; polling continues.
	TypeError = "error"
)

// previewLimit caps free-text previews. Hermes already redacts secrets and
// truncates its own previews at 500 characters; this keeps events small.
const previewLimit = 300

// Event is one item on the feed. Fields not relevant to a type are empty.
type Event struct {
	Type      string    `json:"type"`
	Profile   string    `json:"profile"`
	SessionID string    `json:"session_id,omitempty"`
	At        time.Time `json:"at"`

	// Session is the current session state for snapshot/start/end events.
	Session         *hermes.Session `json:"session,omitempty"`
	ParentSessionID string          `json:"parent_session_id,omitempty"`
	EndReason       string          `json:"end_reason,omitempty"`

	// Tool events.
	Tool    string `json:"tool,omitempty"`
	CallID  string `json:"call_id,omitempty"`
	Preview string `json:"preview,omitempty"`

	// Message events.
	Role      string `json:"role,omitempty"`
	MessageID int64  `json:"message_id,omitempty"`

	// Error events.
	Error *ErrorBody `json:"error,omitempty"`
}

// ErrorBody mirrors the BFF's uniform error shape.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func preview(s string) string {
	if len(s) <= previewLimit {
		return s
	}
	return s[:previewLimit] + "…"
}

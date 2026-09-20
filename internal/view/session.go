// Package view holds the dashboard-facing projections shared by the JSON API
// and the activity feed, so a session looks the same wherever it appears.
package view

import (
	"time"

	"github.com/rals-dev/rals-hermes/internal/hermes"
)

// Session is the dashboard's projection of a Hermes session: usage grouped,
// timestamps as RFC 3339, "open" derived so the UI never re-implements the
// end_reason rule, and no user identifiers.
type Session struct {
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
	Usage            Usage      `json:"usage"`
	EstimatedCostUSD *float64   `json:"estimated_cost_usd"`
	ActualCostUSD    *float64   `json:"actual_cost_usd"`
}

// Usage groups the token counters.
type Usage struct {
	InputTokens      int64 `json:"input_tokens"`
	OutputTokens     int64 `json:"output_tokens"`
	CacheReadTokens  int64 `json:"cache_read_tokens"`
	CacheWriteTokens int64 `json:"cache_write_tokens"`
	ReasoningTokens  int64 `json:"reasoning_tokens"`
}

// FromSession projects a Hermes session.
func FromSession(s hermes.Session) Session {
	v := Session{
		ID: s.ID, Source: s.Source, Model: s.Model, Title: s.Title, Preview: s.Preview,
		StartedAt: s.StartedAt.Time(), EndReason: s.EndReason, Open: s.EndReason == nil && s.EndedAt == nil,
		LastActive: s.LastActive.Time(), MessageCount: s.MessageCount, ToolCallCount: s.ToolCallCount,
		APICallCount: s.APICallCount, ParentSessionID: s.ParentSessionID,
		Usage: Usage{
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

// Package hermes is the HTTP client for one Hermes profile's API server.
//
// One Client per profile, each with its own base URL and bearer key. Every
// call is a single attempt bounded by the profile timeout — there are no
// automatic retries (ADR-017). Failures surface as *UpstreamError whose text
// never includes URLs, headers, or upstream bodies.
package hermes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rals-dev/rals-hermes/internal/config"
)

// maxBodyBytes caps how much of an upstream body is read, so a misbehaving
// upstream cannot exhaust memory.
const maxBodyBytes = 8 << 20

// Options tunes a Client.
type Options struct {
	// Timeout bounds each request end to end. Zero means 2 s.
	Timeout time.Duration
	// Transport overrides the HTTP transport (tests, metrics wrappers).
	Transport http.RoundTripper
	Logger    *slog.Logger
}

// Client talks to a single profile.
type Client struct {
	name    string
	base    *url.URL
	key     config.Secret
	timeout time.Duration
	http    *http.Client
	log     *slog.Logger
}

// NewClient builds a client for profile p. The base URL has already been
// validated and normalised by package config.
func NewClient(p config.Profile, o Options) *Client {
	if o.Timeout <= 0 {
		o.Timeout = 2 * time.Second
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	transport := o.Transport
	if transport == nil {
		transport = &http.Transport{
			Proxy:               nil, // never honour HTTP_PROXY for an internal upstream
			DialContext:         (&net.Dialer{Timeout: o.Timeout}).DialContext,
			MaxIdleConns:        8,
			MaxIdleConnsPerHost: 8,
			IdleConnTimeout:     90 * time.Second,
			ForceAttemptHTTP2:   false,
		}
	}
	base, _ := url.Parse(p.BaseURL) // validated in config
	return &Client{
		name:    p.Name,
		base:    base,
		key:     p.Key,
		timeout: o.Timeout,
		http:    &http.Client{Transport: transport},
		log:     o.Logger.With("profile", p.Name),
	}
}

// Name returns the profile name.
func (c *Client) Name() string { return c.name }

// HealthDetailed calls GET /health/detailed.
func (c *Client) HealthDetailed(ctx context.Context) (*HealthDetailed, error) {
	var out HealthDetailed
	if err := c.getJSON(ctx, "/health/detailed", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Capabilities calls GET /v1/capabilities.
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	var raw json.RawMessage
	if err := c.getJSON(ctx, "/v1/capabilities", nil, &raw); err != nil {
		return nil, err
	}
	var out Capabilities
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, c.wrap("GET /v1/capabilities", KindBadResponse, 200, err)
	}
	out.Raw = raw
	return &out, nil
}

// Models calls GET /v1/models.
func (c *Client) Models(ctx context.Context) (*ModelList, error) {
	var out ModelList
	if err := c.getJSON(ctx, "/v1/models", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Toolsets calls GET /v1/toolsets.
func (c *Client) Toolsets(ctx context.Context) (*ToolsetList, error) {
	var out ToolsetList
	if err := c.getJSON(ctx, "/v1/toolsets", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Skills calls GET /v1/skills and returns the raw body; Hermes 0.21.2
// answers 500 here, so callers treat it as optional.
func (c *Client) Skills(ctx context.Context) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.getJSON(ctx, "/v1/skills", nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// SessionsQuery holds the passthrough parameters of GET /api/sessions.
type SessionsQuery struct {
	Limit           int
	Offset          int
	Source          string
	IncludeChildren bool
}

// Sessions calls GET /api/sessions.
func (c *Client) Sessions(ctx context.Context, q SessionsQuery) (*SessionList, error) {
	v := url.Values{}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	if q.Source != "" {
		v.Set("source", q.Source)
	}
	if q.IncludeChildren {
		v.Set("include_children", "true")
	}
	var out SessionList
	if err := c.getJSON(ctx, "/api/sessions", v, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Session calls GET /api/sessions/{id}.
func (c *Client) Session(ctx context.Context, id string) (*SessionDetail, error) {
	var out SessionDetail
	if err := c.getJSON(ctx, "/api/sessions/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MessagesQuery holds the paging parameters of GET /api/sessions/{id}/messages.
type MessagesQuery struct {
	Limit  int
	Offset int
	Order  string // "oldest" (default) or "newest"
}

// Messages calls GET /api/sessions/{id}/messages.
func (c *Client) Messages(ctx context.Context, id string, q MessagesQuery) (*MessageList, error) {
	v := url.Values{}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	if q.Order != "" {
		v.Set("order", q.Order)
	}
	var out MessageList
	if err := c.getJSON(ctx, "/api/sessions/"+url.PathEscape(id)+"/messages", v, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Run calls GET /v1/runs/{run_id}. A run owned by another profile is a 404
// and therefore KindNotFound.
func (c *Client) Run(ctx context.Context, runID string) (*Run, error) {
	var raw json.RawMessage
	if err := c.getJSON(ctx, "/v1/runs/"+url.PathEscape(runID), nil, &raw); err != nil {
		return nil, err
	}
	var out Run
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, c.wrap("GET /v1/runs/{id}", KindBadResponse, 200, err)
	}
	out.Raw = raw
	return &out, nil
}

// Jobs calls GET /api/jobs.
func (c *Client) Jobs(ctx context.Context) (*JobList, error) {
	var out JobList
	if err := c.getJSON(ctx, "/api/jobs", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// getJSON performs one bounded GET and decodes a 2xx JSON body into out.
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	op := "GET " + path
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	u := *c.base
	u.Path = c.base.Path + path
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return c.wrap(op, KindBadResponse, 0, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.key.Reveal())
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		c.log.Debug("upstream transport error", "op", op, "elapsed", time.Since(start).String(), "err", redactURL(err))
		return c.wrap(op, KindUnreachable, 0, redactURL(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return c.wrap(op, KindUnreachable, resp.StatusCode, err)
	}
	c.log.Debug("upstream response", "op", op, "status", resp.StatusCode, "elapsed", time.Since(start).String(), "bytes", len(body))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// The body is deliberately dropped: it is free text from upstream.
		return c.wrap(op, kindForStatus(resp.StatusCode), resp.StatusCode, nil)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return c.wrap(op, KindBadResponse, resp.StatusCode, err)
	}
	return nil
}

func (c *Client) wrap(op string, kind Kind, status int, cause error) error {
	return &UpstreamError{Profile: c.name, Op: op, Kind: kind, Status: status, cause: cause}
}

// redactURL strips the URL from net/url errors so that wrapped causes carry
// only the operation and the transport reason.
func redactURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return fmt.Errorf("%s: %w", ue.Op, ue.Err)
	}
	return err
}

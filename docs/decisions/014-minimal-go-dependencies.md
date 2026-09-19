# ADR-014: Standard library first; four direct dependencies

Date: 2026-09-19 · Status: Accepted

## Context

The PRD asks for 'no heavy framework'. Each `go.mod` entry is a decision a reviewer will read.

## Decision

Router: `net/http` (Go 1.22+ method/pattern mux). Config: `gopkg.in/yaml.v3`. Concurrency: `golang.org/x/sync` (`errgroup`, `singleflight`). Metrics: `github.com/prometheus/client_golang`. Logging: `log/slog`. Tests: stdlib + `github.com/google/go-cmp`. TTL cache is hand-written (~80 lines) on top of `singleflight`.

## Consequences

- No chi/gin/viper/zap/testify.
- The cache implementation is part of what the portfolio shows.

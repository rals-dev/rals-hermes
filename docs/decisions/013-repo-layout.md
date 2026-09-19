# ADR-013: Monorepo with the Go module at the root

Date: 2026-09-19 · Status: Accepted

## Context

One repository must hold the Go BFF, the Vue frontend, deployment manifests, fixtures, and docs, and read well to a reviewer.

## Decision

Module `github.com/rals-dev/rals-hermes` at the root; `cmd/bff`, `internal/{config,hermes,cache,activity,api,observ}`, `web/`, `deploy/{hermes-dashboard,traefik,alloy,grafana}`, `testdata/fixtures/`, `scripts/`, `docs/`. Everything under `internal/`; nothing is meant to be imported elsewhere.

## Consequences

- `go run ./cmd/bff` works from the root.
- Deployment artefacts are versioned with the code they deploy.

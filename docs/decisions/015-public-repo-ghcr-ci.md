# ADR-015: Public repository; GitHub Actions builds and pushes to GHCR

Date: 2026-09-19 · Status: Accepted

## Context

A portfolio piece should be public with visible CI. Building on the constrained host or on an arm64 workstation is not sustainable.

## Decision

Repository is public. CI runs vet, lint, tests, and builds a `linux/amd64` image pushed to `ghcr.io/rals-dev/rals-hermes`. The host pulls with `docker compose pull`. A manual `docker save | ssh docker load` path is documented as a fallback. All IPs and host-specific values in docs are placeholders; real values live only in `.env` files on the host.

## Consequences

- Fixtures must be redacted before commit.
- No registry credentials are needed on the host for a public package.

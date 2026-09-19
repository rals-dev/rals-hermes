# ADR-003: Local development reaches Hermes through an SSH port-forward with keys in the shell only

Date: 2026-09-19 · Status: Accepted

## Context

The engineer's workstation cannot reach the Hermes API port directly; it is only exposed inside the Docker network. Keys must not leave the host in files.

## Decision

Use `ssh -L 8642:<container-ip>:8642` over Tailscale. Keys are loaded with `read -rs` into the shell environment for the session and never written to disk, committed, or echoed.

## Consequences

- Convenient live verification (SSE, feed) during development.
- Keys transiently exist on the workstation; mitigated by never persisting them and by the isolation that each key only unlocks one profile.

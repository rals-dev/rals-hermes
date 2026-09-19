# ADR-009: Ship logs to Loki from a separate Alloy stack

Date: 2026-09-19 · Status: Accepted

## Context

No Alloy runs on the host, so BFF JSON logs would stop at `docker logs`. Alloy needs the Docker socket, which should not sit in the same compose stack as the BFF.

## Decision

Create `deploy/alloy/` as an independent stack at `/srv/stacks/alloy/` that tails container logs and pushes to Loki on `svrdocker` over the LAN.

## Consequences

- The BFF stack remains socket-free.
- Acceptance test 10 covers both metrics and logs end to end.

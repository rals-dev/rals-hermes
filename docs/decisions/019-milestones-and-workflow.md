# ADR-019: Milestones M0–M4, test-first, one commit per ticket on main

Date: 2026-09-19 · Status: Accepted

## Context

One engineer, one reviewer. PR ceremony adds no review value; a clean commit history and green CI are what a reviewer sees. Observability bolted on at the end tends to miss the interesting code paths.

## Decision

Deliver in milestones M0 (prerequisites) → M1 (core BFF incl. metrics and logs) → M2 (sessions, runs, feed, jobs) → M3 (Vue) → M4 (deployment, docs, acceptance). Write tests from fixtures before implementation. One commit per ticket directly on `main`; tag `v0.1.0` at the end of M4.

## Consequences

- Metrics/log tickets move from Epic 4 into M1.
- No feature branches unless a change is experimental.

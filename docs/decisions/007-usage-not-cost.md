# ADR-007: Show token usage per session; defer cost and per-agent aggregates

Date: 2026-09-19 · Status: Accepted

## Context

Cost requires joining per-session usage with per-model pricing from `/api/model/options` and handling unknown models. Per-agent aggregates require summing over every session on every request or persisting state, which the 'no database' constraint forbids.

## Decision

v1 passes through token usage fields per session and per delegation where Hermes provides them. Cost and aggregates move to v2.

## Consequences

- The PRD problem statement is softened from 'tokens per agent' to 'tokens per session'.
- `/api/model/options` pricing metadata is fetched for the agent detail page but not used in calculations.

# ADR-002: Collect real fixtures before writing the client

Date: 2026-09-19 · Status: Accepted

## Context

Hermes is pre-1.0 and the per-session response schema of `/api/sessions` is undocumented. Guessing struct shapes from documentation alone guarantees a refactor once real payloads arrive.

## Decision

`scripts/collect-fixtures.sh` captures every consumed endpoint for every profile, redacts identifiers and message bodies, and stores the result under `testdata/fixtures/`. Unit tests are written against those fixtures. The script is re-run on every Hermes upgrade and the diff is reviewed.

## Consequences

- Milestone M0 blocks on the operator running the script.
- Fixture diffs become the contract-change detector.

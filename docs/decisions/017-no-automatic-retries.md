# ADR-017: No automatic retries in the Hermes client

Date: 2026-09-19 · Status: Accepted

## Context

The per-profile timeout is 2 s and the overview must load in < 2 s. One retry after a failure already breaks that budget. Retrying against an overloaded Hermes also works against the 'never cause a 429' criterion.

## Decision

The client makes exactly one attempt per request, bounded by the request context. The UI's 5 s polling and the 3 s cache are the retry. A dial-failure-only retry within the remaining deadline may be added later behind the same interface if flapping is observed.

## Consequences

- T-102 wording changes from 'limited retry' to 'per-profile timeout; no automatic retries'.

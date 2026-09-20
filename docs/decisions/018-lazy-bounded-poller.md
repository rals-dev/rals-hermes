# ADR-018: Activity poller is lazy and bounded

Date: 2026-09-19 · Status: Accepted

## Context

A naïve poller (4 profiles × 1 + K requests every 5 s) would put ~5 req/s on Hermes continuously, even with nobody watching — contradicting 'the BFF must stay light' and the spirit of acceptance test 9.

## Decision

Per profile: start on the first SSE subscriber, stop 30 s after the last leaves; poll every 5 s (configurable); track only sessions updated in the last 10 minutes, at most 10; keep one cursor per tracked session and drop it when the session leaves the window; on upstream failure emit an `error` event and keep trying.

## Consequences

- Zero upstream load when the dashboard is closed.
- First events after opening the page may take up to one interval.

## Addendum 2026-09-20 (after live verification)

`message_count` equals the number of rows returned by `/messages`
(verified: count 435, `offset=434` → exactly one row), so the per-session
cursor is the offset. `order=newest` is not supported by Hermes 0.21.2.

A session that was dormant at baseline and becomes active again is *not*
new: the poller fetches only the last few rows (`offset = count − 8`),
drops anything older than the window, and emits a snapshot rather than
`session.started`. The first live run replayed 435 messages before this
rule existed.

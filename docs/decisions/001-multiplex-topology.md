# ADR-001: Use Hermes multiplex topology (one listener, /p/<profile>/ prefixes)

Date: 2026-09-19 · Status: Accepted

## Context

Hermes can serve several profiles either from one listener with `gateway.multiplex_profiles: true` (prefix `/p/<profile>/`) or from one gateway process per profile on separate ports. The host runs Hermes as one container with a CPU limit of 4.0, and multiplexing was found to be already enabled on the host.

## Decision

Use the multiplex topology. The BFF client takes a fully configurable base URL per profile so that separate ports would be a config-only change.

## Consequences

- Partial-degradation testing cannot stop one gateway; it blanks one profile's key instead (→ `unauthorized`).
- Key isolation is enforced by Hermes at the prefix level (July 2026 breaking change): the default key is rejected on named prefixes.

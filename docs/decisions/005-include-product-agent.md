# ADR-005: Include the product-agent profile in v1

Date: 2026-09-19 · Status: Accepted

## Context

A fourth profile, `product-agent`, exists on the host but was absent from the original PRD. Profiles are configuration, not code (principle #4).

## Decision

Configure all four profiles. If `product-agent` lacks an API key it will surface as `unauthorized` until the operator sets one — which doubles as a live demonstration that adding a profile is config-only.

## Consequences

- One more env var (`HERMES_KEY_PRODUCT`), one more fixture set, one more card on the overview.

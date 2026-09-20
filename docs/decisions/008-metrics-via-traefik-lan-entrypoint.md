# ADR-008: Expose /metrics to svrdocker through a Traefik LAN entrypoint with an IP allow-list

Date: 2026-09-19 · Status: Superseded in part (2026-09-20)

## Context

Prometheus lives on `svrdocker`, which is not on the tailnet; it reaches the host over the LAN only. Traefik listens only on loopback and the Tailscale IP, and the BFF port is not published. Options considered: publish the BFF port on the LAN (rejected — exposes the whole API), install Tailscale on `svrdocker` (out of this repo's scope), or defer metrics (leaves Epic 4 unproven).

## Decision

Add a Traefik entrypoint bound to the LAN IP with a router for `/metrics` only, guarded by an `IPAllowList` middleware containing `svrdocker`'s address. Tailscale-facing routers are untouched. Metric retention stays in the existing Prometheus; the BFF stores no history.

## Consequences

- Traefik now listens on the LAN; the surface is one path, one source IP.
- `deploy/traefik/` documents the patch; the operator applies it (M4).

## Addendum 2026-09-20

Prometheus now runs on the same host (`deploy/prometheus/`), joined to
`proxy-net`, scraping `hermes-dashboard:8080/metrics` directly. The Traefik
LAN entrypoint and allow-list are no longer required; the design is kept in
`deploy/traefik/README.md` as the alternative for scraping from another
host. Grafana on `svrdocker` reads this Prometheus over the LAN as a data
source. Retention stays in Prometheus; the BFF still stores nothing.

# Architecture Decision Records

One file per decision, numbered in the order they were taken during the
2026-09-19 handoff review. When the PRD and an ADR disagree, the ADR wins.

| # | Title |
| --- | --- |
| 001 | [Use Hermes multiplex topology (one listener, /p/<profile>/ prefixes)](001-multiplex-topology.md) |
| 002 | [Collect real fixtures before writing the client](002-fixtures-before-code.md) |
| 003 | [Local development reaches Hermes through an SSH port-forward with keys in the shell only](003-local-access-via-port-forward.md) |
| 004 | [Derive the live activity feed from sessions, keep the run SSE relay](004-session-based-activity-feed.md) |
| 005 | [Include the product-agent profile in v1](005-include-product-agent.md) |
| 006 | [Defer Kanban board data to v2](006-defer-kanban.md) |
| 007 | [Show token usage per session; defer cost and per-agent aggregates](007-usage-not-cost.md) |
| 008 | [Expose /metrics to svrdocker through a Traefik LAN entrypoint with an IP allow-list](008-metrics-via-traefik-lan-entrypoint.md) |
| 009 | [Ship logs to Loki from a separate Alloy stack](009-separate-alloy-stack.md) |
| 010 | [Authenticate the browser with a BFF-issued HttpOnly session cookie](010-bff-session-cookie.md) |
| 011 | [Serve the frontend from the Go binary via embed.FS](011-embed-frontend.md) |
| 012 | [Vue 3 + Vite + TypeScript for the frontend](012-vue-frontend.md) |
| 013 | [Monorepo with the Go module at the root](013-repo-layout.md) |
| 014 | [Standard library first; four direct dependencies](014-minimal-go-dependencies.md) |
| 015 | [Public repository; GitHub Actions builds and pushes to GHCR](015-public-repo-ghcr-ci.md) |
| 016 | [Pin the Hermes image by digest](016-pin-hermes-digest.md) |
| 017 | [No automatic retries in the Hermes client](017-no-automatic-retries.md) |
| 018 | [Activity poller is lazy and bounded](018-lazy-bounded-poller.md) |
| 019 | [Milestones M0–M4, test-first, one commit per ticket on main](019-milestones-and-workflow.md) |
| 020 | [Everything in the repository is written in English](020-english-everywhere.md) |
| 021 | [Floor view — agents illustrated as workers](021-floor-view.md) |
| 022 | [Per-agent usage monitoring — on-the-fly, no persisted state](022-usage-monitoring.md) |
| 023 | [Office view — agents walking around a 2D pixel-art office](023-office-view.md) |
| 024 | [Agent art redesign — personas in a "night control room"](024-agent-art-redesign.md) |

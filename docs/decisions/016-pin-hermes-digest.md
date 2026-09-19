# ADR-016: Pin the Hermes image by digest

Date: 2026-09-19 · Status: Accepted

## Context

Production runs `nousresearch/hermes-agent:latest`. Fixtures and contract tests are only valid for one build; a silent re-pull could break the dashboard without any change in this repo.

## Decision

The operator pins the compose image to the digest observed on 2026-09-19 (`sha256:f79d70bc…`). Upgrades are deliberate: bump the digest, re-run the fixture script, review the diff, update the 'tested against' line in the README.

## Consequences

- `/v1/capabilities` still guards against endpoint absence; the digest guards against payload drift.

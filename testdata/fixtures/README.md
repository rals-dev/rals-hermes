# Fixtures

Redacted, real payloads from the Hermes API server, one directory per profile,
captured by `scripts/collect-fixtures.sh`. Free-text fields are replaced with
`<redacted N chars>`; anything resembling a secret becomes `<redacted secret>`.

Re-run the script after every Hermes image bump and review the diff — it is
the contract-change detector for this project (ADR-002, ADR-016).

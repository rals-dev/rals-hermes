# ADR-025: Adding a profile is one command, not auto-discovery

Date: 2026-09-29 · Status: Accepted

## Context

Two new Hermes profiles, `researcher-agent` and `reviewer-agent`, were
created on the host and did not appear on the dashboard. That is principle
#4 working as designed ("profiles are added through configuration, not
code"): the BFF reads `upstream.profiles` once at start-up. Still, the
operator asked why the dashboard does not pick new profiles up by itself.

Adding them by hand showed where the effort actually goes:

- Neither profile had an `API_SERVER_KEY`, so each needed a key generated,
  written to the profile's `.env`, and a Hermes restart to load it.
- The same key had to be copied into the dashboard's `.env`.
- `compose.yaml` listed every `HERMES_KEY_*` by name, so it needed an
  edit too — and so did `config.yaml`. Three files, by hand, for each
  profile.

Discovering *names* is easy: the profile directories are listed under
`/srv/data/hermes/profiles`. The hard part is the key. Hermes has no
cross-profile read-only key (the `default` key gets 401 on every named
prefix, `docs/t003-findings.md`) and no "list profiles" endpoint;
`/health/detailed` only names profiles that have a Telegram platform.

Three levels of automation were considered:

1. **One command.** A host script does the manual steps, with checks.
2. **Detect and nag.** Level 1, and the BFF also lists the profile
   directory names (the parent is world-readable, each profile is `0700`)
   and shows unconfigured profiles as "not connected" cards.
3. **Zero touch.** The BFF reads each profile's `API_SERVER_KEY` itself.
   That means mounting Hermes' data into the dashboard and loosening the
   `0700` profile directories, so the dashboard could read every agent's
   secrets: Telegram tokens, GitHub tokens, model credentials. That
   breaks the credential isolation the PRD requires (the reason
   `tester-agent` exists at all).

## Decision

Level 1. `deploy/hermes-dashboard/profile.sh add|remove <name>` runs on the
host from the dashboard stack directory, or from a checkout through
`ssh <host> 'cd <stack dir> && bash -s -- add <name>' < profile.sh`, so the
host never holds a copy that can go stale.

- **Keys.** An existing `API_SERVER_KEY` in the profile's `.env` is reused.
  Otherwise one is generated (`openssl rand -hex 32`). The key goes into the
  dashboard's `.env` as `HERMES_KEY_<NAME>`: `-agent` stripped, upper-cased,
  `-` → `_`, the convention the first four profiles already follow. Key
  values are never printed.
- **Config.** The entry (`name`, `base_url`, `key_env`) is appended to
  `config.yaml` as text. The script checks the layout first:
  `upstream.profiles` must be the last block, with one `    - name:` line per
  entry. Otherwise it refuses rather than guessing. It also refuses a name
  already configured, a variable already in `.env`, and a profile with no
  directory under Hermes' data.
- **Verify, or roll back.** After backing up both files, the script recreates
  the dashboard and waits (20 s) for its `configuration loaded` log line to
  list the new profile. A start-up error, a missing line, or the wrong
  profile list restores both files and recreates again.
- **Hermes restart is opt-in.** A generated key is written to the profile's
  `.env` only after the dashboard is verified. Hermes reads it on restart,
  which interrupts every agent's in-flight work, so the script restarts
  Hermes only with `--restart-hermes`. Without the flag it prints the
  command, and the profile shows as `unauthorized` until then. An
  interactive prompt is impossible anyway: under `bash -s` stdin is the
  script itself.
- **Remove** is the same flow in reverse. It drops the entry and its
  variable (unless another entry still uses it) and never touches Hermes'
  files. It refuses to remove the last profile.
- **`compose.yaml` uses `env_file: .env`** instead of listing each key. The
  only non-secret extras that reach the container are `IMAGE_OWNER`,
  `IMAGE_TAG`, `DASHBOARD_HOST` and `LOG_LEVEL`. Compose's `${VAR:?}` guard
  is gone. The BFF already refuses to start on an empty `key_env`, and the
  script's verification catches that.
- **Tests.** `profile_test.sh` runs the script through `bash -s`, as the
  runbook does, against a throwaway host layout with a fake `docker` on
  `PATH`. CI runs it and `shellcheck` in a `scripts` job that the image job
  does not depend on.

Principle #4 stands as written. `profile.sh` automates the config change and
the restart; it adds no code path to the BFF.

## Consequences

- Adding a profile is one command and at most one Hermes restart. The
  script touches two dashboard files and, for a new key, the profile's
  `.env`.
- Examples in the repository (`config.example.yaml`, `.env.example`,
  README) keep the original four profiles as illustrations. The PRD's
  "Monitored profiles" table records the live set, and the runbook reads
  key names from `config.yaml` instead of a fixed list.
- Moving the host's `compose.yaml` to `env_file` is a one-time migration.
- A profile created in Hermes still does not appear until someone runs the
  script. Level 2 (detect and nag) can be added later without changing
  anything decided here.
- Profiles other than the four ADR-024 personas get the procedural persona.
  Dedicated art for `researcher-agent` and `reviewer-agent` is out of scope.

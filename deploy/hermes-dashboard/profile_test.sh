#!/usr/bin/env bash
# Tests for profile.sh (ADR-025). Each test builds a throwaway host layout —
# a dashboard stack directory, a Hermes data directory and a fake `docker`
# on PATH — and runs the script exactly as the runbook does: piped into
# `bash -s` from the stack directory.
#
# Run: bash deploy/hermes-dashboard/profile_test.sh   (or: make test-scripts)
#
# The fake docker simulates the dashboard start-up per `docker compose up`:
# FAKE_BFF is a comma-separated list of outcomes, one per call —
#   ok     logs "configuration loaded" with the profiles in config.yaml
#   fail   logs a start-up error, as cmd/bff does before exiting
#   silent logs nothing (a container that never gets that far)
# Calls beyond the list behave as "ok".

# Test functions are invoked indirectly by the runner at the bottom.
# shellcheck disable=SC2317

set -u

HERE=$(cd "$(dirname "$0")" && pwd)
SCRIPT="$HERE/profile.sh"
KEY_EXISTING=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

passed=0
failed=0
T=
status=0
case_failed=0

setup() {
  T=$(mktemp -d)
  mkdir -p "$T/bin" "$T/dash" "$T/hermes" "$T/fake" \
    "$T/data/profiles/coder-agent" \
    "$T/data/profiles/researcher-agent" \
    "$T/data/profiles/reviewer-agent"

  cat > "$T/bin/docker" <<'FAKE'
#!/usr/bin/env bash
printf '%s|%s\n' "$PWD" "$*" >> "$FAKE_DIR/calls"
case "$*" in
  "compose up -d --force-recreate")
    n=$(( $(cat "$FAKE_DIR/ups" 2>/dev/null || echo 0) + 1 ))
    echo "$n" > "$FAKE_DIR/ups"
    mode=$(printf '%s\n' "${FAKE_BFF:-ok}" | cut -d, -f"$n")
    case "${FAKE_BFF:-ok}" in *,*) ;; *) [ "$n" -eq 1 ] || mode=ok ;; esac
    case "${mode:-ok}" in
      ok)
        names=$(sed -n 's/^    - name: \(.*\)$/"\1"/p' config.yaml | paste -sd, -)
        printf '{"time":"t","level":"INFO","msg":"configuration loaded","config":"/app/config.yaml","profiles":[%s],"upstream_timeout":"2s"}\n' "$names" > "$FAKE_DIR/log" ;;
      fail)
        printf 'hermes-bff: config: environment variable HERMES_KEY_X (profile "x") is required but empty\n' > "$FAKE_DIR/log" ;;
      silent)
        : > "$FAKE_DIR/log" ;;
    esac ;;
  "compose logs --no-log-prefix bff")
    cat "$FAKE_DIR/log" 2>/dev/null ;;
  "compose restart hermes") ;;
  *) echo "fake docker: unexpected call: $*" >&2; exit 99 ;;
esac
FAKE
  chmod +x "$T/bin/docker"

  : > "$T/dash/compose.yaml"
  cat > "$T/dash/config.yaml" <<'YAML'
# BFF configuration for the compose stack.
server:
  addr: ":8080"

auth:
  key_env: BFF_API_KEY

upstream:
  timeout: 2s
  profiles:
    - name: default
      base_url: http://hermes:8642
      key_env: HERMES_KEY_DEFAULT
    - name: coder-agent
      base_url: http://hermes:8642/p/coder-agent
      key_env: HERMES_KEY_CODER
YAML
  printf 'IMAGE_OWNER=rals-dev\nBFF_API_KEY=bff\nHERMES_KEY_DEFAULT=k-default\nHERMES_KEY_CODER=k-coder\n' > "$T/dash/.env"
  chmod 600 "$T/dash/.env"

  printf 'API_SERVER_HOST=0.0.0.0\nAPI_SERVER_KEY=k-default\n' > "$T/data/.env"
  printf 'API_SERVER_KEY=k-coder\n' > "$T/data/profiles/coder-agent/.env"
  printf 'TELEGRAM_ALLOWED_USERS=1\n' > "$T/data/profiles/researcher-agent/.env"
  printf 'TELEGRAM_ALLOWED_USERS=1\nAPI_SERVER_KEY=%s\n' "$KEY_EXISTING" > "$T/data/profiles/reviewer-agent/.env"
  chmod 600 "$T"/data/.env "$T"/data/profiles/*/.env

  cp -p "$T/dash/config.yaml" "$T/config.orig"
  cp -p "$T/dash/.env" "$T/env.orig"
  cp -p "$T/data/profiles/researcher-agent/.env" "$T/researcher.orig"
  cp -p "$T/data/profiles/reviewer-agent/.env" "$T/reviewer.orig"
}

teardown() { rm -rf "$T"; }

# run [ENV=value ...] -- <script args>
run() {
  local envs=()
  while [ "$1" != "--" ]; do envs+=("$1"); shift; done
  shift
  (
    cd "$T/dash" &&
      env PATH="$T/bin:$PATH" FAKE_DIR="$T/fake" HERMES_DATA="$T/data" \
        HERMES_STACK="$T/hermes" VERIFY_TIMEOUT=2 ${envs[@]+"${envs[@]}"} \
        bash -s -- "$@" < "$SCRIPT"
  ) > "$T/out" 2>&1
  status=$?
}

check() {
  local what=$1; shift
  if ! "$@"; then
    echo "    FAIL: $what"
    case_failed=1
  fi
}

# Substring match on the whole file, so multi-line blocks must match as a
# block (grep -F would treat each line as a separate pattern).
contains() {
  local content
  content=$(cat "$1" 2>/dev/null)
  case "$content" in *"$2"*) return 0 ;; esac
  return 1
}
lacks() { ! contains "$1" "$2"; }
same() { cmp -s "$1" "$2"; }
no_docker_calls() { [ ! -s "$T/fake/calls" ]; }
env_value() { sed -n "s/^$2=//p" "$1"; }

# --- add -------------------------------------------------------------------

t_add_generates_key_when_profile_has_none() {
  run -- add researcher-agent
  check "exit 0" [ "$status" -eq 0 ]
  check "config entry appended" contains "$T/dash/config.yaml" "    - name: researcher-agent
      base_url: http://hermes:8642/p/researcher-agent
      key_env: HERMES_KEY_RESEARCHER"
  check "config ends with the new entry" [ "$(tail -n 3 "$T/dash/config.yaml" | head -n 1)" = "    - name: researcher-agent" ]
  local dash_key prof_key
  dash_key=$(env_value "$T/dash/.env" HERMES_KEY_RESEARCHER)
  prof_key=$(env_value "$T/data/profiles/researcher-agent/.env" API_SERVER_KEY)
  check "dashboard key is 64 hex chars" grep -Eqx '[0-9a-f]{64}' <<<"$dash_key"
  check "profile got the same key" [ "$dash_key" = "$prof_key" ]
  check "profile's other settings kept" contains "$T/data/profiles/researcher-agent/.env" "TELEGRAM_ALLOWED_USERS=1"
  check "key never printed" lacks "$T/out" "$dash_key"
  check "restart pending announced" contains "$T/out" "Hermes restart pending"
  check "Hermes not restarted" lacks "$T/fake/calls" "restart hermes"
  check "dashboard recreated in the stack dir" contains "$T/fake/calls" "$T/dash|compose up -d --force-recreate"
  check ".env still private" [ -z "$(find "$T/dash/.env" -perm -004)" ]
  check "config backup kept" [ -n "$(ls "$T"/dash/config.yaml.bak-* 2>/dev/null)" ]
  check ".env backup kept" [ -n "$(ls "$T"/dash/.env.bak-* 2>/dev/null)" ]
}

t_add_reuses_existing_key() {
  run -- add reviewer-agent
  check "exit 0" [ "$status" -eq 0 ]
  check "dashboard got the profile's key" [ "$(env_value "$T/dash/.env" HERMES_KEY_REVIEWER)" = "$KEY_EXISTING" ]
  check "profile .env untouched" same "$T/data/profiles/reviewer-agent/.env" "$T/reviewer.orig"
  check "no restart pending" lacks "$T/out" "restart pending"
  check "key never printed" lacks "$T/out" "$KEY_EXISTING"
}

t_add_restarts_hermes_when_asked() {
  run -- add researcher-agent --restart-hermes
  check "exit 0" [ "$status" -eq 0 ]
  check "Hermes restarted from its stack dir" contains "$T/fake/calls" "$T/hermes|compose restart hermes"
  check "no restart pending" lacks "$T/out" "restart pending"
}

t_add_skips_restart_when_key_was_reused() {
  run -- add reviewer-agent --restart-hermes
  check "exit 0" [ "$status" -eq 0 ]
  check "Hermes not restarted" lacks "$T/fake/calls" "restart hermes"
}

t_add_default_profile_uses_root_env_and_url() {
  local tmp
  tmp=$(mktemp)
  awk '$0 == "    - name: default" { skip=1; next } skip && /^    - name: / { skip=0 } !skip' "$T/dash/config.yaml" > "$tmp"
  cat "$tmp" > "$T/dash/config.yaml"
  grep -v '^HERMES_KEY_DEFAULT=' "$T/dash/.env" > "$tmp"
  cat "$tmp" > "$T/dash/.env"
  rm -f "$tmp"
  run -- add default
  check "exit 0" [ "$status" -eq 0 ]
  check "base_url has no /p/ prefix" contains "$T/dash/config.yaml" "    - name: default
      base_url: http://hermes:8642
      key_env: HERMES_KEY_DEFAULT"
  check "key read from the root .env" [ "$(env_value "$T/dash/.env" HERMES_KEY_DEFAULT)" = "k-default" ]
}

t_add_refuses_duplicate() {
  run -- add coder-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "says already configured" contains "$T/out" "already configured"
  check "config unchanged" same "$T/dash/config.yaml" "$T/config.orig"
  check ".env unchanged" same "$T/dash/.env" "$T/env.orig"
  check "docker not called" no_docker_calls
}

t_add_refuses_unknown_profile_dir() {
  run -- add ghost-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "names the missing directory" contains "$T/out" "$T/data/profiles/ghost-agent"
  check "config unchanged" same "$T/dash/config.yaml" "$T/config.orig"
  check "docker not called" no_docker_calls
}

t_add_refuses_invalid_name() {
  run -- add ../etc
  check "exit 1" [ "$status" -eq 1 ]
  check "says invalid" contains "$T/out" "invalid profile name"
  check "docker not called" no_docker_calls
}

t_add_refuses_env_var_already_set() {
  printf 'HERMES_KEY_RESEARCHER=something\n' >> "$T/dash/.env"
  cp -p "$T/dash/.env" "$T/env.orig"
  run -- add researcher-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "names the variable" contains "$T/out" "HERMES_KEY_RESEARCHER is already set"
  check "config unchanged" same "$T/dash/config.yaml" "$T/config.orig"
  check "profile .env untouched" same "$T/data/profiles/researcher-agent/.env" "$T/researcher.orig"
}

t_add_refuses_when_profiles_is_not_the_last_block() {
  printf '\nlogging:\n  level: info\n' >> "$T/dash/config.yaml"
  cp -p "$T/dash/config.yaml" "$T/config.orig"
  run -- add researcher-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "says layout" contains "$T/out" "layout"
  check "config unchanged" same "$T/dash/config.yaml" "$T/config.orig"
  check "docker not called" no_docker_calls
}

t_add_rolls_back_when_dashboard_fails_to_start() {
  run FAKE_BFF=fail,ok -- add researcher-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "shows the start-up error" contains "$T/out" "is required but empty"
  check "says rolled back" contains "$T/out" "rolled back"
  check "config restored" same "$T/dash/config.yaml" "$T/config.orig"
  check ".env restored" same "$T/dash/.env" "$T/env.orig"
  check "profile .env untouched" same "$T/data/profiles/researcher-agent/.env" "$T/researcher.orig"
  check "dashboard recreated twice" [ "$(grep -c 'compose up' "$T/fake/calls")" -eq 2 ]
  check "Hermes not restarted" lacks "$T/fake/calls" "restart hermes"
}

t_add_rolls_back_when_dashboard_stays_silent() {
  run FAKE_BFF=silent,ok -- add researcher-agent --restart-hermes
  check "exit 1" [ "$status" -eq 1 ]
  check "says it timed out" contains "$T/out" "within 2s"
  check "config restored" same "$T/dash/config.yaml" "$T/config.orig"
  check ".env restored" same "$T/dash/.env" "$T/env.orig"
  check "Hermes not restarted" lacks "$T/fake/calls" "restart hermes"
}

t_add_reports_failed_rollback() {
  run FAKE_BFF=fail,fail -- add researcher-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "says rollback failed" contains "$T/out" "ROLLBACK"
  check "config restored anyway" same "$T/dash/config.yaml" "$T/config.orig"
}

# --- remove ----------------------------------------------------------------

t_remove_drops_entry_and_key() {
  run -- remove coder-agent
  check "exit 0" [ "$status" -eq 0 ]
  check "entry gone" lacks "$T/dash/config.yaml" "coder-agent"
  check "key var gone" lacks "$T/dash/.env" "HERMES_KEY_CODER"
  check "default entry intact" contains "$T/dash/config.yaml" "    - name: default
      base_url: http://hermes:8642
      key_env: HERMES_KEY_DEFAULT"
  check "other vars intact" contains "$T/dash/.env" "BFF_API_KEY=bff"
  check "Hermes files untouched" contains "$T/data/profiles/coder-agent/.env" "API_SERVER_KEY=k-coder"
  check "dashboard recreated" contains "$T/fake/calls" "compose up -d --force-recreate"
}

t_remove_then_add_round_trips() {
  run -- add reviewer-agent
  check "add exit 0" [ "$status" -eq 0 ]
  cp -p "$T/dash/config.yaml" "$T/config.added"
  run -- remove reviewer-agent
  check "remove exit 0" [ "$status" -eq 0 ]
  check "config back to original" same "$T/dash/config.yaml" "$T/config.orig"
  check ".env back to original" same "$T/dash/.env" "$T/env.orig"
}

t_remove_refuses_unknown_profile() {
  run -- remove researcher-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "says not configured" contains "$T/out" "not in config.yaml"
  check "docker not called" no_docker_calls
}

t_remove_refuses_last_profile() {
  run -- remove coder-agent
  run -- remove default
  check "exit 1" [ "$status" -eq 1 ]
  check "says only profile" contains "$T/out" "only profile"
  check "default still configured" contains "$T/dash/config.yaml" "    - name: default"
}

t_remove_rolls_back_when_dashboard_fails_to_start() {
  run FAKE_BFF=fail,ok -- remove coder-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "config restored" same "$T/dash/config.yaml" "$T/config.orig"
  check ".env restored" same "$T/dash/.env" "$T/env.orig"
}

# --- usage and environment -------------------------------------------------

t_usage_errors_exit_2() {
  run -- add
  check "missing name → 2" [ "$status" -eq 2 ]
  run -- rename coder-agent
  check "unknown command → 2" [ "$status" -eq 2 ]
  run -- remove coder-agent --restart-hermes
  check "flag on remove → 2" [ "$status" -eq 2 ]
  check "docker not called" no_docker_calls
}

t_refuses_outside_the_stack_dir() {
  rm "$T/dash/compose.yaml"
  run -- add researcher-agent
  check "exit 1" [ "$status" -eq 1 ]
  check "explains where to run" contains "$T/out" "stack directory"
}

# --- runner ----------------------------------------------------------------

for t in $(declare -F | awk '{print $3}' | grep '^t_'); do
  case_failed=0
  setup
  "$t"
  if [ "$case_failed" -eq 0 ]; then
    passed=$((passed + 1))
    echo "ok   $t"
  else
    failed=$((failed + 1))
    echo "FAIL $t"
    sed 's/^/    | /' "$T/out"
  fi
  teardown
done

echo
echo "$passed passed, $failed failed"
[ "$failed" -eq 0 ]

#!/usr/bin/env bash
# Adds a Hermes profile to the dashboard, or removes one (ADR-025).
#
#   profile.sh add <name> [--restart-hermes]
#   profile.sh remove <name>
#
# Run it on the host from the dashboard stack directory (the one holding
# compose.yaml, config.yaml and .env), as the user that owns the Hermes
# data. From a checkout, without copying anything to the host:
#
#   ssh <host> 'cd /srv/stacks/hermes-dashboard && bash -s -- add <name>' \
#     < deploy/hermes-dashboard/profile.sh
#
# add    Reuses the profile's API_SERVER_KEY, or generates one. Appends the
#        profile to config.yaml and its key to .env, recreates the dashboard
#        and waits for it to log the new profile; otherwise both files are
#        restored. A generated key is written to the profile's .env only
#        once the dashboard is up. Hermes reads it on its next restart, which
#        interrupts every agent, so that happens only with --restart-hermes.
# remove Drops the profile from config.yaml and its key from .env, with the
#        same check and rollback. Hermes' own files are never touched.
#
# Key values are never printed. Backups (*.bak-<timestamp>) are left next to
# the files they copy.
#
# Environment: DASHBOARD_DIR (default: current directory), HERMES_DATA
# (/srv/data/hermes), HERMES_STACK (/srv/stacks/hermes), HERMES_URL
# (http://hermes:8642), VERIFY_TIMEOUT (20 seconds).

set -euo pipefail
umask 077

HERMES_DATA=${HERMES_DATA:-/srv/data/hermes}
HERMES_STACK=${HERMES_STACK:-/srv/stacks/hermes}
HERMES_URL=${HERMES_URL:-http://hermes:8642}
VERIFY_TIMEOUT=${VERIFY_TIMEOUT:-20}
DASHBOARD_DIR=${DASHBOARD_DIR:-$PWD}
SERVICE=bff

say() { echo "profile.sh: $*"; }
die() { echo "profile.sh: $*" >&2; exit 1; }
usage() {
  echo "usage: profile.sh add <name> [--restart-hermes]" >&2
  echo "       profile.sh remove <name>" >&2
  exit 2
}

[ $# -ge 2 ] || usage
cmd=$1
name=$2
shift 2
restart_hermes=0
for arg in "$@"; do
  case "$cmd:$arg" in
    add:--restart-hermes) restart_hermes=1 ;;
    *) usage ;;
  esac
done
case "$cmd" in add | remove) ;; *) usage ;; esac

grep -Eqx '[a-z0-9][a-z0-9_-]*' <<<"$name" || die "invalid profile name '$name'"

cd "$DASHBOARD_DIR"
if [ ! -f compose.yaml ] || [ ! -f config.yaml ] || [ ! -f .env ]; then
  die "run this from the dashboard stack directory (compose.yaml, config.yaml, .env) or set DASHBOARD_DIR"
fi
command -v docker >/dev/null || die "docker not found"

tmp=
trap 'rm -f "${tmp:-}"' EXIT

# The script edits config.yaml as text, so it insists on the layout it
# writes: upstream.profiles is the last block, and each entry is a
# "    - name:" line followed by "      key: value" lines.
check_layout() {
  awk '
    !inp && /^[^ \t#]/ { up = ($0 ~ /^upstream:[ \t]*$/); next }
    !inp && up && /^  profiles:[ \t]*$/ { inp = 1; next }
    !inp { next }
    /^[ \t]*$/ || /^[ \t]*#/ { next }
    /^    - name: [a-z0-9][a-z0-9_-]*$/ { entries++; next }
    /^      [a-z_]+: [^ \t]/ { next }
    { bad = 1; exit }
    END { exit (inp && entries > 0 && !bad) ? 0 : 1 }
  ' config.yaml || die "config.yaml layout not recognised (upstream.profiles must be the last block, one '    - name:' line per entry); edit it by hand"
}

is_configured() { grep -qxF "    - name: $name" config.yaml; }

# key_env of this profile's entry, as written in config.yaml.
entry_key_env() {
  awk -v n="$name" '
    $0 == "    - name: " n { blk = 1; next }
    blk && /^    - name: / { exit }
    blk && /^      key_env: / { sub(/^      key_env: /, ""); print; exit }
  ' config.yaml
}

# researcher-agent → HERMES_KEY_RESEARCHER, the convention the first four
# profiles already follow.
env_var_for() {
  local base=${1%-agent}
  printf 'HERMES_KEY_%s' "$(printf '%s' "$base" | tr '[:lower:]' '[:upper:]' | tr '-' '_')"
}

ensure_newline() {
  if [ -s "$1" ] && [ -n "$(tail -c 1 "$1")" ]; then printf '\n' >> "$1"; fi
}

backup() {
  local ts
  ts=$(date +%Y%m%d-%H%M%S)
  cfg_bak=config.yaml.bak-$ts
  env_bak=.env.bak-$ts
  cp -p config.yaml "$cfg_bak"
  cp -p .env "$env_bak"
}

# Rewrites a file in place (cat, not mv) so its inode, owner and mode stay —
# config.yaml is bind-mounted into the container.
replace_with_tmp() {
  cat "$tmp" > "$1"
  rm -f "$tmp"
  tmp=
}

recreate() { docker compose up -d --force-recreate; }

# Waits for the dashboard's start-up log line. $1: present | absent | any —
# whether $name must be among the profiles it loaded.
started() {
  local deadline logs line profiles
  deadline=$(($(date +%s) + VERIFY_TIMEOUT))
  while :; do
    logs=$(docker compose logs --no-log-prefix "$SERVICE" 2>&1 || true)
    # cmd/bff prints start-up errors as "hermes-bff: ..." and they never
    # contain secret values.
    if grep -q '^hermes-bff: ' <<<"$logs"; then
      grep '^hermes-bff: ' <<<"$logs" | tail -n 1 | sed 's/^/profile.sh: dashboard: /' >&2
      return 1
    fi
    line=$(grep '"msg":"configuration loaded"' <<<"$logs" | tail -n 1 || true)
    if [ -n "$line" ]; then
      profiles=$(sed -n 's/.*"profiles":\[\([^]]*\)\].*/\1/p' <<<"$line")
      case "$1:,$profiles," in
        any:*) return 0 ;;
        present:*",\"$name\","*) return 0 ;;
        absent:*",\"$name\","*) ;;
        absent:*) return 0 ;;
      esac
      echo "profile.sh: dashboard started, but its profiles are [$profiles]" >&2
      return 1
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
      echo "profile.sh: the dashboard did not log its start-up within ${VERIFY_TIMEOUT}s" >&2
      return 1
    fi
    sleep 1
  done
}

rollback() {
  cat "$cfg_bak" > config.yaml
  cat "$env_bak" > .env
  echo "profile.sh: restored config.yaml and .env from $cfg_bak and $env_bak" >&2
  if recreate && started any; then
    echo "profile.sh: rolled back; the dashboard runs the previous configuration" >&2
  else
    echo "profile.sh: ROLLBACK did not bring the dashboard up; see 'docker compose logs $SERVICE' in $DASHBOARD_DIR" >&2
  fi
  exit 1
}

add() {
  local pdir penv base_url var key new_key=0
  if [ "$name" = default ]; then
    pdir=$HERMES_DATA
    base_url=$HERMES_URL
  else
    pdir=$HERMES_DATA/profiles/$name
    base_url=$HERMES_URL/p/$name
  fi
  penv=$pdir/.env
  var=$(env_var_for "$name")

  [ -d "$pdir" ] || die "no Hermes profile directory at $pdir"
  check_layout
  if is_configured; then die "profile '$name' is already configured in config.yaml"; fi
  if grep -q "^$var=" .env; then die "$var is already set in .env; remove it first"; fi

  key=
  if [ -f "$penv" ]; then
    key=$(sed -n 's/^API_SERVER_KEY=//p' "$penv" | tail -n 1)
    key=${key#\"} key=${key%\"} key=${key#\'} key=${key%\'}
  fi
  if [ -n "$key" ]; then
    grep -Eqx '[A-Za-z0-9._~+/=-]+' <<<"$key" ||
      die "the API_SERVER_KEY in $penv has characters .env cannot hold unquoted; add it by hand"
  else
    if [ -e "$penv" ]; then [ -w "$penv" ]; else [ -w "$pdir" ]; fi || die "cannot write $penv"
    command -v openssl >/dev/null || die "openssl not found"
    key=$(openssl rand -hex 32)
    new_key=1
  fi

  backup
  ensure_newline config.yaml
  printf '    - name: %s\n      base_url: %s\n      key_env: %s\n' "$name" "$base_url" "$var" >> config.yaml
  ensure_newline .env
  printf '%s=%s\n' "$var" "$key" >> .env

  recreate || rollback
  started present || rollback
  say "the dashboard now monitors '$name' (key in $var)"

  if [ "$new_key" -eq 1 ]; then
    ensure_newline "$penv"
    printf 'API_SERVER_KEY=%s\n' "$key" >> "$penv"
    say "generated a new API_SERVER_KEY and wrote it to $penv"
    if [ "$restart_hermes" -eq 1 ]; then
      say "restarting Hermes; every agent is interrupted for about a minute"
      (cd "$HERMES_STACK" && docker compose restart hermes)
    else
      say "Hermes restart pending: '$name' shows as unauthorized until Hermes restarts."
      say "  cd $HERMES_STACK && docker compose restart hermes   (interrupts every agent)"
    fi
  else
    say "reused the API_SERVER_KEY already in $penv; Hermes needs no restart"
  fi
  key=
  say "backups: $cfg_bak $env_bak"
}

remove() {
  local var
  check_layout
  is_configured || die "profile '$name' is not in config.yaml"
  [ "$(grep -c '^    - name: ' config.yaml)" -gt 1 ] ||
    die "'$name' is the only profile; the dashboard needs at least one"
  var=$(entry_key_env)

  backup
  tmp=$(mktemp "$DASHBOARD_DIR/.profile.XXXXXX")
  awk -v n="$name" '
    $0 == "    - name: " n { skip = 1; next }
    skip && /^    - name: / { skip = 0 }
    !skip
  ' config.yaml > "$tmp"
  replace_with_tmp config.yaml
  # Keep the variable if another entry still points at it.
  if [ -n "$var" ] && ! grep -qxF "      key_env: $var" config.yaml; then
    tmp=$(mktemp "$DASHBOARD_DIR/.profile.XXXXXX")
    grep -v "^$var=" .env > "$tmp" || true
    replace_with_tmp .env
  fi

  recreate || rollback
  started absent || rollback
  say "the dashboard no longer monitors '$name'${var:+ ($var removed from .env)}; Hermes' files were not touched"
  say "backups: $cfg_bak $env_bak"
}

case "$cmd" in
  add) add ;;
  remove) remove ;;
esac

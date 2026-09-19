#!/usr/bin/env bash
# collect-fixtures.sh — capture redacted Hermes API payloads for every profile.
#
# Run from the repository root on a workstation that has an SSH port-forward
# to the Hermes container (see docs/operator-tasks.md § 0.3) and the profile
# keys exported in the shell (§ 0.4). Keys are read from the environment only;
# nothing is written to disk except the redacted JSON fixtures.
#
# Usage:
#   ./scripts/collect-fixtures.sh [BASE_URL]        # default http://127.0.0.1:8642
#
# Output:
#   testdata/fixtures/<profile>/<endpoint>.json     redacted body
#   testdata/fixtures/<profile>/<endpoint>.status   HTTP status code
#   testdata/fixtures/<profile>/_report.txt         one-line-per-endpoint report
set -euo pipefail

BASE_URL="${1:-http://127.0.0.1:8642}"
OUT_ROOT="testdata/fixtures"
CURL_TIMEOUT=10

# name:prefix:key-env — the BFF config mirrors this list.
PROFILES=(
  "default::HERMES_KEY_DEFAULT"
  "coder-agent:/p/coder-agent:HERMES_KEY_CODER"
  "tester-agent:/p/tester-agent:HERMES_KEY_TESTER"
  "product-agent:/p/product-agent:HERMES_KEY_PRODUCT"
)

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

# Redaction rules (applied to every fixture):
#  - values under keys that carry free text or tool I/O are replaced by a
#    length marker, preserving structure for tests;
#  - any string that looks like a bearer key or long hex/base64 token is masked;
#  - absolute home paths are generalised.
REDACT_JQ='
def freetext_keys: ["content","text","message","prompt","summary","title","preview","args","arguments","result","output","input","error_message","instructions","description","system_prompt","name_hint"];
def identity_keys: ["user_id","chat_id","username","email","phone"];
def looks_secret: test("^(sk-|hm-|hermes_)[A-Za-z0-9_-]{16,}$") or test("^[A-Fa-f0-9]{48,}$") or test("^[A-Za-z0-9+/=_-]{56,}$");
def redact:
  if type == "object" then
    with_entries(
      if (.key | IN(freetext_keys[])) and (.value | type == "string")
      then .value = "<redacted \(.value | length) chars>"
      elif (.key | IN(identity_keys[])) and (.value != null)
      then .value = "<redacted id>"
      else .value |= redact end)
  elif type == "array" then map(redact)
  elif type == "string" then
    if looks_secret then "<redacted secret>"
    else gsub("/home/[^/\"]+"; "/home/<user>") | gsub("/Users/[^/\"]+"; "/Users/<user>") end
  else . end;
redact'

fetch() { # profile prefix key endpoint slug
  local profile="$1" prefix="$2" key="$3" endpoint="$4" slug="$5"
  local dir="$OUT_ROOT/$profile" body status
  mkdir -p "$dir"
  body="$(mktemp)"
  status="$(curl -sS --max-time "$CURL_TIMEOUT" -o "$body" -w '%{http_code}' \
      -H "Authorization: Bearer $key" -H 'Accept: application/json' \
      "$BASE_URL$prefix$endpoint" 2>/dev/null || echo "000")"
  echo "$status" > "$dir/$slug.status"
  if jq -e . "$body" >/dev/null 2>&1; then
    jq "$REDACT_JQ" "$body" > "$dir/$slug.json"
  else
    # Non-JSON (HTML error page, empty body): keep a redacted, truncated text.
    head -c 500 "$body" | sed -E "s/$key/<redacted secret>/g" > "$dir/$slug.txt"
  fi
  rm -f "$body"
  printf '%-32s %s\n' "$endpoint" "$status" | tee -a "$dir/_report.txt"
  # Return the raw (unredacted) body path for follow-up ID extraction only when JSON.
  [ "$status" = "200" ] && [ -f "$dir/$slug.json" ] && echo "$dir/$slug.json" >&3 || echo "" >&3
}

extract_first_id() { # file jq-expr
  jq -r "$2 // empty" "$1" 2>/dev/null | head -n1
}

for entry in "${PROFILES[@]}"; do
  IFS=: read -r profile prefix keyenv <<<"$entry"
  key="${!keyenv:-}"
  echo "=== profile: $profile (prefix '${prefix:-/}')"
  if [ -z "$key" ]; then
    echo "  $keyenv is not set — skipping profile" | tee "$OUT_ROOT/$profile/_report.txt" 2>/dev/null || true
    continue
  fi
  : > "$OUT_ROOT/$profile/_report.txt" 2>/dev/null || { mkdir -p "$OUT_ROOT/$profile"; : > "$OUT_ROOT/$profile/_report.txt"; }

  exec 3>/dev/null
  fetch "$profile" "$prefix" "$key" "/health"                                    "health"
  fetch "$profile" "$prefix" "$key" "/health/detailed"                           "health_detailed"
  fetch "$profile" "$prefix" "$key" "/v1/capabilities"                           "capabilities"
  fetch "$profile" "$prefix" "$key" "/v1/models"                                 "models"
  fetch "$profile" "$prefix" "$key" "/api/model/options"                         "model_options"
  fetch "$profile" "$prefix" "$key" "/v1/skills"                                 "skills"
  fetch "$profile" "$prefix" "$key" "/v1/toolsets"                               "toolsets"
  fetch "$profile" "$prefix" "$key" "/api/jobs"                                  "jobs"
  # Undocumented: does a list-runs endpoint exist at all? (decision #4 verification)
  fetch "$profile" "$prefix" "$key" "/v1/runs"                                   "runs_list"

  # Sessions: list with children, then drill into the most recent one.
  sessions_file=""
  exec 3>"$OUT_ROOT/$profile/.last"
  fetch "$profile" "$prefix" "$key" "/api/sessions?limit=5&offset=0&include_children=true" "sessions"
  exec 3>/dev/null
  sessions_file="$(cat "$OUT_ROOT/$profile/.last")"; rm -f "$OUT_ROOT/$profile/.last"
  if [ -n "$sessions_file" ]; then
    sid="$(extract_first_id "$sessions_file" '(.sessions // .items // .data // .)[0] | (.id // .session_id)')"
    if [ -n "$sid" ]; then
      fetch "$profile" "$prefix" "$key" "/api/sessions/$sid"                     "session_detail"
      fetch "$profile" "$prefix" "$key" "/api/sessions/$sid/messages?limit=20"   "session_messages"
    else
      echo "  no session id found in list — skipping detail/messages" | tee -a "$OUT_ROOT/$profile/_report.txt"
    fi
  fi

  # Jobs: drill into the first job if any.
  if [ -f "$OUT_ROOT/$profile/jobs.json" ]; then
    jid="$(extract_first_id "$OUT_ROOT/$profile/jobs.json" '(.jobs // .items // .data // .)[0] | (.id // .job_id)')"
    [ -n "$jid" ] && fetch "$profile" "$prefix" "$key" "/api/jobs/$jid"          "job_detail"
  fi

  # Cross-profile isolation check: this profile's key on the default prefix
  # (or default's key on this prefix) must be rejected. Recorded, not fixtured.
  if [ -n "$prefix" ] && [ -n "${HERMES_KEY_DEFAULT:-}" ]; then
    code="$(curl -sS --max-time "$CURL_TIMEOUT" -o /dev/null -w '%{http_code}' \
        -H "Authorization: Bearer $HERMES_KEY_DEFAULT" "$BASE_URL$prefix/v1/capabilities" 2>/dev/null || echo 000)"
    printf '%-32s %s  (expect 401)\n' "isolation: default key on $prefix" "$code" | tee -a "$OUT_ROOT/$profile/_report.txt"
  fi
  echo
done

# Summary for the engineer — safe to paste into chat.
echo "=== SUMMARY (safe to share) ==="
for entry in "${PROFILES[@]}"; do
  IFS=: read -r profile _ _ <<<"$entry"
  d="$OUT_ROOT/$profile"
  [ -f "$d/_report.txt" ] || continue
  echo "--- $profile"
  cat "$d/_report.txt"
  if [ -f "$d/health_detailed.json" ]; then
    echo "health_detailed keys: $(jq -r 'paths(scalars) | join(".")' "$d/health_detailed.json" | tr '\n' ' ')"
  fi
  if [ -f "$d/sessions.json" ]; then
    echo "sessions top-level keys: $(jq -r 'if type=="object" then keys|join(",") else "array[\(length)]" end' "$d/sessions.json")"
    echo "first session keys: $(jq -r '((.sessions // .items // .data // .)[0] // {}) | keys | join(",")' "$d/sessions.json")"
  fi
  if [ -f "$d/session_messages.json" ]; then
    echo "messages top-level keys: $(jq -r 'if type=="object" then keys|join(",") else "array[\(length)]" end' "$d/session_messages.json")"
    echo "first message keys: $(jq -r '((.messages // .items // .data // .)[0] // {}) | keys | join(",")' "$d/session_messages.json")"
  fi
done
echo
echo "Fixtures written under $OUT_ROOT/. Review 'git diff' before committing."

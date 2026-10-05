#!/usr/bin/env bash
# Deploy a published rals-hermes image to this stack, with automatic rollback.
#
#   deploy.sh <tag>      tag = short commit SHA (7 hex) or a v* release tag
#   deploy.sh status     print the pinned tag and the live /healthz
#
# Lives in the dashboard stack directory (next to compose.yaml and .env).
# Run by hand, or by the CI deploy job over SSH: that key is pinned to this
# script in ~/.ssh/authorized_keys (docs/runbook.md § 2), so the requested
# tag arrives in SSH_ORIGINAL_COMMAND and nothing else can run.
#
# The tag is pulled first, then pinned in .env and the container recreated.
# If /healthz does not report that version within HEALTH_TIMEOUT seconds,
# the previous tag is restored. Every step is appended to
# deploy-history.log.
set -euo pipefail

STACK_DIR=$(dirname "$(readlink -f "$0")")
ENV_FILE=$STACK_DIR/.env
HISTORY=$STACK_DIR/deploy-history.log
HEALTH_TIMEOUT=45

cd "$STACK_DIR"

arg=${SSH_ORIGINAL_COMMAND:-${1:-}}

env_get() { sed -n "s/^$1=//p" "$ENV_FILE" | tail -n1; }

IMAGE_OWNER=$(env_get IMAGE_OWNER)
DASHBOARD_HOST=$(env_get DASHBOARD_HOST)
IMAGE=ghcr.io/$IMAGE_OWNER/rals-hermes

healthz() { curl -fsS -m 3 "http://$DASHBOARD_HOST/healthz" 2>/dev/null; }

log() { printf '%s %s\n' "$(date -u +%FT%TZ)" "$*" | tee -a "$HISTORY"; }

set_tag() {
	local tmp
	tmp=$(mktemp "$STACK_DIR/.env.XXXXXX")
	sed "s/^IMAGE_TAG=.*/IMAGE_TAG=$1/" "$ENV_FILE" >"$tmp"
	chmod 600 "$tmp"
	mv "$tmp" "$ENV_FILE"
}

# Waits until /healthz reports ok and a version containing $1.
wait_healthy() {
	local want=$1 deadline=$((SECONDS + HEALTH_TIMEOUT)) body
	while ((SECONDS < deadline)); do
		body=$(healthz || true)
		if [[ $body == *'"status":"ok"'* && $body == *"$want"* ]]; then
			echo "healthy: $body"
			return 0
		fi
		sleep 2
	done
	echo "unhealthy after ${HEALTH_TIMEOUT}s, last /healthz: ${body:-<no response>}" >&2
	return 1
}

up() { docker compose --project-directory "$STACK_DIR" up -d bff; }

case $arg in
status)
	echo "pinned: $(env_get IMAGE_TAG)"
	echo "live:   $(healthz || echo '<no response>')"
	exit 0
	;;
'' | *[!0-9A-Za-z.-]*)
	echo "usage: deploy.sh <short-sha|vX.Y.Z> | status" >&2
	exit 2
	;;
esac

tag=$arg
if ! [[ $tag =~ ^[0-9a-f]{7}$ || $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
	echo "refusing tag '$tag': want a 7-hex short SHA or vX.Y.Z" >&2
	exit 2
fi

# One deploy at a time.
exec 9>"$STACK_DIR/.deploy.lock"
flock -n 9 || { echo "another deploy is running" >&2; exit 75; }

prev=$(env_get IMAGE_TAG)
log "deploy $tag (previous $prev)"

# Pull first: a missing tag must not touch the running container.
docker pull -q "$IMAGE:$tag"

set_tag "$tag"
if up && wait_healthy "$tag"; then
	log "deployed $tag ($(docker image inspect -f '{{index .RepoDigests 0}}' "$IMAGE:$tag"))"
	exit 0
fi

log "deploy $tag FAILED, rolling back to $prev"
docker logs --tail 30 hermes-dashboard >&2 || true
set_tag "$prev"
up
# "latest" carries no version to match; accept any healthy response.
[[ $prev == latest ]] && want='"status":"ok"' || want=$prev
if wait_healthy "$want"; then
	log "rolled back to $prev"
else
	log "ROLLBACK to $prev ALSO UNHEALTHY - manual attention needed"
fi
exit 1

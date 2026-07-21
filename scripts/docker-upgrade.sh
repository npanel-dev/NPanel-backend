#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

COMPOSE=${DOCKER_COMPOSE:-"docker compose"}
COMPOSE_FILE=${NPANEL_COMPOSE_FILE:-docker-compose.yml}

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "ERROR: Compose file not found: $COMPOSE_FILE" >&2
  exit 1
fi

if [ "${SKIP_GIT_PULL:-0}" != "1" ] && [ -d .git ]; then
  echo "==> Updating source tree"
  git pull --ff-only
fi

echo "==> Rebuilding NPanel backend image"
$COMPOSE -f "$COMPOSE_FILE" build npanel

echo "==> Restarting NPanel backend container"
$COMPOSE -f "$COMPOSE_FILE" up -d --no-deps --force-recreate npanel

echo "==> Current service status"
$COMPOSE -f "$COMPOSE_FILE" ps

echo "==> Upgrade complete"

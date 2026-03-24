#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
if [[ -f .env ]]; then
  docker compose -f infra/docker/docker-compose.yml --env-file .env up --build
else
  docker compose -f infra/docker/docker-compose.yml up --build
fi

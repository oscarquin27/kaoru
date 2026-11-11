#!/usr/bin/env bash
# Apaga el Kaoru de desarrollo. Con -v borra también sus datos.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
docker compose -p kaoru-dev -f "$root/infra/docker-compose.yml" --env-file "$root/infra/.env.example" down "$@"

#!/usr/bin/env bash
# Levanta un Postgres limpio y desechable con el esquema y los roles de Kaoru,
# ejecuta el comando que recibe y lo apaga. Exporta para el comando:
#   KAORU_TEST_OWNER_URL  conexión de kaoru_owner (migra)
#   KAORU_TEST_APP_URL    conexión de kaoru_app (ejecuta)
#   KAORU_TEST_PG         nombre del contenedor
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
image="${POSTGRES_IMAGE:-postgres:17-alpine}"
port="${CHECK_PG_PORT:-55432}"
export KAORU_TEST_PG="kaoru-check-pg-$$"

docker run -d --rm --name "$KAORU_TEST_PG" -e POSTGRES_PASSWORD=lab -e POSTGRES_DB=kaoru \
    -p "127.0.0.1:$port:5432" "$image" >/dev/null
trap 'docker stop "$KAORU_TEST_PG" >/dev/null' EXIT

until docker exec "$KAORU_TEST_PG" psql -U postgres -d kaoru -qc 'select 1' >/dev/null 2>&1; do sleep 1; done

docker exec -i "$KAORU_TEST_PG" psql -U postgres -d kaoru -v ON_ERROR_STOP=1 -q \
    -v owner_password=lab -v app_password=lab <"$root/infra/postgres/init.sql"

export KAORU_TEST_OWNER_URL="postgres://kaoru_owner:lab@127.0.0.1:$port/kaoru?sslmode=disable"
export KAORU_TEST_APP_URL="postgres://kaoru_app:lab@127.0.0.1:$port/kaoru?sslmode=disable"

"$@"

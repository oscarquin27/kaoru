#!/usr/bin/env bash
# Prueba de humo de la imagen kaoru-api contra el Postgres de tools/with-postgres.sh:
# arranca, migra, queda sana, exige sesión y se apaga en orden.
set -euo pipefail

image="${KAORU_API_IMAGE:-kaoru-api:dev}"
name="kaoru-smoke-api-$$"
pg_port="${CHECK_PG_PORT:-55432}"

docker run -d --name "$name" -p 127.0.0.1:58081:8080 -p 127.0.0.1:59091:9090 \
    -e KAORU_PUBLIC_URL=https://kaoru.lab.invalid \
    -e POSTGRES_HOST="host.docker.internal:$pg_port" \
    -e POSTGRES_OWNER_PASSWORD=lab -e POSTGRES_APP_PASSWORD=lab \
    -e KEYCLOAK_ISSUER=https://kaoru.lab.invalid/realms/kaoru \
    -e KEYCLOAK_JWKS_URL=http://127.0.0.1:1/certs \
    "$image" >/dev/null
trap 'docker rm -f "$name" >/dev/null 2>&1' EXIT

fail() { echo "FALLO: $1" >&2; docker logs "$name" >&2; exit 1; }
status() { curl -s -o /dev/null -w '%{http_code}' "${@}"; }

for _ in $(seq 1 30); do
    [ "$(status http://127.0.0.1:59091/readyz)" = 200 ] && break
    sleep 1
done
[ "$(status http://127.0.0.1:59091/readyz)" = 200 ] || fail "/readyz no llegó a 200"
[ "$(status http://127.0.0.1:59091/livez)" = 200 ] || fail "/livez no responde 200"
[ "$(status http://127.0.0.1:58081/livez)" = 404 ] || fail "el puerto público no debe servir /livez"
[ "$(status http://127.0.0.1:58081/api/me)" = 401 ] || fail "/api/me sin sesión debe ser 401"
[ "$(status -H 'Authorization: Bearer basura' http://127.0.0.1:58081/api/me)" = 401 ] || fail "un token ilegible debe ser 401"

body="$(curl -s http://127.0.0.1:58081/api/me)"
case "$body" in
    *'"code":"NOT_AUTHENTICATED"'*'"traceId"'*) ;;
    *) fail "el 401 no es problem+json: $body" ;;
esac

metrics="$(curl -s http://127.0.0.1:59091/metrics)"
printf '%s' "$metrics" | grep -q 'http_server_requests_total{method="GET",route="/api/me",status="401"}' \
    || fail "/metrics no trae las peticiones con la plantilla de ruta"
printf '%s' "$metrics" | grep -q '^db_pool_max_connections ' || fail "/metrics no trae las métricas del pool"
[ "$(status http://127.0.0.1:58081/metrics)" = 404 ] || fail "el puerto público no debe servir /metrics"

docker exec "$name" /kaoru-api healthcheck || fail "el healthcheck del contenedor falló"
docker exec "$name" /kaoru-api prune-documents | grep -q '0 archivos revisados, 0 sin registro' \
    || fail "la limpieza de archivos sin registro no corrió"
migrations="$(ls "$(dirname "$0")/../src/migrations"/*.sql | wc -l | tr -d ' ')"
# Los logs se leen enteros antes de buscar: con `grep -q` en la tubería, docker recibe SIGPIPE y falla.
logs="$(docker logs "$name" 2>&1)"
printf '%s' "$logs" | grep -q "\"msg\":\"migraciones al día\",\"applied\":$migrations" || fail "no aplicó las $migrations migraciones al arrancar"

docker stop -t 25 "$name" >/dev/null
[ "$(docker inspect -f '{{.State.ExitCode}}' "$name")" = 0 ] || fail "no se apagó con código 0"
logs="$(docker logs "$name" 2>&1)"
printf '%s' "$logs" | grep -q 'kaoru-api detenido' || fail "no registró el apagado"

echo "Imagen kaoru-api: OK (arranca, migra, exige sesión y se apaga en orden)"

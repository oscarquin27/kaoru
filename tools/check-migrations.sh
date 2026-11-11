#!/usr/bin/env bash
# Comprueba las migraciones sobre un Postgres limpio y desechable:
# suben, bajan a cero, vuelven a subir, y `kaoru_app` no puede alterar lo inmutable.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
image="${POSTGRES_IMAGE:-postgres:17-alpine}"
goose="github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION:-v3.24.1}"
name="kaoru-check-migrations-$$"
port="${CHECK_PG_PORT:-55432}"

docker run -d --rm --name "$name" -e POSTGRES_PASSWORD=lab -e POSTGRES_DB=kaoru \
    -p "127.0.0.1:$port:5432" "$image" >/dev/null
trap 'docker stop "$name" >/dev/null' EXIT

until docker exec "$name" psql -U postgres -d kaoru -qc 'select 1' >/dev/null 2>&1; do sleep 1; done

psql_as() { docker exec -i -e PGPASSWORD=lab "$name" psql -h 127.0.0.1 -U "$1" -d kaoru -v ON_ERROR_STOP=1 -qAt "${@:2}"; }

psql_as postgres -v owner_password=lab -v app_password=lab <"$root/infra/postgres/init.sql"

run_goose() {
    (cd "$root/src/migrations" && go run "$goose" postgres \
        "postgres://kaoru_owner:lab@127.0.0.1:$port/kaoru?sslmode=disable" "$@")
}

run_goose up
run_goose down-to 0
run_goose up

psql_as kaoru_owner <<'SQL'
INSERT INTO app_user VALUES ('u1', 'lab', 'Lab', NULL, now(), now());
INSERT INTO audit_entry (id, occurred_at, actor_id, entity_type, entity_id, action, hash)
VALUES (gen_random_uuid(), now(), 'u1', 'requirement', gen_random_uuid(), 'create', repeat('a', 64));
SQL

# Cada sentencia debe fallar por permisos para `kaoru_app`.
for stmt in \
    "UPDATE audit_entry SET reason = 'x'" \
    "DELETE FROM audit_entry" \
    "TRUNCATE audit_entry" \
    "UPDATE test_run SET notes = 'x'" \
    "DELETE FROM document_version" \
    "DELETE FROM comment" \
    "CREATE TABLE intruso (id int)"; do
    if out="$(psql_as kaoru_app -c "$stmt" 2>&1)"; then
        echo "FALLO: kaoru_app pudo ejecutar: $stmt" >&2
        exit 1
    fi
    case "$out" in
        *"permission denied"*) ;;
        *) echo "FALLO: error inesperado en «$stmt»: $out" >&2; exit 1 ;;
    esac
done

# Y lo que sí debe poder hacer.
psql_as kaoru_app -c "SELECT count(*) FROM audit_entry" >/dev/null
psql_as kaoru_app -c "INSERT INTO code_sequence VALUES ('REQ', 2026, 1)" >/dev/null

echo "Migraciones: OK ($(psql_as kaoru_app -c "SELECT count(*) FROM workflow_state") estados, $(psql_as kaoru_app -c "SELECT count(*) FROM workflow_transition") transiciones, $(psql_as kaoru_app -c "SELECT count(*) FROM blocking_rule") reglas)"

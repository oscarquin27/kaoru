#!/usr/bin/env bash
# Comprueba las migraciones sobre el Postgres limpio de tools/with-postgres.sh:
# suben, bajan a cero, vuelven a subir, y `kaoru_app` no puede alterar lo inmutable.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
goose="github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION:-v3.24.1}"

psql_as() { docker exec -i -e PGPASSWORD=lab "$KAORU_TEST_PG" psql -h 127.0.0.1 -U "$1" -d kaoru -v ON_ERROR_STOP=1 -qAt "${@:2}"; }

run_goose() { (cd "$root/src/migrations" && go run "$goose" postgres "$KAORU_TEST_OWNER_URL" "$@" 2>&1 | grep -v ' OK ' || true); }

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
    "UPDATE checklist_signature SET comment = 'x'" \
    "DELETE FROM diagnostic_review" \
    "UPDATE smoke_test SET result = 'passed'" \
    "UPDATE conformance_report SET content = 'x'" \
    "DELETE FROM conformance_report" \
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

#!/usr/bin/env bash
# Prepara cada consulta de sqlc contra el Postgres limpio de tools/with-postgres.sh, ya migrado:
# una consulta que la base real no acepta falla aquí y no en producción (regla sqlc/db-prepare).
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
goose="github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION:-v3.24.1}"
(cd "$root/src/migrations" && go run "$goose" postgres "$KAORU_TEST_OWNER_URL" up >/dev/null 2>&1)

# La configuración de vet es la de siempre más la base y la regla; no se versiona
# para que `sqlc generate` siga sin necesitar una base.
config="$root/src/.sqlc-vet.yaml"
trap 'rm -f "$config"' EXIT
ruby -ryaml -e '
  cfg = YAML.load_file(ARGV[0])
  cfg["sql"].each do |block|
    block["database"] = { "uri" => ENV.fetch("KAORU_TEST_OWNER_URL") + "&search_path=kaoru" }
    block["rules"] = ["sqlc/db-prepare"]
  end
  File.write(ARGV[1], cfg.to_yaml)
' "$root/src/sqlc.yaml" "$config"

(cd "$root/src" && sqlc vet -f .sqlc-vet.yaml)
echo "sqlc vet: OK ($(grep -c 'name:' "$root"/src/internal/feature/*/adapter/postgres/queries/*.sql | awk -F: '{s+=$2} END {print s}') consultas preparadas contra Postgres)"

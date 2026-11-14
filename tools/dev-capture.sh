#!/usr/bin/env bash
# Graba respuestas REALES de la API del Kaoru de desarrollo en web/mocks/fixtures.json, para ver
# y probar el front sin levantar todo. Requiere tools/dev-up.sh y tools/dev-seed.sh.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
set -a; . "$root/infra/.env.example"; set +a
base="$KAORU_PUBLIC_URL"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
compose() { docker compose -p kaoru-dev -f "$root/infra/docker-compose.yml" --env-file "$root/infra/.env.example" "$@"; }
lab_password="$(cat "$root/infra/.lab-password")"
# shellcheck disable=SC1091
. "$root/tools/e2e-lib.sh"
login lab-admin

get() { api lab-admin GET "$1"; }
first() { get "$1" | json_code "$2"; }
chg="$(first '/api/changes?system=SYPAGO' CHG)"
req="$(get "/api/defects?severity=high" | grep -o '"requirement":"REQ-[0-9]*-[0-9]*"' | head -1 | cut -d'"' -f4)"
def="$(get "/api/defects?severity=high" | json_code DEF)"
doc="$(get "/api/documents?entityType=requirement&entityCode=$req" | json_code DOC)"
tc="$(get "/api/test-cases?requirement=$req" | json_code TC)"
uc="$(get "/api/requirements/$req/functional-cases" | json_code UC)"
run="$(get "/api/runs?requirement=$req" | json_code RUN)"
pkg="$(first /api/packages PKG)"
build="$(first /api/builds BUILD)"
env="$(first /api/environments ENV)"
dep="$(first /api/deployments DEP)"
comp="$(first /api/components COMP)"
chk="$(first /api/checklists CHK)"
diag="$(first /api/diagnostics DIAG)"
smoke="$(first /api/smoke-tests SMOKE)"
rpt="$(first /api/reports RPT)"

paths=(
    /api/me /api/users /api/systems
    /api/workflows/requirement /api/workflows/change /api/workflows/defect /api/workflows/package
    /api/components "/api/components/$comp" "/api/requirements/$req/components" "/api/test-cases/$tc/components"
    /api/builds "/api/builds/$build" /api/environments "/api/environments/$env"
    /api/deployments "/api/deployments/$dep" "/api/deployments?environment=$env"
    /api/packages "/api/packages/$pkg" "/api/packages/$pkg/regression"
    "/api/documents?entityType=package&entityCode=$pkg" "/api/comments?entityType=package&entityCode=$pkg" "/api/audit?entityCode=$pkg"
    /api/changes "/api/changes/$chg" "/api/requirements?change=$chg"
    /api/requirements "/api/requirements/$req" "/api/requirements/$req/overview"
    "/api/requirements/$req/links" "/api/requirements/$req/entry-contract" "/api/requirements/$req/risk"
    "/api/requirements/$req/functional-cases" "/api/functional-cases/$uc"
    /api/test-cases "/api/test-cases?requirement=$req" "/api/test-cases/$tc"
    /api/runs "/api/runs?requirement=$req" "/api/runs/$run" "/api/explorations?requirement=$req"
    /api/defects "/api/defects?requirement=$req" "/api/defects/$def"
    "/api/documents?entityType=requirement&entityCode=$req" "/api/documents/$doc"
    "/api/comments?entityType=requirement&entityCode=$req" "/api/comments?entityType=defect&entityCode=$def"
    /api/traceability/matrix "/api/search?q=l%C3%ADmite" "/api/search?q=$req"
    /api/audit "/api/audit?entityCode=$req" /api/audit/verification
    /api/admin/role-permissions
    '/api/saved-views?screen=requirements' '/api/saved-views?screen=changes' '/api/saved-views?screen=defects'
    '/api/saved-views?screen=matrix' '/api/saved-views?screen=test-cases' '/api/saved-views?screen=audit'
    /api/checklist-templates /api/checklists "/api/checklists/$chk" "/api/checklists?package=$pkg"
    "/api/documents?entityType=checklist&entityCode=$chk" "/api/audit?entityCode=$chk"
    /api/diagnostics "/api/diagnostics/$diag" "/api/diagnostics?package=$pkg" "/api/documents?entityType=diagnostic&entityCode=$diag"
    /api/smoke-tests "/api/smoke-tests/$smoke" "/api/smoke-tests?package=$pkg" "/api/documents?entityType=smoke_test&entityCode=$smoke"
    "/api/deployments?package=$pkg" "/api/environments?system=SYPAGO"
    /api/reports "/api/reports/$rpt" "/api/reports?package=$pkg" "/api/audit?entityCode=$rpt"
    /api/calendar /api/milestones '/api/milestones?status=pending'
    /api/metrics/coverage /api/metrics/defect-states /api/metrics/regression /api/metrics/escaped-defects
)
{
    printf '{\n  "_nota": "Respuestas reales de la API con datos de EJEMPLO. Se regeneran con tools/dev-capture.sh.",\n'
    printf '  "_ids": {"change": "%s", "requirement": "%s", "defect": "%s", "document": "%s", "testCase": "%s", "run": "%s", "package": "%s", "build": "%s", "environment": "%s", "deployment": "%s", "component": "%s", "checklist": "%s", "diagnostic": "%s", "smokeTest": "%s", "report": "%s"},\n' "$chg" "$req" "$def" "$doc" "$tc" "$run" "$pkg" "$build" "$env" "$dep" "$comp" "$chk" "$diag" "$smoke" "$rpt"
    sep=''
    for path in "${paths[@]}"; do
        body="$(get "$path")"
        [ "$(code)" = 200 ] || fail "GET $path respondió $(code)"
        printf '%s  "GET %s": %s' "$sep" "$path" "$body"
        sep=$',\n'
    done
    printf ',\n  "TEXT /api/documents/%s/versions/2/content": %s\n}\n' "$doc" \
        "$(curl -sk -b "$work/lab-admin.jar" "$base/api/documents/$doc/versions/2/content" | python3 -c 'import json, sys; print(json.dumps(sys.stdin.read()), end="")')"
} >"$root/web/mocks/fixtures.json"
python3 -c 'import json, sys; d = json.load(open(sys.argv[1], encoding="utf-8")); print(f"Grabadas {len(d) - 2} respuestas en {sys.argv[1]}")' "$root/web/mocks/fixtures.json"

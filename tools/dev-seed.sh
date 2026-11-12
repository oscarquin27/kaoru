#!/usr/bin/env bash
# Siembra datos de EJEMPLO (ficticios) en el Kaoru de desarrollo, usando la API con sesión real:
# dos CHG, cuatro REQ en distintos estados, RISK, UC, TC, un BUILD desplegado en un ambiente, RUN, DEF,
# enlaces, un documento y comentarios.
# Requiere tools/dev-up.sh. Cada corrida agrega otro juego de datos.
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

for user in lab-manager lab-qa lab-dev; do login "$user"; done

post() { # usuario ruta cuerpo → imprime la respuesta; falla si no es 2xx
    api "$1" POST "$2" "$3"
    case "$(code)" in 2*) ;; *) fail "POST $2 respondió $(code): $(cat "$work/body")" ;; esac
}
put() { api "$1" PUT "$2" "$3" >/dev/null; case "$(code)" in 2*) ;; *) fail "PUT $2 respondió $(code): $(cat "$work/body")" ;; esac; }
patch() { api "$1" PATCH "$2" "$3" >/dev/null; case "$(code)" in 2*) ;; *) fail "PATCH $2 respondió $(code): $(cat "$work/body")" ;; esac; }
move() { post "$1" "/api/$2/$3/transitions" "{\"to\":\"$4\",\"reason\":\"${5:-}\"}" >/dev/null; }
# El checksum de un BUILD no se repite: cada corrida registra el suyo.
checksum="$(printf 'build-de-ejemplo-%s' "$(date -u +%Y%m%dT%H%M%S)-$$" | shasum -a 256 | cut -d' ' -f1)"
ago() { date -u -v-"$1"H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "$1 hours ago" +%Y-%m-%dT%H:%M:%SZ; }
run() { # tc resultado horas-atrás → código del RUN, sobre el BUILD y el ambiente de SyPago
    post lab-qa /api/runs "{\"testCase\":\"$1\",\"build\":\"$build\",\"environment\":\"$env\",\"result\":\"$2\",\"executedAt\":\"$(ago "$3")\"}" | json_code RUN
}
# to_testing req: lleva un REQ de «Registrado» a «En pruebas».
to_testing() {
    move lab-qa requirements "$1" risk_assessed
    move lab-qa requirements "$1" test_design
    move lab-dev requirements "$1" in_development
    for item in ids components build_manifest install_windows install_linux unit_tests rollback test_data known_issues; do
        put lab-dev "/api/requirements/$1/entry-contract/$item" '{"note":""}'
    done
    move lab-dev requirements "$1" delivered_to_qa
    move lab-qa requirements "$1" in_testing
}

# --- CHG de SyPago (ruta ligera) ---
chg="$(post lab-manager /api/changes '{"title":"Límite diario en pagos móviles","description":"Datos de ejemplo.","system":"SYPAGO","windowStart":"2026-12-01T02:00:00Z","windowEnd":"2026-12-01T05:00:00Z"}' | json_code CHG)"
move lab-manager changes "$chg" approved
move lab-manager changes "$chg" in_progress

# REQ 1: se queda en pruebas, bloqueado por un TC fallido y un DEF de severidad alta.
req1="$(post lab-manager /api/requirements "{\"title\":\"Límite diario por ordenante\",\"description\":\"El acumulado del día no puede pasar el límite configurado.\",\"acceptanceCriteria\":\"- Rechaza el pago que supera el límite.\\n- Los pagos reversados no cuentan.\",\"system\":\"SYPAGO\",\"change\":\"$chg\",\"owner\":\"$(api lab-qa GET /api/me | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)\"}" | json_code REQ)"
put lab-qa "/api/requirements/$req1/risk" '{"level":"medium","justification":"Toca el cálculo de acumulados, con alternativa operativa.","characteristics":["functional_suitability","reliability"]}'
to_testing "$req1"
# El catálogo sobre el que se ejecuta: el BUILD que trae el REQ y el ambiente de línea base de SyPago.
# El nombre de un ambiente no se repite en su sistema: si ya existe de otra corrida, se reutiliza.
build="$(post lab-dev /api/builds "{\"system\":\"SYPAGO\",\"version\":\"2.14.0\",\"checksum\":\"$checksum\",\"requirements\":[\"$req1\"],\"migrations\":\"0042_limite_diario.sql\",\"dependencies\":\"libpagos 3.1.0\",\"notes\":\"Datos de ejemplo.\"}" | json_code BUILD)"
env="$(api lab-qa GET '/api/environments?system=SYPAGO&kind=baseline' | json_code ENV || true)"
[ -n "$env" ] || env="$(post lab-qa /api/environments '{"system":"SYPAGO","name":"QA-1 (línea base)","kind":"baseline","os":"linux","notes":"Datos de ejemplo."}' | json_code ENV)"
post lab-qa /api/deployments "{\"environment\":\"$env\",\"build\":\"$build\",\"result\":\"success\",\"notes\":\"Datos de ejemplo.\",\"executedAt\":\"$(ago 36)\"}" >/dev/null
uc1="$(post lab-qa "/api/requirements/$req1/functional-cases" '{"title":"Pago que supera el límite diario","description":"El ordenante ya acumuló parte del límite."}' | json_code UC)"
uc2="$(post lab-qa "/api/requirements/$req1/functional-cases" '{"title":"Pago reversado no cuenta para el acumulado"}' | json_code UC)"
tc() { post lab-qa /api/test-cases "{\"title\":\"$1\",\"system\":\"SYPAGO\",\"preconditions\":\"Ordenante con límite de 1.000.\",\"steps\":\"1. Acumular 900.\\n2. Pagar 200.\",\"expectedResult\":\"$2\",\"isCritical\":$3,\"functionalCases\":[\"$4\"]}" | json_code TC; }
tc1="$(tc 'Rechaza el pago que supera el límite' 'Pago rechazado con código de límite.' true "$uc1")"
tc2="$(tc 'Acepta el pago dentro del límite' 'Pago procesado.' false "$uc1")"
tc3="$(tc 'Un pago reversado libera el acumulado' 'El acumulado baja tras el reverso.' false "$uc2")"
run "$tc1" passed 30 >/dev/null
run "$tc2" passed 29 >/dev/null
bad="$(run "$tc3" failed 28)"
def="$(post lab-qa /api/defects "{\"title\":\"El reverso no libera el acumulado\",\"description\":\"Tras reversar un pago, el acumulado del día no baja.\",\"requirement\":\"$req1\",\"run\":\"$bad\",\"severity\":\"high\"}" | json_code DEF)"
low="$(post lab-qa /api/defects "{\"title\":\"Mensaje de rechazo con falta de ortografía\",\"description\":\"Dice «limite» sin tilde.\",\"requirement\":\"$req1\",\"run\":\"$bad\",\"severity\":\"low\"}" | json_code DEF)"
move lab-qa defects "$def" triaged
move lab-dev defects "$def" in_fix
move lab-qa defects "$low" triaged
move lab-manager defects "$low" deferred "Cosmético; va al siguiente paquete."
post lab-qa "/api/requirements/$req1/links" '{"url":"https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz0123456789/edit#gid=1842&range=A1:F40","label":"Especificación SyPago · pestaña Pagos · A1:F40","isOfficial":true}' >/dev/null
post lab-dev "/api/requirements/$req1/links" '{"url":"https://youtrack.lab.invalid/issue/SY-112","label":"SY-112 en YouTrack"}' >/dev/null
printf '# Límite diario en pagos móviles\n\nInstantánea de EJEMPLO de la especificación.\n\n## Regla\n\n- El acumulado del día suma los pagos aprobados.\n- Los pagos reversados no cuentan.\n' >"$work/especificacion.md"
curl -sk -b "$work/lab-qa.jar" -H "Origin: $base" -o "$work/body" -F 'title=Especificación funcional: límite diario' -F kind=snapshot \
    -F entityType=requirement -F "entityCode=$req1" -F "file=@$work/especificacion.md" "$base/api/documents" >/dev/null
printf '# Límite diario en pagos móviles\n\nInstantánea de EJEMPLO, versión 2.\n\n## Regla\n\n- El acumulado del día suma los pagos aprobados.\n- Los pagos reversados **no** cuentan.\n- El límite se reinicia a medianoche.\n' >"$work/especificacion.md"
doc="$(json_code DOC <"$work/body")"
curl -sk -b "$work/lab-qa.jar" -H "Origin: $base" -o /dev/null -F "file=@$work/especificacion.md" "$base/api/documents/$doc/versions"
post lab-dev /api/comments "{\"entityType\":\"defect\",\"entityCode\":\"$def\",\"body\":\"Reproducido en QA-1. El reverso no pasa por el cálculo del acumulado.\"}" >/dev/null
post lab-qa /api/comments "{\"entityType\":\"requirement\",\"entityCode\":\"$req1\",\"body\":\"Queda en espera de la corrección de $def.\"}" >/dev/null
post lab-qa /api/explorations "{\"requirement\":\"$req1\",\"build\":\"$build\",\"environment\":\"$env\",\"manualChanges\":\"Reloj del servidor adelantado un día.\",\"findings\":\"El límite no se reinicia a medianoche.\",\"executedAt\":\"$(ago 20)\"}" >/dev/null

# REQ 2: aprobado por QA.
req2="$(post lab-manager /api/requirements "{\"title\":\"Aviso al acercarse al límite\",\"acceptanceCriteria\":\"Avisa al 80 % del límite.\",\"system\":\"SYPAGO\",\"change\":\"$chg\"}" | json_code REQ)"
put lab-qa "/api/requirements/$req2/risk" '{"level":"low","justification":"Solo agrega un aviso."}'
to_testing "$req2"
uc3="$(post lab-qa "/api/requirements/$req2/functional-cases" '{"title":"Aviso al 80 % del límite"}' | json_code UC)"
tc4="$(tc 'Muestra el aviso al 80 %' 'Aviso visible.' false "$uc3")"
run "$tc4" passed 10 >/dev/null
move lab-qa requirements "$req2" approved_qa

# REQ 3: recién registrado, sin CHG.
post lab-manager /api/requirements '{"title":"Límite configurable por tipo de cuenta","system":"SYPAGO"}' >/dev/null

# --- CHG de SIMF (ruta alta) ---
chg2="$(post lab-manager /api/changes '{"title":"Certificación de mensajería SIMF 4.2","description":"Datos de ejemplo.","system":"SIMF"}' | json_code CHG)"
req4="$(post lab-manager /api/requirements "{\"title\":\"Validación de firma en mensajes entrantes\",\"system\":\"SIMF\",\"change\":\"$chg2\"}" | json_code REQ)"
put lab-qa "/api/requirements/$req4/risk" '{"level":"high","justification":"Alcance nacional; afecta la seguridad de los mensajes.","characteristics":["security","reliability","compatibility"]}'
move lab-qa requirements "$req4" risk_assessed

echo "Datos de ejemplo: $chg ($req1 bloqueado por $def, $req2 aprobado), $chg2 ($req4), $build en $env, documento $doc"

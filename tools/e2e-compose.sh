#!/usr/bin/env bash
# Prueba de punta a punta del despliegue completo: levanta el Compose desde cero, crea dos
# usuarios de laboratorio en Keycloak, inicia sesión como lo haría Chrome y usa la API a
# través de Nginx y oauth2-proxy. Al terminar apaga todo y borra los volúmenes.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
env_file="$root/infra/.env.example"
compose() { docker compose -p kaoru-e2e -f "$root/infra/docker-compose.yml" --env-file "$env_file" "$@"; }
# shellcheck disable=SC1090
set -a; . "$env_file"; set +a
base="$KAORU_PUBLIC_URL"
work="$(mktemp -d)"
# Contraseña de laboratorio generada en cada corrida; cumple la política del realm y no se imprime.
lab_password="Lab-$(openssl rand -hex 8)-9a"

cleanup() {
    status=$?
    if [ $status -ne 0 ]; then
        echo "--- últimos logs ---" >&2
        compose logs --tail 25 nginx oauth2-proxy kaoru-api 2>&1 | tail -80 >&2 || true
    fi
    compose down -v --remove-orphans >/dev/null 2>&1 || true
    rm -rf "$work"
    exit $status
}
trap cleanup EXIT

# shellcheck disable=SC1091
. "$root/tools/e2e-lib.sh"

"$root/tools/make-lab-cert.sh" localhost >/dev/null
docker build -q -f "$root/infra/api/Dockerfile" -t kaoru-api:dev "$root" >/dev/null
compose down -v --remove-orphans >/dev/null 2>&1 || true
require_dev_down
compose up -d --wait --wait-timeout 300 >/dev/null 2>&1 || fail "el Compose no llegó a estar sano"

# Usuarios de laboratorio, creados con la herramienta de administración de Keycloak.
kcadm_login
new_user lab-manager Carla Mora change-manager
new_user lab-qa Ana Rojas qa

# Sin sesión: la API responde 401 y el front redirige al inicio de sesión.
[ "$(status "$base/api/me")" = 401 ] || fail "/api/me sin sesión debe ser 401"
[ "$(status -H 'Authorization: Bearer falsificado' "$base/api/me")" = 401 ] || fail "un token puesto por el navegador no debe valer"
[ "$(status "$base/")" = 302 ] || fail "/ sin sesión debe redirigir al inicio de sesión"
[ "$(status "$base/auth/admin/")" = 404 ] || fail "la consola de administración de Keycloak no debe estar publicada"
curl -skI "$base/healthz" | grep -qi 'strict-transport-security' || fail "faltan las cabeceras de seguridad"

# Inicio de sesión como lo hace el navegador (ver login en e2e-lib.sh).
login lab-manager
login lab-qa

# El navegador solo tiene cookies. La de la aplicación va cifrada: no trae un token legible.
# (Keycloak guarda además su propia cookie de sesión, un JWT interno que no sirve contra la API.)
grep '_oauth2_proxy' "$work/lab-qa.jar" >"$work/session-cookie" || fail "falta la cookie de sesión de la aplicación"
if grep -Eq 'eyJ[A-Za-z0-9_-]{20,}\.eyJ' "$work/session-cookie"; then fail "la cookie de sesión no debe traer un JWT legible"; fi
grep -q '^#HttpOnly_' "$work/session-cookie" || fail "la cookie de sesión debe ser HttpOnly"
keycloak_cookie="$(grep 'KEYCLOAK_IDENTITY' "$work/lab-qa.jar" | head -1 | awk '{print $NF}')"
[ -n "$keycloak_cookie" ] || fail "no se encontró la cookie de sesión de Keycloak"
[ "$(status -H "Authorization: Bearer $keycloak_cookie" "$base/api/me")" = 401 ] || fail "la cookie de Keycloak no debe servir como token de la API"

me="$(api lab-qa GET /api/me)"
[ "$(code)" = 200 ] || fail "/api/me con sesión respondió $(code): $me"
case "$me" in
    *'"username":"lab-qa"'*'"displayName":"Ana Rojas"'*'"roles":["qa"]'*) ;;
    *) fail "perfil inesperado: $me" ;;
esac
case "$me" in *'"run.create"'*) ;; *) fail "qa debe tener run.create: $me" ;; esac

# Permisos y CSRF a través del perímetro real.
api lab-qa POST /api/changes '{"title":"Cambio e2e","system":"SYPAGO"}' >/dev/null
[ "$(code)" = 403 ] || fail "qa no debe poder crear un CHG ($(code))"
[ "$(curl -sk -b "$work/lab-manager.jar" -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' \
    -d '{"title":"x","system":"SYPAGO"}' "$base/api/changes")" = 403 ] || fail "una escritura sin Origin debe rechazarse"
change="$(api lab-manager POST /api/changes '{"title":"Cambio e2e","system":"SYPAGO"}')"
[ "$(code)" = 201 ] || fail "change-manager debía crear el CHG ($(code)): $change"
chg="$(printf '%s' "$change" | json_code CHG)"
case "$change" in *'"displayName":"Carla Mora"'*) ;; *) fail "el CHG debe decir quién lo creó: $change" ;; esac

# El login lleva el tema de Kaoru, y sus recursos salen del propio servidor.
page="$(curl -skL "$base/")"
css="$(printf '%s' "$page" | grep -o '/auth/resources/[a-z0-9]*/login/kaoru/css/kaoru.css' | head -1)"
[ -n "$css" ] || fail "la página de login no usa el tema de Kaoru"
[ "$(curl -sk -o /dev/null -w '%{http_code}' "$base$css")" = 200 ] || fail "la hoja de estilos del tema no se sirve"
[ "$(curl -sk -o /dev/null -w '%{http_code}' "$base${css%css/kaoru.css}img/kaoru.svg")" = 200 ] || fail "la marca del tema no se sirve"
if printf '%s' "$page" | grep -Eo '(src|href)="https?://[^"]*"' | grep -v "$base" | grep -q .; then
    fail "la página de login pide recursos a otro servidor"
fi

# Un recorrido corto con las dos sesiones: REQ, documento adjunto y auditoría.
req="$(api lab-manager POST /api/requirements "{\"title\":\"REQ e2e\",\"system\":\"SYPAGO\",\"change\":\"$chg\"}" | json_code REQ)"
[ -n "$req" ] || fail "no se creó el REQ"
printf '%%PDF-1.7\nprueba e2e\n' >"$work/spec.pdf"
upload="$(curl -sk -b "$work/lab-qa.jar" -H "Origin: $base" -o "$work/body" -w '%{http_code}' \
    -F title=Especificación -F kind=specification -F entityType=requirement -F "entityCode=$req" \
    -F "file=@$work/spec.pdf" "$base/api/documents")"
[ "$upload" = 201 ] || fail "la subida del documento respondió $upload: $(cat "$work/body")"
doc="$(json_code DOC <"$work/body")"
curl -sk -b "$work/lab-qa.jar" -o "$work/descarga.pdf" "$base/api/documents/$doc/versions/1/content"
cmp -s "$work/spec.pdf" "$work/descarga.pdf" || fail "el documento descargado no es el que se subió"
# El visor incrusta el contenido: solo el propio Kaoru puede enmarcarlo, y el archivo va aislado.
content="$base/api/documents/$doc/versions/1/content"
headers="$(curl -sk -b "$work/lab-qa.jar" -o /dev/null -D - "$content" | tr -d '\r')"
printf '%s' "$headers" | grep -qi "^content-security-policy: frame-ancestors 'self'$" || fail "el contenido de un documento debe poder incrustarse solo desde Kaoru"
printf '%s' "$headers" | grep -qi "^content-security-policy: sandbox" || fail "el contenido de un documento debe servirse aislado"
printf '%s' "$headers" | grep -qi "^x-frame-options: SAMEORIGIN$" || fail "X-Frame-Options del contenido debe ser SAMEORIGIN"
[ "$(curl -sk -o /dev/null -w '%{http_code}' "$content")" = 401 ] || fail "el contenido de un documento exige sesión"
[ "$(curl -sk -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer falso' "$content")" = 401 ] || fail "el contenido no acepta un token puesto a mano"
curl -sk -b "$work/lab-qa.jar" -o /dev/null -D - "$base/api/documents/$doc" | tr -d '\r' | grep -qi "^x-frame-options: DENY$" \
    || fail "el resto de la API no debe poder enmarcarse"
# Sin el servicio opcional de Office, la vista previa convertida no existe.
[ "$(curl -sk -b "$work/lab-qa.jar" -o /dev/null -w '%{http_code}' "$base/api/documents/$doc/versions/1/preview")" = 415 ] \
    || fail "sin conversor, la vista previa debe responder 415"
overview="$(api lab-qa GET "/api/requirements/$req/overview")"
case "$overview" in *'"status":"current"'*'"missing"'*) ;; *) fail "ficha consolidada inesperada: $overview" ;; esac
api lab-qa GET /api/audit/verification | grep -q '"intact":true' || fail "la cadena de auditoría debe estar íntegra"

# El id de traza que genera Nginx vuelve en la respuesta.
curl -skI -b "$work/lab-qa.jar" "$base/api/me" | grep -qi '^x-request-id:' || fail "falta X-Request-Id en la respuesta"

# Opcional (tarda 6 minutos): la sesión sigue viva cuando vence el token de acceso, que dura 5.
if [ "${E2E_TOKEN_REFRESH:-0}" = 1 ]; then
    sleep 330
    curl -sk -b "$work/lab-manager.jar" -c "$work/lab-manager.jar" -o "$work/body" -w '%{http_code}' "$base/api/me" >"$work/code"
    [ "$(code)" = 200 ] || fail "tras vencer el token, la sesión debía renovarse sola ($(code)): $(cat "$work/body")"
    api lab-manager POST /api/changes '{"title":"Tras renovar el token","system":"SIMF"}' >/dev/null
    [ "$(code)" = 201 ] || fail "tras renovar el token no se pudo escribir ($(code))"
    echo "Renovación del token: OK"
fi

# Cierre de sesión: se cierra también en Keycloak y la cookie deja de valer.
curl -skL -b "$work/lab-qa.jar" -c "$work/lab-qa.jar" -o /dev/null "$base/oauth2/sign_out"
api lab-qa GET /api/me >/dev/null
[ "$(code)" = 401 ] || fail "tras cerrar sesión /api/me debe ser 401 ($(code))"
[ "$(curl -skL -b "$work/lab-qa.jar" -c "$work/lab-qa.jar" -o "$work/again.html" -w '%{http_code}' "$base/")" = 200 ] || fail "/ tras cerrar sesión"
grep -q 'Sesión iniciada' "$work/again.html" && fail "tras cerrar sesión no debe entrar sin contraseña"

echo "Compose de punta a punta: OK (sesión real con Keycloak, permisos, CSRF, documento y cierre de sesión)"

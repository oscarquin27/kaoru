#!/usr/bin/env bash
# Comprueba que el front compilado (web/dist) funciona detrás del perímetro real: se sirve solo
# con sesión, sus archivos cargan con la política de seguridad de Nginx y las rutas del front
# abren directo. No reemplaza mirarlo en Chrome: comprueba lo que se puede sin un navegador.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
dist="$root/web/dist"
[ -f "$dist/index.html" ] || { echo "FALLO: falta $dist/index.html. Compila el front: npm --prefix web run build" >&2; exit 1; }

env_file="$root/infra/.env.example"
export KAORU_WEB_DIR="$dist"
compose() { docker compose -p kaoru-e2e -f "$root/infra/docker-compose.yml" --env-file "$env_file" "$@"; }
# shellcheck disable=SC1090
set -a; . "$env_file"; set +a
base="$KAORU_PUBLIC_URL"
work="$(mktemp -d)"
lab_password="Lab-$(openssl rand -hex 8)-9a"
export LANDING_MARK='id="root"'
# shellcheck disable=SC1091
. "$root/tools/e2e-lib.sh"

cleanup() {
    status=$?
    compose down -v --remove-orphans >/dev/null 2>&1 || true
    rm -rf "$work"
    exit $status
}
trap cleanup EXIT

# El build no puede traer scripts en línea ni recursos de otros orígenes: la política los bloquearía.
if grep -Eq '<script(>| [^>]*>)[^<]+' "$dist/index.html" && ! grep -Eq '<script[^>]* src=' "$dist/index.html"; then
    fail "index.html trae un script en línea"
fi
if grep -Eoq '(src|href)="https?://' "$dist/index.html"; then fail "index.html carga recursos de otro origen"; fi
if grep -rEl 'https?://(fonts\.googleapis|fonts\.gstatic|cdn\.|unpkg\.|cdnjs\.)' "$dist" >/dev/null 2>&1; then
    fail "el build referencia un CDN"
fi

"$root/tools/make-lab-cert.sh" localhost >/dev/null
docker build -q -f "$root/infra/api/Dockerfile" -t kaoru-api:dev "$root" >/dev/null
compose down -v --remove-orphans >/dev/null 2>&1 || true
require_dev_down
compose up -d --wait --wait-timeout 300 >/dev/null 2>&1 || fail "el Compose no llegó a estar sano"
kcadm_login
new_user lab-qa Ana Rojas qa

# Sin sesión no se entrega nada del front.
assets="$(grep -Eo '(src|href)="/[^"]+\.(js|css)"' "$dist/index.html" | cut -d'"' -f2)"
[ -n "$assets" ] || fail "index.html no referencia ningún archivo .js o .css"
for asset in $assets; do
    [ "$(status "$base$asset")" = 302 ] || fail "$asset se entregó sin sesión"
done

login lab-qa

# Con sesión: la página, sus archivos y las rutas del front.
for asset in $assets; do
    type="$(curl -sk -b "$work/lab-qa.jar" -o /dev/null -w '%{http_code} %{content_type}' "$base$asset")"
    case "$asset:$type" in
        *.js:200\ *javascript*|*.css:200\ text/css*) ;;
        *) fail "$asset respondió «$type»" ;;
    esac
done
for route in / /requerimientos /requerimientos/REQ-2026-0001 /pruebas /defectos /produccion /produccion/CHK-2026-0001 /informes /informes/RPT-2026-0001 /calendario /planificacion /planificacion/actividades /metricas /metricas/escapados /administracion/auditoria; do
    curl -sk -b "$work/lab-qa.jar" -o "$work/page.html" -w '%{http_code}' "$base$route" >"$work/code"
    [ "$(code)" = 200 ] || fail "la ruta $route respondió $(code)"
    cmp -s "$dist/index.html" "$work/page.html" || fail "la ruta $route no entregó el front"
done
curl -skI -b "$work/lab-qa.jar" "$base/" | grep -qi "content-security-policy: default-src 'self'" || fail "falta la política de seguridad de contenido"

# El front y la API comparten origen y sesión.
me="$(api lab-qa GET /api/me)"
case "$me" in *'"username":"lab-qa"'*) ;; *) fail "/api/me desde la sesión del front: $me" ;; esac

echo "Front detrás del perímetro: OK (solo con sesión, $(echo "$assets" | wc -l | tr -d ' ') archivos, rutas directas y política de seguridad)"

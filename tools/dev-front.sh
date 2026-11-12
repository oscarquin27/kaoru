#!/usr/bin/env bash
# Sirve el front en desarrollo (http://localhost:5173, con recarga al guardar) contra la API REAL
# del Kaoru de desarrollo, con la sesión de un usuario de laboratorio. Requiere tools/dev-up.sh.
#   tools/dev-front.sh [usuario]     por omisión, lab-admin
# La sesión la pone el servidor de desarrollo, no el navegador: no hay login ni logout en pantalla.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
user="${1:-lab-admin}"
# shellcheck disable=SC1091
set -a; . "$root/infra/.env.example"; set +a
base="$KAORU_PUBLIC_URL"
work="$(mktemp -d)"
lab_password="$(cat "$root/infra/.lab-password")"
# shellcheck disable=SC1091
. "$root/tools/e2e-lib.sh"
login "$user"

# La cookie de sesión de oauth2-proxy puede venir partida en varias.
KAORU_DEV_COOKIE="$(awk '$6 ~ /^_oauth2_proxy/ { printf "%s=%s; ", $6, $7 }' "$work/$user.jar")"
rm -rf "$work"
[ -n "$KAORU_DEV_COOKIE" ] || fail "no quedó cookie de sesión para $user"
export KAORU_DEV_COOKIE KAORU_DEV_API="$base"
# El servidor de desarrollo confía en el certificado de laboratorio, no en cualquiera.
export NODE_EXTRA_CA_CERTS="$root/infra/tls/tls.crt"
echo "Front de desarrollo como $user contra $base"
exec npm run dev --prefix "$root/web"

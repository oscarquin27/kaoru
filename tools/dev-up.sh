#!/usr/bin/env bash
# Levanta Kaoru completo para desarrollo en https://localhost:8443, con seis usuarios de
# laboratorio (uno por rol). Los datos se conservan entre corridas; `tools/dev-down.sh -v` los borra.
#   KAORU_WEB_DIR=/ruta/a/web/dist tools/dev-up.sh   → sirve el front compilado
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
env_file="$root/infra/.env.example"
export KAORU_WEB_DIR="${KAORU_WEB_DIR:-$root/infra/web}"
compose() { docker compose -p kaoru-dev -f "$root/infra/docker-compose.yml" --env-file "$env_file" "$@"; }
# shellcheck disable=SC1090
set -a; . "$env_file"; set +a
base="$KAORU_PUBLIC_URL"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
# shellcheck disable=SC1091
. "$root/tools/e2e-lib.sh"

# La contraseña de laboratorio se genera una vez y queda en un archivo que no se versiona.
password_file="$root/infra/.lab-password"
[ -f "$password_file" ] || printf 'Lab-%s-9a\n' "$(openssl rand -hex 8)" >"$password_file"
lab_password="$(cat "$password_file")"

"$root/tools/make-lab-cert.sh" localhost >/dev/null
docker build -q -f "$root/infra/api/Dockerfile" -t kaoru-api:dev "$root" >/dev/null
compose up -d --wait --wait-timeout 300 >/dev/null 2>&1 || fail "el Compose no llegó a estar sano"
# Keycloak guarda una copia comprimida de cada archivo del tema la primera vez que se pide y no la
# renueva si el archivo cambia: tras actualizar, el login seguiría con la marca anterior. Se borra
# en cada arranque; Keycloak la rehace sola, sin reiniciarse.
compose exec -T keycloak rm -rf /opt/keycloak/data/tmp/kc-gzip-cache || fail "no se pudo vaciar la caché del tema de Keycloak"

kcadm_login
# Los usuarios se crean una sola vez: los datos se conservan entre corridas.
if ! kcadm get users -r kaoru -q username=lab-qa --fields username 2>/dev/null | grep -q 'lab-qa'; then
    new_user lab-manager Carla Mora change-manager
    new_user lab-qa Ana Rojas qa
    new_user lab-dev Juan Díaz dev
    new_user lab-admin Marta León qa-admin
    new_user lab-ops Luis Paz ops
    new_user lab-auditor Vera Soto auditor
fi

echo "Kaoru de desarrollo: $base"
echo "Usuarios: lab-manager, lab-qa, lab-dev, lab-admin, lab-ops, lab-auditor"
echo "Contraseña de laboratorio: en $password_file"

# Común a install.sh, backup.sh y restore.sh. Se incluye con `.`; no se ejecuta solo.
# Variables que se pueden cambiar: KAORU_PROJECT (nombre del proyecto de Compose) y
# KAORU_ENV_FILE (archivo de entorno).
here="$(cd "$(dirname "$0")" && pwd)"
project="${KAORU_PROJECT:-kaoru}"
env_file="${KAORU_ENV_FILE:-$here/.env}"

compose() { docker compose -p "$project" -f "$here/docker-compose.yml" --env-file "$env_file" "$@"; }

die() { echo "ERROR: $1" >&2; exit 1; }

[ -f "$env_file" ] || die "falta $env_file. Copia .env.example a .env y cambia las contraseñas."

# La imagen de Postgres ya está en el servidor: sirve también para copiar el volumen de documentos.
helper_image() { compose config --images | grep '^postgres' | head -1; }

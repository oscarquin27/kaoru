#!/usr/bin/env bash
# Arma el paquete para instalar Kaoru en un servidor sin internet: las imágenes (docker save),
# el Compose, la configuración y los scripts de instalación, respaldo y restauración.
# Se ejecuta donde hay internet. Deja dist/kaoru-offline-<versión>.tar.gz
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
version="$(git -C "$root" describe --always --dirty)"
name="kaoru-offline-$version"
stage="$root/dist/$name"
rm -rf "$stage" && mkdir -p "$stage"

api_image="kaoru-api:$version"
docker build -q -f "$root/infra/api/Dockerfile" -t "$api_image" "$root" >/dev/null

cp -R "$root/infra/docker-compose.yml" "$root/infra/.env.example" "$root/infra/nginx" "$root/infra/postgres" \
    "$root/infra/keycloak" "$root/infra/web" "$root/infra/lib.sh" "$root/infra/install.sh" \
    "$root/infra/backup.sh" "$root/infra/restore.sh" "$stage/"
mkdir -p "$stage/tls"
cp "$root/docs/instalacion.md" "$stage/LEEME.md"
# El paquete usa la imagen de esta versión, no la de desarrollo.
printf '\n# Imagen de kaoru-api incluida en este paquete.\nKAORU_API_IMAGE=%s\n' "$api_image" >>"$stage/.env.example"

# `docker load` no conserva el digest en todos los motores: dentro del paquete las imágenes se
# referencian por tag. El digest ya se verificó aquí, al bajarlas; la integridad del paquete
# la da la suma de images.tar.gz.
sed -E -i.bak 's/@sha256:[0-9a-f]{64}//' "$stage/docker-compose.yml" && rm "$stage/docker-compose.yml.bak"

images="$(docker compose -f "$stage/docker-compose.yml" --env-file "$stage/.env.example" config --images | sort -u)"
# shellcheck disable=SC2086
docker save $images | gzip >"$stage/images.tar.gz"
(cd "$stage" && (sha256sum images.tar.gz >SHA256SUMS 2>/dev/null || shasum -a 256 images.tar.gz >SHA256SUMS))

tar czf "$root/dist/$name.tar.gz" -C "$root/dist" "$name"
rm -rf "$stage"
echo "$root/dist/$name.tar.gz"

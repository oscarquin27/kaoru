#!/usr/bin/env sh
# Restaura un respaldo hecho con backup.sh. REEMPLAZA lo que haya: las dos bases y los documentos.
#   ./restore.sh /ruta/de/respaldos/kaoru-AAAAMMDD-HHMMSS
# Sirve en un servidor recién instalado o sobre una instalación existente.
set -eu
. "$(dirname "$0")/lib.sh"

[ $# -eq 1 ] || die "uso: $0 <directorio del respaldo>"
backup="$(cd "$1" && pwd)"
for f in kaoru.dump keycloak.dump documents.tar.gz SHA256SUMS; do
    [ -f "$backup/$f" ] || die "falta $f en $backup."
done
(cd "$backup" && (sha256sum -c SHA256SUMS >/dev/null 2>&1 || shasum -a 256 -c SHA256SUMS >/dev/null 2>&1)) \
    || die "el respaldo está dañado: las sumas de comprobación no coinciden."

# Solo Postgres arriba: nada debe escribir mientras se restaura.
compose up -d --wait postgres
compose stop nginx oauth2-proxy kaoru-api keycloak >/dev/null 2>&1 || true

for db in kaoru keycloak; do
    compose exec -T postgres pg_restore -U postgres -d "$db" --clean --if-exists --exit-on-error <"$backup/$db.dump"
done

docker run --rm -v "${project}_documents:/data" -v "$backup:/backup:ro" "$(helper_image)" \
    sh -c 'find /data -mindepth 1 -delete && tar xzf /backup/documents.tar.gz -C /data && chown -R 65532:65532 /data'

compose up -d --wait --wait-timeout 300
echo "Respaldo restaurado y Kaoru arriba."

#!/usr/bin/env sh
# Respalda Kaoru en un directorio: la base de la aplicación, la de Keycloak y los documentos.
#   ./backup.sh /ruta/de/respaldos        → crea /ruta/de/respaldos/kaoru-AAAAMMDD-HHMMSS/
# Se puede ejecutar con el sistema en uso.
set -eu
. "$(dirname "$0")/lib.sh"

[ $# -eq 1 ] || die "uso: $0 <directorio de respaldos>"
out="$1/kaoru-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$out"

for db in kaoru keycloak; do
    compose exec -T postgres pg_dump -U postgres -Fc "$db" >"$out/$db.dump"
    [ -s "$out/$db.dump" ] || die "el respaldo de la base $db salió vacío."
done

docker run --rm -v "${project}_documents:/data:ro" -v "$out:/backup" "$(helper_image)" \
    tar czf /backup/documents.tar.gz -C /data .

# Suma de comprobación de cada archivo, para verificar el respaldo antes de restaurarlo.
(cd "$out" && sha256sum kaoru.dump keycloak.dump documents.tar.gz >SHA256SUMS 2>/dev/null || shasum -a 256 kaoru.dump keycloak.dump documents.tar.gz >SHA256SUMS)
echo "$out"

#!/usr/bin/env sh
# Instala o actualiza Kaoru en un servidor sin internet, a partir del paquete offline.
#   1. Copia .env.example a .env y cambia todas las contraseñas.
#   2. Pon el certificado del servidor en tls/tls.crt y tls/tls.key.
#   3. Ejecuta ./install.sh
set -eu
. "$(dirname "$0")/lib.sh"

[ -f "$here/tls/tls.crt" ] && [ -f "$here/tls/tls.key" ] || die "falta el certificado en $here/tls/ (tls.crt y tls.key)."
if grep -q 'lab-' "$env_file"; then
    echo "AVISO: $env_file todavía tiene valores de laboratorio. No lo uses así fuera de una prueba." >&2
fi

if [ -f "$here/images.tar.gz" ]; then
    (cd "$here" && (sha256sum -c SHA256SUMS >/dev/null 2>&1 || shasum -a 256 -c SHA256SUMS >/dev/null 2>&1)) \
        || die "images.tar.gz está dañado: su suma de comprobación no coincide."
    echo "Cargando imágenes…"
    gunzip -c "$here/images.tar.gz" | docker load >/dev/null
fi
echo "Levantando Kaoru…"
compose up -d --wait --wait-timeout 300
# Nginx monta kaoru.conf como archivo suelto: al actualizar, el paquete nuevo lo reemplaza y el
# contenedor seguiría viendo el anterior. Se reinicia para que tome el de esta versión.
compose restart nginx >/dev/null
compose up -d --wait --wait-timeout 60 nginx
# Keycloak guarda una copia comprimida de cada archivo del tema la primera vez que se pide y no la
# renueva si el archivo cambia: tras actualizar, el login seguiría con la marca anterior. Se borra
# en cada arranque; Keycloak la rehace sola, sin reiniciarse.
compose exec -T keycloak rm -rf /opt/keycloak/data/tmp/kc-gzip-cache
echo "Kaoru está arriba. Ábrelo en Chrome en la dirección de KAORU_PUBLIC_URL."

#!/usr/bin/env sh
# Crea el primer usuario de Kaoru (rol qa-admin) si todavía no existe, con una contraseña al azar
# que queda en .admin-password, legible solo por root. Lo llama infra/contabo/deploy.yml:
#   ./crear-admin.sh <usuario>
# Los demás usuarios se crean en la consola de Keycloak (docs/instalacion.md, «Crear los usuarios»).
set -eu
. "$(dirname "$0")/lib.sh"
set -a; . "$env_file"; set +a
user="$1"

kcadm() { compose exec -T keycloak /opt/keycloak/bin/kcadm.sh "$@" --config /tmp/kcadm.config; }
kcadm config credentials --server http://localhost:8080/auth --realm master \
    --user "$KEYCLOAK_ADMIN_USER" --password "$KEYCLOAK_ADMIN_PASSWORD" >/dev/null 2>&1 \
    || die "no se pudo entrar a la administración de Keycloak"

if kcadm get users -r kaoru -q "username=$user" -q exact=true --fields username | grep -q "\"$user\""; then
    echo "ya existe"
    exit 0
fi

# La política del realm pide 12 caracteres con mayúscula, minúscula y dígito.
password="Kaoru-$(openssl rand -hex 10)"
kcadm create users -r kaoru -s "username=$user" -s enabled=true \
    -s firstName=Administración -s lastName=Kaoru >/dev/null || die "no se pudo crear $user"
kcadm set-password -r kaoru --username "$user" --new-password "$password" >/dev/null || die "no se pudo poner la contraseña"
kcadm add-roles -r kaoru --uusername "$user" --rolename qa-admin >/dev/null || die "no se pudo dar el rol qa-admin"
(umask 077 && printf '%s\n' "$password" >"$here/.admin-password")
echo "creado"

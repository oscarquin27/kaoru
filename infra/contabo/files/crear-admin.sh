#!/usr/bin/env sh
# Crea el primer usuario de Kaoru si todavía no existe, con una contraseña al azar que queda en
# .admin-password, legible solo por root. Lleva qa-admin y change-manager: es quien carga los CHG
# y los REQ mientras no haya más usuarios. Lo llama infra/contabo/deploy.yml:
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

roles="qa-admin change-manager"
add_roles() {
    for role in $roles; do
        kcadm add-roles -r kaoru --uusername "$user" --rolename "$role" >/dev/null || die "no se pudo dar el rol $role"
    done
}

# A un usuario que ya existe solo se le aseguran los roles: la contraseña no se toca.
if kcadm get users -r kaoru -q "username=$user" -q exact=true --fields username | grep -q "\"$user\""; then
    add_roles
    echo "ya existe"
    exit 0
fi

# El perfil de usuario del realm exige correo: sin él, Keycloak lo pide al entrar. Kaoru no envía
# correos, así que va uno de un dominio reservado (.invalid).
# La política del realm pide 12 caracteres con mayúscula, minúscula y dígito.
password="Kaoru-$(openssl rand -hex 10)"
kcadm create users -r kaoru -s "username=$user" -s enabled=true \
    -s firstName=Administración -s lastName=Kaoru \
    -s "email=$user@kaoru.invalid" -s emailVerified=true >/dev/null || die "no se pudo crear $user"
kcadm set-password -r kaoru --username "$user" --new-password "$password" >/dev/null || die "no se pudo poner la contraseña"
add_roles
(umask 077 && printf '%s\n' "$password" >"$here/.admin-password")
echo "creado"

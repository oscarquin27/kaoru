# Funciones comunes de las pruebas de punta a punta. Se incluye con `.`.
# Espera definidas: compose (función), base (URL pública), work (directorio temporal),
# lab_password, KEYCLOAK_ADMIN_USER y KEYCLOAK_ADMIN_PASSWORD.

fail() { echo "FALLO: $1" >&2; exit 1; }
status() { curl -sk -o /dev/null -w '%{http_code}' "$@"; }
code() { cat "$work/code"; }

kcadm() { compose exec -T keycloak /opt/keycloak/bin/kcadm.sh "$@" --config /tmp/kcadm.config; }

kcadm_login() {
    kcadm config credentials --server http://localhost:8080/auth --realm master \
        --user "$KEYCLOAK_ADMIN_USER" --password "$KEYCLOAK_ADMIN_PASSWORD" >/dev/null 2>&1 \
        || fail "no se pudo entrar a la administración de Keycloak"
}

# new_user usuario nombre apellido rol: crea un usuario de laboratorio en el realm kaoru.
new_user() {
    kcadm create users -r kaoru -s "username=$1" -s enabled=true -s "firstName=$2" -s "lastName=$3" \
        -s "email=$1@lab.invalid" -s emailVerified=true >/dev/null 2>&1 || fail "no se pudo crear $1"
    kcadm set-password -r kaoru --username "$1" --new-password "$lab_password" >/dev/null 2>&1 || fail "no se pudo poner la contraseña de $1"
    kcadm add-roles -r kaoru --uusername "$1" --rolename "$4" >/dev/null 2>&1 || fail "no se pudo dar el rol $4 a $1"
}

# login usuario: inicia sesión como lo hace el navegador (seguir las redirecciones hasta el
# formulario de Keycloak, enviarlo y volver) y deja las cookies en $work/<usuario>.jar.
login() {
    jar="$work/$1.jar"
    rm -f "$jar"
    page="$(curl -skL -c "$jar" -b "$jar" "$base/")"
    action="$(printf '%s' "$page" | grep -o 'action="[^"]*"' | head -1 | sed 's/action="//; s/"$//; s/&amp;/\&/g')"
    [ -n "$action" ] || fail "no apareció el formulario de inicio de sesión de Keycloak"
    http="$(curl -skL -c "$jar" -b "$jar" -o "$work/$1.html" -w '%{http_code}' \
        --data-urlencode "username=$1" --data-urlencode "password=$lab_password" "$action")"
    [ "$http" = 200 ] || fail "el inicio de sesión de $1 terminó en $http"
    grep -q 'Sesión iniciada' "$work/$1.html" || fail "tras iniciar sesión, $1 no llegó a la aplicación"
}

# api usuario método ruta [cuerpo]: imprime el cuerpo; el código HTTP queda en $work/code.
api() {
    args=(-sk -b "$work/$1.jar" -c "$work/$1.jar" -o "$work/body" -w '%{http_code}' -X "$2")
    if [ "$2" != GET ]; then args+=(-H "Origin: $base"); fi
    if [ -n "${4:-}" ]; then args+=(-H 'Content-Type: application/json' -d "$4"); fi
    curl "${args[@]}" "$base$3" >"$work/code"
    cat "$work/body"
}

# json_code prefijo: saca el primer ID legible con ese prefijo de lo que lee por la entrada.
json_code() { grep -o "\"code\":\"$1-[0-9]*-[0-9]*\"" | head -1 | cut -d'"' -f4; }

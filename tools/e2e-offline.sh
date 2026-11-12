#!/usr/bin/env bash
# Prueba el paquete offline y el respaldo: arma el paquete, instala desde él, usa el sistema con
# sesión real, respalda, destruye todo, reinstala desde cero, restaura y comprueba que nada se perdió.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
export KAORU_PROJECT=kaoru-offline-e2e
lab_password="Lab-$(openssl rand -hex 8)-9a"
# El paquete trae el front compilado: así se reconoce su página.
export LANDING_MARK='id="root"'

bundle="$("$root/tools/offline-bundle.sh" | tail -1)"
tar xzf "$bundle" -C "$work"
app="$(echo "$work"/kaoru-offline-*)"

cp "$app/.env.example" "$app/.env"
# shellcheck disable=SC1091
set -a; . "$app/.env"; set +a
base="$KAORU_PUBLIC_URL"
compose() { docker compose -p "$KAORU_PROJECT" -f "$app/docker-compose.yml" --env-file "$app/.env" "$@"; }
# shellcheck disable=SC1091
. "$root/tools/e2e-lib.sh"

cleanup() {
    status=$?
    if [ $status -ne 0 ]; then
        compose logs --tail 20 nginx oauth2-proxy kaoru-api keycloak 2>&1 | tail -60 >&2 || true
    fi
    compose down -v --remove-orphans >/dev/null 2>&1 || true
    rm -rf "$work"
    exit $status
}
trap cleanup EXIT

# El paquete no trae certificado ni referencias por digest, y sí la suma de las imágenes.
[ -f "$app/tls/tls.crt" ] && fail "el paquete no debe traer un certificado"
grep -q '@sha256:' "$app/docker-compose.yml" && fail "el Compose del paquete no debe depender del digest"
(cd "$app" && (sha256sum -c SHA256SUMS >/dev/null 2>&1 || shasum -a 256 -c SHA256SUMS >/dev/null 2>&1)) || fail "la suma de images.tar.gz no coincide"
"$app/install.sh" >/dev/null 2>&1 && fail "install.sh debe negarse a instalar sin certificado"

mkdir -p "$app/tls"
openssl req -x509 -newkey rsa:2048 -nodes -days 2 -keyout "$app/tls/tls.key" -out "$app/tls/tls.crt" \
    -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost" 2>/dev/null
chmod 644 "$app/tls/tls.key"

compose down -v --remove-orphans >/dev/null 2>&1 || true
"$app/install.sh" >/dev/null 2>&1 || fail "install.sh no dejó el sistema sano"

# Datos reales: un usuario, un CHG y un documento adjunto.
kcadm_login
new_user lab-manager Carla Mora change-manager
login lab-manager
chg="$(api lab-manager POST /api/changes '{"title":"Cambio antes del respaldo","system":"SIMF"}' | json_code CHG)"
[ -n "$chg" ] || fail "no se creó el CHG"
head -c 20000 /dev/urandom >"$work/random.bin"
{ printf '%%PDF-1.7\n'; cat "$work/random.bin"; } >"$work/evidencia.pdf"
upload="$(curl -sk -b "$work/lab-manager.jar" -H "Origin: $base" -o "$work/body" -w '%{http_code}' \
    -F title=Evidencia -F kind=evidence -F entityType=change -F "entityCode=$chg" \
    -F "file=@$work/evidencia.pdf" "$base/api/documents")"
[ "$upload" = 201 ] || fail "la subida del documento respondió $upload: $(cat "$work/body")"
doc="$(json_code DOC <"$work/body")"

# Respaldo.
backup="$("$app/backup.sh" "$work/respaldos" | tail -1)"
for f in kaoru.dump keycloak.dump documents.tar.gz SHA256SUMS; do
    [ -s "$backup/$f" ] || fail "falta $f en el respaldo"
done

# Desastre: se pierden los contenedores y todos los volúmenes.
compose down -v >/dev/null 2>&1
"$app/install.sh" >/dev/null 2>&1 || fail "la reinstalación desde cero no quedó sana"
# En un subshell: `login` termina el proceso cuando falla, que aquí es lo esperado.
if (login lab-manager) >/dev/null 2>&1; then fail "en una instalación nueva el usuario todavía no debe existir"; fi

# Un respaldo dañado se rechaza.
cp -R "$backup" "$work/danado" && echo basura >>"$work/danado/kaoru.dump"
"$app/restore.sh" "$work/danado" >/dev/null 2>&1 && fail "restore.sh debe rechazar un respaldo dañado"

"$app/restore.sh" "$backup" >/dev/null 2>&1 || fail "restore.sh falló"

# Todo volvió: el mismo usuario con su contraseña, el CHG, el documento idéntico y la auditoría íntegra.
login lab-manager
api lab-manager GET "/api/changes/$chg" | grep -q 'Cambio antes del respaldo' || fail "el CHG no volvió con la restauración"
curl -sk -b "$work/lab-manager.jar" -o "$work/descarga.pdf" "$base/api/documents/$doc/versions/1/content"
cmp -s "$work/evidencia.pdf" "$work/descarga.pdf" || fail "el documento restaurado no es idéntico al original"
api lab-manager GET /api/audit/verification | grep -q '"intact":true' || fail "tras restaurar, la cadena de auditoría debe estar íntegra"
# Y se puede seguir escribiendo: los permisos y los dueños de las tablas y del volumen quedaron bien.
api lab-manager POST /api/changes '{"title":"Cambio después de restaurar","system":"SIMF"}' >/dev/null
[ "$(code)" = 201 ] || fail "tras restaurar no se pudo crear un CHG ($(code))"
upload="$(curl -sk -b "$work/lab-manager.jar" -H "Origin: $base" -o /dev/null -w '%{http_code}' \
    -F title=Otra -F kind=evidence -F entityType=change -F "entityCode=$chg" -F "file=@$work/evidencia.pdf;filename=otra.pdf" "$base/api/documents")"
[ "$upload" = 201 ] || fail "tras restaurar no se pudo adjuntar un documento ($upload)"

echo "Paquete offline y respaldo: OK (instalar desde el paquete, respaldar, perder todo, reinstalar y restaurar)"

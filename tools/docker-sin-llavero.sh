#!/usr/bin/env bash
# Corre un comando de Docker sin credenciales guardadas: las imágenes base de Kaoru son públicas.
#   tools/docker-sin-llavero.sh docker buildx build …
# En macOS, Docker pide las credenciales de docker.io al llavero aunque la configuración no lo
# nombre (lo detecta por docker-credential-osxkeychain en el PATH). Si nadie está frente al equipo
# para aceptar el aviso, el build falla con «User canceled the operation». Aquí la configuración es
# una copia temporal sin credenciales, con los mismos complementos y el mismo contexto, y un
# ayudante falso que siempre responde «no hay credenciales». La configuración real no se toca.
set -euo pipefail

real="${DOCKER_CONFIG:-$HOME/.docker}"
context="$(docker context show)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/bin"
for dir in cli-plugins contexts; do
    [ -d "$real/$dir" ] && ln -s "$real/$dir" "$tmp/$dir"
done
printf '{"currentContext":"%s"}\n' "$context" >"$tmp/config.json"
for helper in osxkeychain desktop; do
    cat >"$tmp/bin/docker-credential-$helper" <<'EOF'
#!/bin/sh
case "$1" in
  get) cat >/dev/null; echo "credentials not found in native keychain"; exit 1 ;;
  list) echo '{}' ;;
  *) cat >/dev/null ;;
esac
EOF
    chmod +x "$tmp/bin/docker-credential-$helper"
done

DOCKER_CONFIG="$tmp" PATH="$tmp/bin:$PATH" "$@"

#!/usr/bin/env bash
# Genera un certificado autofirmado de LABORATORIO para Nginx en infra/tls/.
# En el banco se reemplaza por el certificado interno: mismos nombres de archivo.
set -euo pipefail

dir="$(cd "$(dirname "$0")/.." && pwd)/infra/tls"
host="${1:-localhost}"
mkdir -p "$dir"
if [ -f "$dir/tls.crt" ] && [ -f "$dir/tls.key" ]; then
    echo "Ya existe un certificado en $dir; no se toca."
    exit 0
fi
openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
    -keyout "$dir/tls.key" -out "$dir/tls.crt" \
    -subj "/CN=$host" -addext "subjectAltName=DNS:$host,IP:127.0.0.1" 2>/dev/null
chmod 644 "$dir/tls.crt" "$dir/tls.key"
echo "Certificado de laboratorio para $host en $dir"

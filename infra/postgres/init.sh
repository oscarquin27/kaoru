#!/bin/sh
# Se ejecuta una sola vez, cuando el volumen de Postgres nace vacío.
# Crea el esquema y los roles de Kaoru, y la base separada de Keycloak.
set -eu

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -v owner_password="$POSTGRES_OWNER_PASSWORD" -v app_password="$POSTGRES_APP_PASSWORD" \
    -f /kaoru/init.sql

psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres -v keycloak_password="$KEYCLOAK_DB_PASSWORD" <<'SQL'
CREATE ROLE keycloak LOGIN PASSWORD :'keycloak_password';
CREATE DATABASE keycloak OWNER keycloak;
SQL

# Las comprobaciones de CI, en local (reglas §A.18).
# Hoy cubren lo que existe: migraciones y contrato. Cada fase agrega lo suyo
# (Go, sqlc, front, build de Docker) cuando aparece el código.

.PHONY: check check-migrations check-openapi

check: check-openapi check-migrations

# Requiere Docker y Go. Levanta un Postgres desechable.
check-migrations:
	./tools/check-migrations.sh

check-openapi:
	ruby tools/check-openapi.rb api/openapi.yaml

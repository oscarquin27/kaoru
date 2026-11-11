# Las comprobaciones de CI, en local (reglas §A.18). Requiere Go, Docker, sqlc y ruby.
# Cada fase agrega lo suyo cuando aparece el código (front, Compose, paquete offline).

IMAGE ?= kaoru-api:dev

.PHONY: check generate check-go check-generated check-openapi check-migrations check-integration check-image

check: check-go check-generated check-openapi check-migrations check-integration check-image
	@echo "make check: OK"

# Código generado por sqlc a partir de src/migrations y de las consultas de cada feature.
generate:
	cd src && sqlc generate

# Formato, vet, pruebas unitarias, de arquitectura y de contrato.
check-go:
	@cd src && test -z "$$(gofmt -l .)" || { echo "FALLO: gofmt en:"; gofmt -l src; exit 1; }
	cd src && go vet ./... && go vet -tags integration ./...
	cd src && go test ./...

# El código generado está al día: sqlc compara lo que generaría con lo que hay en disco.
check-generated:
	cd src && sqlc diff

check-openapi:
	ruby tools/check-openapi.rb api/openapi.yaml

# Migraciones sobre un Postgres limpio: suben, bajan, suben, y los permisos de kaoru_app.
check-migrations:
	./tools/with-postgres.sh ./tools/check-migrations.sh

# Pruebas de integración: un paquete a la vez, porque comparten la base.
check-integration:
	./tools/with-postgres.sh sh -c 'cd src && go test -tags integration -p 1 ./...'

# Build de Docker y prueba de humo de la imagen.
check-image:
	docker build -q -f infra/api/Dockerfile -t $(IMAGE) .
	KAORU_API_IMAGE=$(IMAGE) ./tools/with-postgres.sh ./tools/smoke-api.sh

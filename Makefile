# Las comprobaciones de CI, en local (reglas §A.18). Requiere Go, Docker, sqlc y ruby.
# Cada fase agrega lo suyo cuando aparece el código (front, Compose, paquete offline).

IMAGE ?= kaoru-api:dev
GOLANGCI_VERSION ?= v2.15.0

.PHONY: check generate image offline-bundle check-go check-generated check-openapi check-migrations check-sqlvet \
	check-integration check-image check-web check-e2e check-front check-offline

check: check-go check-generated check-openapi check-migrations check-sqlvet check-integration check-image check-web check-e2e check-front
	@echo "make check: OK"

# Imagen de kaoru-api.
image:
	docker build -f infra/api/Dockerfile -t $(IMAGE) .

# Paquete para instalar en un servidor sin internet: deja dist/kaoru-offline-<versión>.tar.gz
offline-bundle:
	./tools/offline-bundle.sh

# Código generado por sqlc a partir de src/migrations y de las consultas de cada feature.
generate:
	cd src && sqlc generate

# Formato, vet, pruebas unitarias, de arquitectura y de contrato.
check-go:
	@cd src && test -z "$$(gofmt -l .)" || { echo "FALLO: gofmt en:"; gofmt -l src; exit 1; }
	cd src && go vet ./... && go vet -tags integration ./...
	cd src && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION) run --timeout 5m ./...
	cd src && go test ./...

# El código generado está al día: sqlc compara lo que generaría con lo que hay en disco.
check-generated:
	cd src && sqlc diff

check-openapi:
	ruby tools/check-openapi.rb api/openapi.yaml
	cd web && node ../tools/lint-openapi.mjs

# Cada consulta de sqlc se prepara contra un Postgres real ya migrado.
check-sqlvet:
	./tools/with-postgres.sh ./tools/check-sqlc-vet.sh

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

# El front: lint, tipos, pruebas y build. Requiere `npm --prefix web ci` una vez.
check-web:
	npm --prefix web run check

# El front compilado detrás del perímetro real: solo con sesión, rutas directas y política de seguridad.
check-front:
	./tools/e2e-front.sh

# El despliegue completo con sesión real: Keycloak, oauth2-proxy, Nginx y la API.
check-e2e:
	./tools/e2e-compose.sh

# El paquete offline y el respaldo: instalar, respaldar, perder todo, reinstalar y restaurar.
# Tarda varios minutos; se corre antes de entregar un paquete, no en cada cambio.
check-offline:
	./tools/e2e-offline.sh

# Imagen de kaoru-api. El contexto de build es la raíz del repositorio:
#   docker build -f infra/api/Dockerfile -t kaoru-api:dev .
# Se construye donde hay internet; al servidor offline viaja la imagen ya hecha.

FROM golang:1.26-alpine@sha256:30b1e546a9c7fa3b747c2bdba3a0c7bdf3a02ebb4c5b8405d2f948d7e8aac4ec AS build
WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/kaoru-api ./cmd/kaoru-api \
    && mkdir -p /out/documents

# Un solo binario estático: sin shell, sin paquetes, sin usuario root.
FROM scratch
COPY --from=build /out/kaoru-api /kaoru-api
COPY infra/api/config.json /etc/kaoru/config.json
# El directorio de documentos nace vacío y del usuario de la app; en producción es un volumen.
COPY --from=build --chown=65532:65532 /out/documents /var/lib/kaoru/documents
VOLUME /var/lib/kaoru/documents
ENV KAORU_CONFIG_PATH=/etc/kaoru/config.json
USER 65532:65532
EXPOSE 8080 9090
HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=3 CMD ["/kaoru-api", "healthcheck"]
ENTRYPOINT ["/kaoru-api"]

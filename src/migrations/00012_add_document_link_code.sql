-- +goose Up
-- El enlace guarda también el ID legible de la entidad. Los códigos son inmutables, así que
-- repetirlo aquí es seguro y evita que `document` tenga que leer las tablas de otros features.
ALTER TABLE document_link
    ADD COLUMN entity_code text;

-- La tabla todavía no tiene filas en ninguna instalación; si las tuviera, quedarían sin código.
DELETE
FROM document_link
WHERE entity_code IS NULL;

ALTER TABLE document_link
    ALTER COLUMN entity_code SET NOT NULL;

-- +goose Down
ALTER TABLE document_link
    DROP COLUMN entity_code;

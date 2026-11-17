-- +goose Up
-- La clase de un COMP deja de ser una lista cerrada: QA escribe la que necesite (por ejemplo
-- «Servicio de Windows»). Las seis de antes se siguen ofreciendo y conservan su clave.
ALTER TABLE component
    DROP CONSTRAINT component_kind_check,
    ADD CONSTRAINT component_kind_check CHECK (char_length(kind) BETWEEN 1 AND 60 AND kind = btrim(kind));

-- +goose Down
UPDATE component SET kind = 'other'
WHERE kind NOT IN ('microservice', 'library', 'script', 'kafka_topic', 'db_schema', 'other');
ALTER TABLE component
    DROP CONSTRAINT component_kind_check,
    ADD CONSTRAINT component_kind_check CHECK (kind IN ('microservice', 'library', 'script', 'kafka_topic', 'db_schema', 'other'));

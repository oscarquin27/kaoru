-- +goose Up
-- Auditoría de solo inserción, encadenada por hash (regla de negocio 9).
-- El tiempo en cada estado se calcula al leer, con las filas action = 'transition'.
CREATE TABLE audit_entry (
    id          uuid PRIMARY KEY,
    seq         bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
    occurred_at timestamptz NOT NULL,
    actor_id    text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    entity_type text        NOT NULL,
    entity_id   uuid        NOT NULL,
    entity_code text,
    action      text        NOT NULL CHECK (action IN ('create', 'update', 'transition', 'decision', 'delete')),
    field       text,
    old_value   text,
    new_value   text,
    reason      text,
    -- SHA-256 en hexadecimal. `prev_hash` es NULL solo en la primera fila.
    prev_hash   text CHECK (prev_hash ~ '^[0-9a-f]{64}$'),
    hash        text        NOT NULL UNIQUE CHECK (hash ~ '^[0-9a-f]{64}$')
);

CREATE INDEX audit_entry_entity ON audit_entry (entity_type, entity_id, seq);

REVOKE UPDATE, DELETE ON audit_entry FROM kaoru_app;

-- +goose Down
DROP TABLE audit_entry;

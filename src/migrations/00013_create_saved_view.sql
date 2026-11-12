-- +goose Up
-- Vistas guardadas: los filtros de una lista con nombre, personales de cada usuario.
-- Compartir sigue siendo por URL. No se auditan: son una preferencia, no un dato del proceso.
CREATE TABLE saved_view (
    id         uuid PRIMARY KEY,
    owner_id   text        NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    screen     text        NOT NULL CHECK (screen IN ('requirements', 'changes', 'defects', 'matrix', 'test-cases', 'audit')),
    name       text        NOT NULL,
    -- Los filtros tal como van en la URL, sin el signo de interrogación.
    query      text        NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (owner_id, screen, name)
);

-- +goose Down
DROP TABLE saved_view;

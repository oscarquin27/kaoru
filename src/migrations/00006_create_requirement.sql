-- +goose Up
CREATE TABLE requirement (
    id                  uuid PRIMARY KEY,
    code                text        NOT NULL UNIQUE,
    title               text        NOT NULL,
    description         text        NOT NULL DEFAULT '',
    acceptance_criteria text        NOT NULL DEFAULT '',
    system_id           uuid        NOT NULL REFERENCES system_under_test (id) ON DELETE RESTRICT,
    -- Un REQ puede existir antes de que un CHG lo agrupe.
    change_id           uuid REFERENCES change (id) ON DELETE RESTRICT,
    state_id            uuid        NOT NULL REFERENCES workflow_state (id) ON DELETE RESTRICT,
    -- Responsable actual.
    owner_id            text REFERENCES app_user (id) ON DELETE RESTRICT,
    created_by          text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at          timestamptz NOT NULL,
    updated_at          timestamptz NOT NULL
);

CREATE INDEX requirement_change ON requirement (change_id);
CREATE INDEX requirement_system ON requirement (system_id);

CREATE TABLE requirement_link (
    id                   uuid PRIMARY KEY,
    requirement_id       uuid        NOT NULL REFERENCES requirement (id) ON DELETE RESTRICT,
    kind                 text        NOT NULL CHECK (kind IN ('google_sheets', 'youtrack', 'shared_folder', 'wiki', 'other')),
    label                text        NOT NULL,
    url                  text        NOT NULL,
    -- Solo para Google Sheets: ID de la hoja, pestaña y rango extraídos de la URL.
    sheet_id             text,
    sheet_gid            text,
    sheet_range          text,
    is_official          boolean     NOT NULL DEFAULT false,
    -- Instantánea adjunta: es la que cuenta para el informe de conformidad.
    snapshot_document_id uuid REFERENCES document (id) ON DELETE RESTRICT,
    added_by             text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at           timestamptz NOT NULL,
    updated_at           timestamptz NOT NULL,
    CHECK ((kind = 'google_sheets') = (sheet_id IS NOT NULL)),
    CHECK (sheet_id IS NOT NULL OR sheet_gid IS NULL AND sheet_range IS NULL)
);

CREATE INDEX requirement_link_requirement ON requirement_link (requirement_id);

-- Contrato de entrada dev → QA (regla de negocio 8): ítems configurables.
CREATE TABLE entry_contract_item (
    key        text PRIMARY KEY,
    label      text        NOT NULL,
    position   integer     NOT NULL,
    enabled    boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

-- Una fila significa «ítem cumplido». Quitar la marca borra la fila; queda en auditoría.
CREATE TABLE requirement_entry_check (
    requirement_id uuid        NOT NULL REFERENCES requirement (id) ON DELETE RESTRICT,
    item_key       text        NOT NULL REFERENCES entry_contract_item (key) ON DELETE RESTRICT,
    note           text        NOT NULL DEFAULT '',
    checked_by     text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    checked_at     timestamptz NOT NULL,
    PRIMARY KEY (requirement_id, item_key)
);

INSERT INTO entry_contract_item (key, label, position, created_at, updated_at)
VALUES ('ids', 'IDs de REQ y CHG', 1, now(), now()),
       ('components', 'Componentes impactados', 2, now(), now()),
       ('build_manifest', 'BUILD con manifiesto', 3, now(), now()),
       ('install_windows', 'Script de instalación para Windows probado en ambiente limpio', 4, now(), now()),
       ('install_linux', 'Script de instalación para Linux probado en ambiente limpio', 5, now(), now()),
       ('unit_tests', 'Pruebas unitarias y cobertura', 6, now(), now()),
       ('rollback', 'Notas de rollback', 7, now(), now()),
       ('test_data', 'Datos de prueba', 8, now(), now()),
       ('known_issues', 'Known issues', 9, now(), now());

-- Comentarios de la pestaña Actividad, sobre cualquier entidad.
CREATE TABLE comment (
    id          uuid PRIMARY KEY,
    entity_type text        NOT NULL CHECK (entity_type IN ('requirement', 'change', 'defect')),
    entity_id   uuid        NOT NULL,
    body        text        NOT NULL,
    author_id   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at  timestamptz NOT NULL
);

CREATE INDEX comment_entity ON comment (entity_type, entity_id, created_at);

REVOKE UPDATE, DELETE ON comment FROM kaoru_app;

-- +goose Down
DROP TABLE comment;
DROP TABLE requirement_entry_check;
DROP TABLE entry_contract_item;
DROP TABLE requirement_link;
DROP TABLE requirement;

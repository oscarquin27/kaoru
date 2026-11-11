-- +goose Up
-- El contenido vive en el almacenamiento de archivos, nombrado por su SHA-256.
CREATE TABLE document (
    id         uuid PRIMARY KEY,
    code       text        NOT NULL UNIQUE,
    title      text        NOT NULL,
    kind       text        NOT NULL CHECK (kind IN ('specification', 'manual', 'evidence', 'minutes', 'snapshot', 'other')),
    created_by text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE document_version (
    id          uuid PRIMARY KEY,
    document_id uuid        NOT NULL REFERENCES document (id) ON DELETE RESTRICT,
    version     integer     NOT NULL CHECK (version >= 1),
    file_name   text        NOT NULL,
    media_type  text        NOT NULL,
    size_bytes  bigint      NOT NULL CHECK (size_bytes >= 0),
    checksum    text        NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
    uploaded_by text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at  timestamptz NOT NULL,
    UNIQUE (document_id, version)
);

REVOKE UPDATE, DELETE ON document_version FROM kaoru_app;

-- Un documento se enlaza a cualquier entidad. Sin llave foránea hacia la
-- entidad: el caso de uso comprueba que existe.
CREATE TABLE document_link (
    document_id uuid        NOT NULL REFERENCES document (id) ON DELETE RESTRICT,
    entity_type text        NOT NULL CHECK (entity_type IN (
        'requirement', 'change', 'functional_case', 'test_case', 'test_run', 'exploratory_session', 'defect')),
    entity_id   uuid        NOT NULL,
    linked_by   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at  timestamptz NOT NULL,
    PRIMARY KEY (document_id, entity_type, entity_id)
);

CREATE INDEX document_link_entity ON document_link (entity_type, entity_id);

-- +goose Down
DROP TABLE document_link;
DROP TABLE document_version;
DROP TABLE document;

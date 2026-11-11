-- +goose Up
-- BUILD- y ENV- llegan en la fase 3. Hasta entonces, RUN y EXP guardan la
-- versión y el checksum del BUILD y el nombre del ambiente como texto.

-- RUN-: ejecución oficial. Inmutable una vez creada.
CREATE TABLE test_run (
    id               uuid PRIMARY KEY,
    code             text        NOT NULL UNIQUE,
    test_case_id     uuid        NOT NULL REFERENCES test_case (id) ON DELETE RESTRICT,
    build_version    text        NOT NULL,
    build_checksum   text        NOT NULL CHECK (build_checksum ~ '^[0-9a-f]{64}$'),
    environment_name text        NOT NULL,
    result           text        NOT NULL CHECK (result IN ('passed', 'failed', 'blocked')),
    notes            text        NOT NULL DEFAULT '',
    executed_by      text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    executed_at      timestamptz NOT NULL,
    created_at       timestamptz NOT NULL
);

CREATE INDEX test_run_test_case ON test_run (test_case_id, executed_at);

REVOKE UPDATE, DELETE ON test_run FROM kaoru_app;

-- EXP-: ejecución exploratoria. No cuenta como evidencia de conformidad.
CREATE TABLE exploratory_session (
    id               uuid PRIMARY KEY,
    code             text        NOT NULL UNIQUE,
    requirement_id   uuid        NOT NULL REFERENCES requirement (id) ON DELETE RESTRICT,
    build_version    text        NOT NULL,
    environment_name text        NOT NULL,
    -- Cambios manuales hechos sobre el BUILD o el ambiente.
    manual_changes   text        NOT NULL DEFAULT '',
    findings         text        NOT NULL DEFAULT '',
    executed_by      text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    executed_at      timestamptz NOT NULL,
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL
);

CREATE INDEX exploratory_session_requirement ON exploratory_session (requirement_id);

-- +goose Down
DROP TABLE exploratory_session;
DROP TABLE test_run;

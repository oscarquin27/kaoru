-- +goose Up
-- DEF-: ligado a un REQ y al RUN que lo reprodujo (regla de negocio 5).
-- Quién aceptó diferirlo y por qué queda en la auditoría de la transición.
CREATE TABLE defect (
    id             uuid PRIMARY KEY,
    code           text        NOT NULL UNIQUE,
    title          text        NOT NULL,
    description    text        NOT NULL,
    requirement_id uuid        NOT NULL REFERENCES requirement (id) ON DELETE RESTRICT,
    test_run_id    uuid        NOT NULL REFERENCES test_run (id) ON DELETE RESTRICT,
    severity       text        NOT NULL CHECK (severity IN ('high', 'medium', 'low')),
    state_id       uuid        NOT NULL REFERENCES workflow_state (id) ON DELETE RESTRICT,
    assignee_id    text REFERENCES app_user (id) ON DELETE RESTRICT,
    created_by     text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL
);

CREATE INDEX defect_requirement ON defect (requirement_id);
CREATE INDEX defect_test_run ON defect (test_run_id);

-- +goose Down
DROP TABLE defect;

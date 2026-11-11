-- +goose Up
-- UC-: caso de uso funcional (`FunctionalCase` en el código, reglas §A.2).
CREATE TABLE functional_case (
    id             uuid PRIMARY KEY,
    code           text        NOT NULL UNIQUE,
    requirement_id uuid        NOT NULL REFERENCES requirement (id) ON DELETE RESTRICT,
    title          text        NOT NULL,
    description    text        NOT NULL DEFAULT '',
    created_by     text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL
);

CREATE INDEX functional_case_requirement ON functional_case (requirement_id);

CREATE TABLE test_case (
    id              uuid PRIMARY KEY,
    code            text        NOT NULL UNIQUE,
    system_id       uuid        NOT NULL REFERENCES system_under_test (id) ON DELETE RESTRICT,
    title           text        NOT NULL,
    preconditions   text        NOT NULL DEFAULT '',
    steps           text        NOT NULL,
    expected_result text        NOT NULL,
    is_critical     boolean     NOT NULL DEFAULT false,
    -- Un TC con ejecuciones no se borra: se retira.
    retired_at      timestamptz,
    created_by      text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at      timestamptz NOT NULL,
    updated_at      timestamptz NOT NULL
);

CREATE INDEX test_case_system ON test_case (system_id);

-- Un TC valida uno o más UC.
CREATE TABLE test_case_functional_case (
    test_case_id       uuid NOT NULL REFERENCES test_case (id) ON DELETE RESTRICT,
    functional_case_id uuid NOT NULL REFERENCES functional_case (id) ON DELETE RESTRICT,
    PRIMARY KEY (test_case_id, functional_case_id)
);

CREATE INDEX test_case_functional_case_fc ON test_case_functional_case (functional_case_id);

-- +goose Down
DROP TABLE test_case_functional_case;
DROP TABLE test_case;
DROP TABLE functional_case;

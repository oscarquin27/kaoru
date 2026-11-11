-- +goose Up
CREATE TABLE change (
    id           uuid PRIMARY KEY,
    code         text        NOT NULL UNIQUE,
    title        text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    system_id    uuid        NOT NULL REFERENCES system_under_test (id) ON DELETE RESTRICT,
    route        text        NOT NULL CHECK (route IN ('high', 'light', 'minor_fix')),
    state_id     uuid        NOT NULL REFERENCES workflow_state (id) ON DELETE RESTRICT,
    -- Ventana de cambio; la define change management.
    window_start timestamptz,
    window_end   timestamptz,
    created_by   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL,
    CHECK (window_end IS NULL OR window_start IS NOT NULL AND window_end > window_start)
);

CREATE INDEX change_system ON change (system_id);

-- +goose Down
DROP TABLE change;

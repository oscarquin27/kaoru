-- +goose Up
-- Flujos configurables: estados, transiciones, permisos y reglas de bloqueo.
-- Los valores iniciales están en 00011_seed_workflow.sql.

CREATE TABLE workflow_state (
    id          uuid PRIMARY KEY,
    entity_type text        NOT NULL CHECK (entity_type IN ('requirement', 'change', 'defect')),
    key         text        NOT NULL,
    name        text        NOT NULL,
    -- Orden en la línea de vida. NULL: estado lateral (rechazado, diferido…).
    position    integer,
    is_initial  boolean     NOT NULL DEFAULT false,
    is_final    boolean     NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL,
    UNIQUE (entity_type, key)
);

CREATE UNIQUE INDEX workflow_state_one_initial
    ON workflow_state (entity_type) WHERE is_initial;

CREATE TABLE permission (
    key         text PRIMARY KEY,
    description text NOT NULL
);

-- `role` es el nombre del rol de realm en Keycloak.
CREATE TABLE role_permission (
    role           text NOT NULL,
    permission_key text NOT NULL REFERENCES permission (key) ON DELETE RESTRICT,
    PRIMARY KEY (role, permission_key)
);

CREATE TABLE workflow_transition (
    id              uuid PRIMARY KEY,
    from_state_id   uuid        NOT NULL REFERENCES workflow_state (id) ON DELETE RESTRICT,
    to_state_id     uuid        NOT NULL REFERENCES workflow_state (id) ON DELETE RESTRICT,
    permission_key  text        NOT NULL REFERENCES permission (key) ON DELETE RESTRICT,
    requires_reason boolean     NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL,
    updated_at      timestamptz NOT NULL,
    UNIQUE (from_state_id, to_state_id),
    CHECK (from_state_id <> to_state_id)
);

-- `rule_key` nombra una regla que el dominio sabe evaluar; `params` la ajusta.
-- Con `blocks = false` la regla solo avisa.
CREATE TABLE blocking_rule (
    id            uuid PRIMARY KEY,
    transition_id uuid        NOT NULL REFERENCES workflow_transition (id) ON DELETE CASCADE,
    rule_key      text        NOT NULL CHECK (rule_key IN (
        'risk_assessed',
        'risk_has_quality_characteristics',
        'entry_contract_complete',
        'min_coverage',
        'no_open_defect_with_severity',
        'defect_severity_is_not',
        'links_have_snapshot')),
    params        jsonb       NOT NULL DEFAULT '{}',
    -- NULL: aplica a todas las rutas del CHG.
    route         text CHECK (route IN ('high', 'light', 'minor_fix')),
    blocks        boolean     NOT NULL DEFAULT true,
    enabled       boolean     NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL,
    updated_at    timestamptz NOT NULL
);

CREATE INDEX blocking_rule_transition ON blocking_rule (transition_id);

-- +goose Down
DROP TABLE blocking_rule;
DROP TABLE workflow_transition;
DROP TABLE role_permission;
DROP TABLE permission;
DROP TABLE workflow_state;

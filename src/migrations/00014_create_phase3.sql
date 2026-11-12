-- +goose Up
-- Fase 3: catálogo de componentes, BUILD, ambientes, despliegues, paquetes y selección de regresión.

-- COMP-: componente del catálogo de un sistema (microservicio, DLL, script, tópico, esquema…).
CREATE TABLE component (
    id          uuid PRIMARY KEY,
    code        text        NOT NULL UNIQUE,
    system_id   uuid        NOT NULL REFERENCES system_under_test (id) ON DELETE RESTRICT,
    kind        text        NOT NULL CHECK (kind IN ('microservice', 'library', 'script', 'kafka_topic', 'db_schema', 'other')),
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    -- Un componente retirado no se ofrece para declarar impacto, pero conserva su historia.
    retired_at  timestamptz,
    created_by  text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL,
    UNIQUE (system_id, name)
);

-- Impacto: qué componentes toca un REQ y cuáles ejercita un TC (regla de negocio 4).
CREATE TABLE requirement_component (
    requirement_id uuid NOT NULL REFERENCES requirement (id) ON DELETE RESTRICT,
    component_id   uuid NOT NULL REFERENCES component (id) ON DELETE RESTRICT,
    PRIMARY KEY (requirement_id, component_id)
);
CREATE INDEX requirement_component_component ON requirement_component (component_id);

CREATE TABLE test_case_component (
    test_case_id uuid NOT NULL REFERENCES test_case (id) ON DELETE RESTRICT,
    component_id uuid NOT NULL REFERENCES component (id) ON DELETE RESTRICT,
    PRIMARY KEY (test_case_id, component_id)
);
CREATE INDEX test_case_component_component ON test_case_component (component_id);

-- BUILD-: artefacto inmutable. Su manifiesto dice qué REQ incluye, qué migraciones y qué dependencias.
CREATE TABLE build (
    id           uuid PRIMARY KEY,
    code         text        NOT NULL UNIQUE,
    system_id    uuid        NOT NULL REFERENCES system_under_test (id) ON DELETE RESTRICT,
    version      text        NOT NULL,
    -- SHA-256 del artefacto. El mismo artefacto no se registra dos veces.
    checksum     text        NOT NULL UNIQUE CHECK (checksum ~ '^[0-9a-f]{64}$'),
    migrations   text        NOT NULL DEFAULT '',
    dependencies text        NOT NULL DEFAULT '',
    notes        text        NOT NULL DEFAULT '',
    built_by     text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at   timestamptz NOT NULL
);
CREATE INDEX build_system ON build (system_id, created_at);

CREATE TABLE build_requirement (
    build_id       uuid NOT NULL REFERENCES build (id) ON DELETE RESTRICT,
    requirement_id uuid NOT NULL REFERENCES requirement (id) ON DELETE RESTRICT,
    PRIMARY KEY (build_id, requirement_id)
);
CREATE INDEX build_requirement_requirement ON build_requirement (requirement_id);

-- Un BUILD no se modifica después de construirse.
REVOKE UPDATE, DELETE ON build FROM kaoru_app;
REVOKE UPDATE, DELETE ON build_requirement FROM kaoru_app;

-- ENV-: ambiente de un sistema. La versión instalada se lee de su último despliegue exitoso.
CREATE TABLE environment (
    id         uuid PRIMARY KEY,
    code       text        NOT NULL UNIQUE,
    system_id  uuid        NOT NULL REFERENCES system_under_test (id) ON DELETE RESTRICT,
    name       text        NOT NULL,
    kind       text        NOT NULL CHECK (kind IN ('clean', 'baseline', 'offline', 'production')),
    os         text        NOT NULL CHECK (os IN ('windows', 'linux')),
    notes      text        NOT NULL DEFAULT '',
    retired_at timestamptz,
    created_by text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (system_id, name)
);

-- PKG-: conjunto de BUILD y REQ listo para desplegar. Su estado sigue un flujo configurable.
CREATE TABLE package (
    id          uuid PRIMARY KEY,
    code        text        NOT NULL UNIQUE,
    title       text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    system_id   uuid        NOT NULL REFERENCES system_under_test (id) ON DELETE RESTRICT,
    state_id    uuid        NOT NULL REFERENCES workflow_state (id) ON DELETE RESTRICT,
    created_by  text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at  timestamptz NOT NULL,
    updated_at  timestamptz NOT NULL
);
CREATE INDEX package_system ON package (system_id);

CREATE TABLE package_build (
    package_id uuid        NOT NULL REFERENCES package (id) ON DELETE RESTRICT,
    build_id   uuid        NOT NULL REFERENCES build (id) ON DELETE RESTRICT,
    added_by   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    added_at   timestamptz NOT NULL,
    PRIMARY KEY (package_id, build_id)
);
CREATE INDEX package_build_build ON package_build (build_id);

-- Un REQ puede estar en más de un paquete: el que se difiere pasa al siguiente.
CREATE TABLE package_requirement (
    package_id     uuid        NOT NULL REFERENCES package (id) ON DELETE RESTRICT,
    requirement_id uuid        NOT NULL REFERENCES requirement (id) ON DELETE RESTRICT,
    added_by       text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    added_at       timestamptz NOT NULL,
    PRIMARY KEY (package_id, requirement_id)
);
CREATE INDEX package_requirement_requirement ON package_requirement (requirement_id);

-- DEP-: despliegue registrado de un BUILD o de un PKG en un ambiente. No se corrige: se registra otro.
CREATE TABLE deployment (
    id             uuid PRIMARY KEY,
    code           text        NOT NULL UNIQUE,
    environment_id uuid        NOT NULL REFERENCES environment (id) ON DELETE RESTRICT,
    build_id       uuid REFERENCES build (id) ON DELETE RESTRICT,
    package_id     uuid REFERENCES package (id) ON DELETE RESTRICT,
    result         text        NOT NULL CHECK (result IN ('success', 'failed', 'rolled_back')),
    notes          text        NOT NULL DEFAULT '',
    executed_by    text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    executed_at    timestamptz NOT NULL,
    created_at     timestamptz NOT NULL,
    -- Se despliega un BUILD o un PKG, nunca los dos ni ninguno.
    CHECK ((build_id IS NULL) <> (package_id IS NULL))
);
CREATE INDEX deployment_environment ON deployment (environment_id, executed_at);
CREATE INDEX deployment_build ON deployment (build_id);
CREATE INDEX deployment_package ON deployment (package_id);

REVOKE UPDATE, DELETE ON deployment FROM kaoru_app;

-- Selección de regresión de un paquete: lo que el sistema propuso y lo que QA dejó al final.
CREATE TABLE regression_selection (
    package_id   uuid PRIMARY KEY REFERENCES package (id) ON DELETE RESTRICT,
    proposed_by  text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    proposed_at  timestamptz NOT NULL,
    -- Confirmada, la selección ya no cambia.
    confirmed_by text REFERENCES app_user (id) ON DELETE RESTRICT,
    confirmed_at timestamptz,
    CHECK ((confirmed_by IS NULL) = (confirmed_at IS NULL))
);

CREATE TABLE regression_item (
    package_id   uuid    NOT NULL REFERENCES regression_selection (package_id) ON DELETE CASCADE,
    test_case_id uuid    NOT NULL REFERENCES test_case (id) ON DELETE RESTRICT,
    -- Por qué está: cruza con un componente cambiado, es crítico del sistema o lo agregó QA.
    origin       text    NOT NULL CHECK (origin IN ('impact', 'critical', 'manual')),
    -- `proposed` es lo que propuso el sistema; `included`, lo que queda en la selección final.
    proposed     boolean NOT NULL,
    included     boolean NOT NULL,
    -- El criterio de cada ajuste de QA: obligatorio cuando `included` difiere de `proposed`.
    reason       text    NOT NULL DEFAULT '',
    decided_by   text REFERENCES app_user (id) ON DELETE RESTRICT,
    decided_at   timestamptz,
    PRIMARY KEY (package_id, test_case_id),
    CHECK (included = proposed OR reason <> '')
);
CREATE INDEX regression_item_test_case ON regression_item (test_case_id);

-- RUN y EXP se registran sobre un BUILD y un ENV del catálogo. La versión, el checksum y el nombre
-- del ambiente siguen copiados en la fila: son la evidencia tal como era al ejecutar.
-- Las filas anteriores a esta fase quedan sin BUILD ni ENV.
ALTER TABLE test_run
    ADD COLUMN build_id       uuid REFERENCES build (id) ON DELETE RESTRICT,
    ADD COLUMN environment_id uuid REFERENCES environment (id) ON DELETE RESTRICT;
CREATE INDEX test_run_build ON test_run (build_id);

ALTER TABLE exploratory_session
    ADD COLUMN build_id       uuid REFERENCES build (id) ON DELETE RESTRICT,
    ADD COLUMN environment_id uuid REFERENCES environment (id) ON DELETE RESTRICT;

-- El paquete tiene flujo propio, reglas propias, documentos y comentarios.
ALTER TABLE workflow_state
    DROP CONSTRAINT workflow_state_entity_type_check,
    ADD CONSTRAINT workflow_state_entity_type_check CHECK (entity_type IN ('requirement', 'change', 'defect', 'package'));

ALTER TABLE blocking_rule
    DROP CONSTRAINT blocking_rule_rule_key_check,
    ADD CONSTRAINT blocking_rule_rule_key_check CHECK (rule_key IN (
        'risk_assessed',
        'risk_has_quality_characteristics',
        'entry_contract_complete',
        'min_coverage',
        'no_open_defect_with_severity',
        'defect_severity_is_not',
        'links_have_snapshot',
        'package_has_content',
        'regression_confirmed',
        'regression_passed',
        'deployed_to_production'));

ALTER TABLE document_link
    DROP CONSTRAINT document_link_entity_type_check,
    ADD CONSTRAINT document_link_entity_type_check CHECK (entity_type IN (
        'requirement', 'change', 'functional_case', 'test_case', 'test_run', 'exploratory_session', 'defect',
        'package', 'deployment'));

ALTER TABLE comment
    DROP CONSTRAINT comment_entity_type_check,
    ADD CONSTRAINT comment_entity_type_check CHECK (entity_type IN ('requirement', 'change', 'defect', 'package'));

-- +goose Down
DELETE FROM comment WHERE entity_type = 'package';
ALTER TABLE comment
    DROP CONSTRAINT comment_entity_type_check,
    ADD CONSTRAINT comment_entity_type_check CHECK (entity_type IN ('requirement', 'change', 'defect'));

DELETE FROM document_link WHERE entity_type IN ('package', 'deployment');
ALTER TABLE document_link
    DROP CONSTRAINT document_link_entity_type_check,
    ADD CONSTRAINT document_link_entity_type_check CHECK (entity_type IN (
        'requirement', 'change', 'functional_case', 'test_case', 'test_run', 'exploratory_session', 'defect'));

ALTER TABLE blocking_rule
    DROP CONSTRAINT blocking_rule_rule_key_check,
    ADD CONSTRAINT blocking_rule_rule_key_check CHECK (rule_key IN (
        'risk_assessed',
        'risk_has_quality_characteristics',
        'entry_contract_complete',
        'min_coverage',
        'no_open_defect_with_severity',
        'defect_severity_is_not',
        'links_have_snapshot'));

ALTER TABLE exploratory_session
    DROP COLUMN environment_id,
    DROP COLUMN build_id;
ALTER TABLE test_run
    DROP COLUMN environment_id,
    DROP COLUMN build_id;

DROP TABLE regression_item;
DROP TABLE regression_selection;
DROP TABLE deployment;
DROP TABLE package_requirement;
DROP TABLE package_build;
DROP TABLE package;

ALTER TABLE workflow_state
    DROP CONSTRAINT workflow_state_entity_type_check,
    ADD CONSTRAINT workflow_state_entity_type_check CHECK (entity_type IN ('requirement', 'change', 'defect'));

DROP TABLE environment;
DROP TABLE build_requirement;
DROP TABLE build;
DROP TABLE test_case_component;
DROP TABLE requirement_component;
DROP TABLE component;

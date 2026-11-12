-- +goose Up
-- Fase 4: checklist de producción con firmas, diagnóstico, prueba de humo, informe de conformidad e hitos.

-- Plantilla del checklist de un sistema. Es el checklist «versionado»: no se edita, se publica otra versión.
CREATE TABLE checklist_template (
    id         uuid PRIMARY KEY,
    system_id  uuid        NOT NULL REFERENCES system_under_test (id) ON DELETE RESTRICT,
    version    integer     NOT NULL CHECK (version >= 1),
    created_by text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL,
    UNIQUE (system_id, version)
);

CREATE TABLE checklist_template_step (
    template_id uuid    NOT NULL REFERENCES checklist_template (id) ON DELETE RESTRICT,
    position    integer NOT NULL CHECK (position >= 1),
    title       text    NOT NULL,
    description text    NOT NULL DEFAULT '',
    PRIMARY KEY (template_id, position)
);

-- CHK-: la ejecución de una versión de la plantilla para un PKG en un ambiente de producción.
CREATE TABLE checklist (
    id             uuid PRIMARY KEY,
    code           text        NOT NULL UNIQUE,
    package_id     uuid        NOT NULL REFERENCES package (id) ON DELETE RESTRICT,
    environment_id uuid        NOT NULL REFERENCES environment (id) ON DELETE RESTRICT,
    template_id    uuid        NOT NULL REFERENCES checklist_template (id) ON DELETE RESTRICT,
    started_by     text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at     timestamptz NOT NULL
);
CREATE INDEX checklist_package ON checklist (package_id, created_at);

-- La firma de un paso: quién, cuándo y con qué comentario. Un paso se firma una vez y no se desfirma.
CREATE TABLE checklist_signature (
    checklist_id uuid        NOT NULL REFERENCES checklist (id) ON DELETE RESTRICT,
    position     integer     NOT NULL,
    comment      text        NOT NULL DEFAULT '',
    signed_by    text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    signed_at    timestamptz NOT NULL,
    PRIMARY KEY (checklist_id, position)
);

REVOKE UPDATE, DELETE ON checklist_template FROM kaoru_app;
REVOKE UPDATE, DELETE ON checklist_template_step FROM kaoru_app;
REVOKE UPDATE, DELETE ON checklist FROM kaoru_app;
REVOKE UPDATE, DELETE ON checklist_signature FROM kaoru_app;

-- DIAG-: lo que dev u ops entregan del diagnóstico posterior a un despliegue en producción.
CREATE TABLE diagnostic (
    id            uuid PRIMARY KEY,
    code          text        NOT NULL UNIQUE,
    deployment_id uuid        NOT NULL REFERENCES deployment (id) ON DELETE RESTRICT,
    notes         text        NOT NULL DEFAULT '',
    submitted_by  text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at    timestamptz NOT NULL
);
CREATE INDEX diagnostic_deployment ON diagnostic (deployment_id);

-- La revisión de QA de un DIAG: marca el resultado, una sola vez (regla de negocio 7).
CREATE TABLE diagnostic_review (
    diagnostic_id uuid PRIMARY KEY REFERENCES diagnostic (id) ON DELETE RESTRICT,
    result        text        NOT NULL CHECK (result IN ('conforming', 'non_conforming')),
    notes         text        NOT NULL DEFAULT '',
    reviewed_by   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    reviewed_at   timestamptz NOT NULL
);

-- SMOKE-: el resultado de la prueba de humo posterior a un despliegue en producción, que registra QA.
CREATE TABLE smoke_test (
    id            uuid PRIMARY KEY,
    code          text        NOT NULL UNIQUE,
    deployment_id uuid        NOT NULL REFERENCES deployment (id) ON DELETE RESTRICT,
    result        text        NOT NULL CHECK (result IN ('passed', 'failed')),
    notes         text        NOT NULL DEFAULT '',
    executed_by   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at    timestamptz NOT NULL
);
CREATE INDEX smoke_test_deployment ON smoke_test (deployment_id);

REVOKE UPDATE, DELETE ON diagnostic FROM kaoru_app;
REVOKE UPDATE, DELETE ON diagnostic_review FROM kaoru_app;
REVOKE UPDATE, DELETE ON smoke_test FROM kaoru_app;

-- RPT-: informe de conformidad de un PKG. Inmutable: si algo cambia, se emite otra versión
-- que referencia la anterior (regla de negocio 11).
CREATE TABLE conformance_report (
    id          uuid PRIMARY KEY,
    code        text        NOT NULL UNIQUE,
    package_id  uuid        NOT NULL REFERENCES package (id) ON DELETE RESTRICT,
    version     integer     NOT NULL CHECK (version >= 1),
    previous_id uuid REFERENCES conformance_report (id) ON DELETE RESTRICT,
    -- El Markdown tal como se emitió y su SHA-256: el hash del informe.
    content     text        NOT NULL,
    checksum    text        NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
    issued_by   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at  timestamptz NOT NULL,
    UNIQUE (package_id, version),
    -- Solo la primera versión no reemplaza a ninguna.
    CHECK ((version = 1) = (previous_id IS NULL))
);

REVOKE UPDATE, DELETE ON conformance_report FROM kaoru_app;

-- Hito del calendario. Las ventanas de cambio no se copian aquí: se leen del CHG.
CREATE TABLE milestone (
    id           uuid PRIMARY KEY,
    title        text        NOT NULL,
    due_date     date        NOT NULL,
    notes        text        NOT NULL DEFAULT '',
    -- El responsable; nulo si nadie lo tiene.
    owner_id     text REFERENCES app_user (id) ON DELETE RESTRICT,
    -- Cumplido, el hito deja de contar como atraso.
    completed_at timestamptz,
    created_by   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL
);
CREATE INDEX milestone_due_date ON milestone (due_date);

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
        'deployed_to_production',
        'checklist_signed',
        'diagnostic_reviewed',
        'smoke_passed'));

-- La evidencia de un paso del checklist, de un DIAG o de un SMOKE es un DOC enlazado.
ALTER TABLE document_link
    DROP CONSTRAINT document_link_entity_type_check,
    ADD CONSTRAINT document_link_entity_type_check CHECK (entity_type IN (
        'requirement', 'change', 'functional_case', 'test_case', 'test_run', 'exploratory_session', 'defect',
        'package', 'deployment', 'checklist', 'diagnostic', 'smoke_test'));

-- +goose Down
DELETE FROM document_link WHERE entity_type IN ('checklist', 'diagnostic', 'smoke_test');
ALTER TABLE document_link
    DROP CONSTRAINT document_link_entity_type_check,
    ADD CONSTRAINT document_link_entity_type_check CHECK (entity_type IN (
        'requirement', 'change', 'functional_case', 'test_case', 'test_run', 'exploratory_session', 'defect',
        'package', 'deployment'));

DELETE FROM blocking_rule WHERE rule_key IN ('checklist_signed', 'diagnostic_reviewed', 'smoke_passed');
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

DROP TABLE milestone;
DROP TABLE conformance_report;
DROP TABLE smoke_test;
DROP TABLE diagnostic_review;
DROP TABLE diagnostic;
DROP TABLE checklist_signature;
DROP TABLE checklist;
DROP TABLE checklist_template_step;
DROP TABLE checklist_template;

-- +goose Up
-- Fase 6: planificación. Actividades con fechas y operadores, y la regla que avisa si un REQ entra
-- en pruebas sin el período de certificación de su CHG reservado.

-- ACT-: un bloque de trabajo de uno o varios operadores entre dos días (ambos incluidos, en UTC).
-- No se borra: se cancela, y una cancelada no cuenta para los solapes ni para la regla.
CREATE TABLE activity (
    id           uuid PRIMARY KEY,
    code         text        NOT NULL UNIQUE,
    kind         text        NOT NULL CHECK (kind IN ('certification', 'regression', 'production', 'other')),
    title        text        NOT NULL,
    -- El proyecto. La certificación siempre es de un CHG.
    change_id    uuid REFERENCES change (id) ON DELETE RESTRICT,
    start_date   date        NOT NULL,
    end_date     date        NOT NULL,
    notes        text        NOT NULL DEFAULT '',
    cancelled_at timestamptz,
    created_by   text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at   timestamptz NOT NULL,
    updated_at   timestamptz NOT NULL,
    CHECK (end_date >= start_date),
    CHECK (kind <> 'certification' OR change_id IS NOT NULL)
);
CREATE INDEX activity_dates ON activity (start_date, end_date);
CREATE INDEX activity_change ON activity (change_id);

-- Los operadores de una actividad. Se reemplazan al editar.
CREATE TABLE activity_operator (
    activity_id uuid NOT NULL REFERENCES activity (id) ON DELETE CASCADE,
    user_id     text NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    PRIMARY KEY (activity_id, user_id)
);
CREATE INDEX activity_operator_user ON activity_operator (user_id);

INSERT INTO permission (key, description)
VALUES ('activity.manage', 'Planificar actividades y asignar operadores');

-- El coordinador de QA (qa-admin) y change management planifican. [SUPUESTO] fase 6, pregunta 3.
INSERT INTO role_permission (role, permission_key)
VALUES ('qa-admin', 'activity.manage'),
       ('change-manager', 'activity.manage');

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
        'smoke_passed',
        'certification_planned'));

-- Al entrar en pruebas, el CHG del REQ tiene reservado el período de certificación con operadores.
-- Solo avisa (decisión del coordinador, 14 nov 2026); se endurece desde Administración.
INSERT INTO blocking_rule (id, transition_id, rule_key, params, route, blocks, created_at, updated_at)
SELECT gen_random_uuid(), tr.id, 'certification_planned', '{}'::jsonb, NULL, false, now(), now()
FROM workflow_state f
         JOIN workflow_state t ON t.entity_type = 'requirement' AND t.key = 'in_testing'
         JOIN workflow_transition tr ON tr.from_state_id = f.id AND tr.to_state_id = t.id
WHERE f.entity_type = 'requirement'
  AND f.key = 'delivered_to_qa';

-- +goose Down
DELETE FROM blocking_rule WHERE rule_key = 'certification_planned';
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
DELETE FROM role_permission WHERE permission_key = 'activity.manage';
DELETE FROM permission WHERE key = 'activity.manage';
DROP TABLE activity_operator;
DROP TABLE activity;

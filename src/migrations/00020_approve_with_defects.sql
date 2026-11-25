-- +goose Up
-- Un REQ «Con defectos» se puede aprobar por QA aceptando sus DEF (decisión del coordinador,
-- 20 nov 2026): cuando el requerimiento vino incompleto desde el inicio, el defecto no se pasa por
-- alto, pero tampoco detiene la aprobación. Los DEF siguen abiertos y ligados al REQ como traza, y
-- el motivo de aprobar así queda en la auditoría de la transición.
--
-- Los DEF abiertos al aprobar así quedan aceptados: quién, cuándo y por qué. Un DEF aceptado ya no
-- bloquea al REQ ni al PKG que lo incluya; uno que se abra después sí.
ALTER TABLE defect
    ADD COLUMN accepted_at     timestamptz,
    ADD COLUMN accepted_by     text REFERENCES app_user (id) ON DELETE RESTRICT,
    ADD COLUMN accepted_reason text,
    ADD CONSTRAINT defect_acceptance_complete CHECK (
        (accepted_at IS NULL AND accepted_by IS NULL AND accepted_reason IS NULL) OR
        (accepted_at IS NOT NULL AND accepted_by IS NOT NULL AND accepted_reason IS NOT NULL));

INSERT INTO permission (key, description)
VALUES ('req.approve_with_defects', 'Aprobar un REQ por QA aceptando sus defectos abiertos');

INSERT INTO role_permission (role, permission_key)
VALUES ('qa', 'req.approve_with_defects'),
       ('qa-admin', 'req.approve_with_defects');

INSERT INTO workflow_transition (id, from_state_id, to_state_id, permission_key, requires_reason, created_at, updated_at)
SELECT gen_random_uuid(), f.id, t.id, 'req.approve_with_defects', true, now(), now()
FROM workflow_state f
         JOIN workflow_state t ON t.entity_type = 'requirement' AND t.key = 'approved_qa'
WHERE f.entity_type = 'requirement'
  AND f.key = 'with_defects';

-- Las mismas condiciones que la aprobación normal, pero solo avisan: quien aprueba ve qué acepta.
-- Se endurecen desde Administración si el negocio lo pide.
INSERT INTO blocking_rule (id, transition_id, rule_key, params, route, blocks, created_at, updated_at)
SELECT gen_random_uuid(), tr.id, v.rule_key, v.params::jsonb, NULL, false, now(), now()
FROM (VALUES
    ('min_coverage',                 '{"high": 100, "medium": 100, "low": 100}'),
    ('no_open_defect_with_severity', '{"severity": "high"}'),
    ('no_open_defect_with_severity', '{"severity": "medium"}'),
    ('links_have_snapshot',          '{}')
) AS v (rule_key, params)
         JOIN workflow_state f ON f.entity_type = 'requirement' AND f.key = 'with_defects'
         JOIN workflow_state t ON t.entity_type = 'requirement' AND t.key = 'approved_qa'
         JOIN workflow_transition tr ON tr.from_state_id = f.id AND tr.to_state_id = t.id;

-- +goose Down
DELETE FROM blocking_rule
WHERE transition_id IN (SELECT id FROM workflow_transition WHERE permission_key = 'req.approve_with_defects');
DELETE FROM workflow_transition WHERE permission_key = 'req.approve_with_defects';
DELETE FROM role_permission WHERE permission_key = 'req.approve_with_defects';
DELETE FROM permission WHERE key = 'req.approve_with_defects';
ALTER TABLE defect
    DROP CONSTRAINT defect_acceptance_complete,
    DROP COLUMN accepted_reason,
    DROP COLUMN accepted_by,
    DROP COLUMN accepted_at;

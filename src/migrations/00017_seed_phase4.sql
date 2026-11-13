-- +goose Up
-- Valores predeterminados de la fase 4: permisos y reglas de bloqueo del PKG por ruta.
-- `qa-admin` los edita desde Administración.

INSERT INTO permission (key, description)
VALUES ('chk.manage_template', 'Publicar una versión de la plantilla del checklist de producción'),
       ('chk.start', 'Abrir el checklist de producción de un PKG'),
       ('chk.sign', 'Firmar un paso del checklist de producción'),
       ('diag.submit', 'Entregar un diagnóstico posterior al despliegue'),
       ('diag.review', 'Revisar un diagnóstico y marcar su resultado'),
       ('smoke.create', 'Registrar una prueba de humo posterior al despliegue'),
       ('rpt.issue', 'Emitir un informe de conformidad'),
       ('milestone.manage', 'Mantener los hitos del calendario');

-- Reparto inicial según la tabla de roles del requerimiento (§4). QA no firma el checklist (regla 6).
INSERT INTO role_permission (role, permission_key)
VALUES ('change-manager', 'chk.manage_template'),
       ('change-manager', 'chk.start'),
       ('change-manager', 'chk.sign'),
       ('ops', 'chk.sign'),
       ('dev', 'diag.submit'),
       ('ops', 'diag.submit'),
       ('qa', 'diag.review'),
       ('qa-admin', 'diag.review'),
       ('qa', 'smoke.create'),
       ('qa-admin', 'smoke.create'),
       ('qa', 'rpt.issue'),
       ('qa-admin', 'rpt.issue'),
       ('change-manager', 'milestone.manage');

-- Reglas del PKG que dependen de la ruta (requerimiento §8.2). La ruta de un PKG es la más exigente
-- de las de sus REQ. Un fix menor no exige ninguna de las tres. [SUPUESTO]
INSERT INTO blocking_rule (id, transition_id, rule_key, params, route, blocks, created_at, updated_at)
SELECT gen_random_uuid(), tr.id, v.rule_key, '{}'::jsonb, v.route, true, now(), now()
FROM (VALUES
    ('approved_for_production', 'deployed', 'checklist_signed',    'high'),
    ('deployed',                'verified', 'diagnostic_reviewed', 'high'),
    ('deployed',                'verified', 'smoke_passed',        'light')
) AS v (from_key, to_key, rule_key, route)
JOIN workflow_state f ON f.entity_type = 'package' AND f.key = v.from_key
JOIN workflow_state t ON t.entity_type = 'package' AND t.key = v.to_key
JOIN workflow_transition tr ON tr.from_state_id = f.id AND tr.to_state_id = t.id;

-- +goose Down
DELETE FROM blocking_rule WHERE rule_key IN ('checklist_signed', 'diagnostic_reviewed', 'smoke_passed');
DELETE FROM role_permission
WHERE permission_key IN ('chk.manage_template', 'chk.start', 'chk.sign', 'diag.submit', 'diag.review', 'smoke.create',
                         'rpt.issue', 'milestone.manage');
DELETE FROM permission
WHERE key IN ('chk.manage_template', 'chk.start', 'chk.sign', 'diag.submit', 'diag.review', 'smoke.create',
              'rpt.issue', 'milestone.manage');

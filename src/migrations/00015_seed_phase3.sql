-- +goose Up
-- Valores predeterminados de la fase 3: flujo del PKG (requerimiento §8.4), permisos y reglas.
-- `qa-admin` los edita desde Administración.

INSERT INTO workflow_state (id, entity_type, key, name, position, is_initial, is_final, created_at, updated_at)
SELECT gen_random_uuid(), 'package', v.key, v.name, v.position, v.is_initial, v.is_final, now(), now()
FROM (VALUES
    ('in_assembly',             'En armado',                1, true,  false),
    ('in_testing',              'En pruebas',               2, false, false),
    ('ready_to_release',        'Listo para liberar',       3, false, false),
    ('approved_for_production', 'Aprobado para producción', 4, false, false),
    ('deployed',                'Desplegado',               5, false, false),
    ('verified',                'Verificado',               6, false, true)
) AS v (key, name, position, is_initial, is_final);

INSERT INTO permission (key, description)
VALUES ('comp.manage', 'Mantener el catálogo de componentes'),
       ('build.register', 'Registrar un BUILD'),
       ('pkg.build', 'Armar un PKG: crearlo y cambiar sus BUILD y REQ'),
       ('pkg.start_testing', 'Pasar un PKG a pruebas'),
       ('pkg.reopen', 'Devolver un PKG a armado'),
       ('pkg.mark_ready', 'Dejar un PKG listo para liberar'),
       ('pkg.approve_production', 'Aprobar un PKG para producción'),
       ('pkg.deploy', 'Marcar un PKG como desplegado'),
       ('pkg.verify', 'Verificar un PKG después del despliegue'),
       ('regression.manage', 'Proponer, ajustar y confirmar la selección de regresión'),
       ('env.manage', 'Mantener los ambientes'),
       ('dep.create', 'Registrar un despliegue en un ambiente de prueba'),
       ('dep.create_production', 'Registrar un despliegue en producción');

-- Quién arma el PKG está [POR DEFINIR] (regla de negocio 10): se deja en dev, como `req.package`.
INSERT INTO role_permission (role, permission_key)
VALUES ('dev', 'comp.manage'),
       ('dev', 'build.register'),
       ('dev', 'pkg.build'),
       ('qa', 'comp.manage'),
       ('qa', 'pkg.start_testing'),
       ('qa', 'pkg.reopen'),
       ('qa', 'pkg.mark_ready'),
       ('qa', 'pkg.verify'),
       ('qa', 'regression.manage'),
       ('qa', 'env.manage'),
       ('qa', 'dep.create'),
       ('qa-admin', 'comp.manage'),
       ('qa-admin', 'pkg.start_testing'),
       ('qa-admin', 'pkg.reopen'),
       ('qa-admin', 'pkg.mark_ready'),
       ('qa-admin', 'pkg.verify'),
       ('qa-admin', 'regression.manage'),
       ('qa-admin', 'env.manage'),
       ('qa-admin', 'dep.create'),
       ('change-manager', 'pkg.approve_production'),
       ('change-manager', 'pkg.deploy'),
       ('change-manager', 'dep.create_production');

INSERT INTO workflow_transition (id, from_state_id, to_state_id, permission_key, requires_reason, created_at, updated_at)
SELECT gen_random_uuid(), f.id, t.id, v.permission_key, v.requires_reason, now(), now()
FROM (VALUES
    ('in_assembly',             'in_testing',              'pkg.start_testing',      false),
    -- Volver a armado no está en el requerimiento: [SUPUESTO] para poder corregir el contenido.
    ('in_testing',              'in_assembly',             'pkg.reopen',             true),
    ('in_testing',              'ready_to_release',        'pkg.mark_ready',         false),
    ('ready_to_release',        'approved_for_production', 'pkg.approve_production', false),
    ('approved_for_production', 'deployed',                'pkg.deploy',             false),
    ('deployed',                'verified',                'pkg.verify',             false)
) AS v (from_key, to_key, permission_key, requires_reason)
JOIN workflow_state f ON f.entity_type = 'package' AND f.key = v.from_key
JOIN workflow_state t ON t.entity_type = 'package' AND t.key = v.to_key;

-- Reglas de bloqueo del PKG. La de severidad alta es la regla de negocio 2; las demás son [SUPUESTO].
INSERT INTO blocking_rule (id, transition_id, rule_key, params, route, blocks, created_at, updated_at)
SELECT gen_random_uuid(), tr.id, v.rule_key, v.params::jsonb, NULL, true, now(), now()
FROM (VALUES
    ('in_assembly',             'in_testing',       'package_has_content',          '{}'),
    ('in_assembly',             'in_testing',       'regression_confirmed',         '{}'),
    ('in_testing',              'ready_to_release', 'regression_passed',            '{}'),
    ('in_testing',              'ready_to_release', 'no_open_defect_with_severity', '{"severity": "high"}'),
    ('approved_for_production', 'deployed',         'deployed_to_production',       '{}')
) AS v (from_key, to_key, rule_key, params)
JOIN workflow_state f ON f.entity_type = 'package' AND f.key = v.from_key
JOIN workflow_state t ON t.entity_type = 'package' AND t.key = v.to_key
JOIN workflow_transition tr ON tr.from_state_id = f.id AND tr.to_state_id = t.id;

-- +goose Down
DELETE FROM blocking_rule
WHERE transition_id IN (SELECT tr.id
                        FROM workflow_transition tr
                                 JOIN workflow_state f ON f.id = tr.from_state_id
                        WHERE f.entity_type = 'package');
DELETE FROM workflow_transition
WHERE from_state_id IN (SELECT id FROM workflow_state WHERE entity_type = 'package');
DELETE FROM role_permission
WHERE permission_key IN ('comp.manage', 'build.register', 'pkg.build', 'pkg.start_testing', 'pkg.reopen', 'pkg.mark_ready',
                         'pkg.approve_production', 'pkg.deploy', 'pkg.verify', 'regression.manage', 'env.manage',
                         'dep.create', 'dep.create_production');
DELETE FROM permission
WHERE key IN ('comp.manage', 'build.register', 'pkg.build', 'pkg.start_testing', 'pkg.reopen', 'pkg.mark_ready',
              'pkg.approve_production', 'pkg.deploy', 'pkg.verify', 'regression.manage', 'env.manage',
              'dep.create', 'dep.create_production');
DELETE FROM workflow_state WHERE entity_type = 'package';

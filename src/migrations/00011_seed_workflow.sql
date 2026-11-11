-- +goose Up
-- Valores predeterminados de los flujos (requerimiento §8). `qa-admin` los edita
-- desde Administración; esta migración solo deja el punto de partida.

INSERT INTO workflow_state (id, entity_type, key, name, position, is_initial, is_final, created_at, updated_at)
SELECT gen_random_uuid(), v.entity_type, v.key, v.name, v.position, v.is_initial, v.is_final, now(), now()
FROM (VALUES
    -- REQ (§8.1)
    ('requirement', 'registered',      'Registrado',                  1,    true,  false),
    ('requirement', 'risk_assessed',   'Riesgo evaluado',             2,    false, false),
    ('requirement', 'test_design',     'Diseño de pruebas',           3,    false, false),
    ('requirement', 'in_development',  'En desarrollo',               4,    false, false),
    ('requirement', 'delivered_to_qa', 'Entregado a QA',              5,    false, false),
    ('requirement', 'in_testing',      'En pruebas',                  6,    false, false),
    ('requirement', 'with_defects',    'Con defectos',                NULL, false, false),
    ('requirement', 'approved_qa',     'Aprobado QA',                 7,    false, false),
    ('requirement', 'in_package',      'En paquete',                  8,    false, false),
    ('requirement', 'released',        'Liberado',                    9,    false, false),
    ('requirement', 'verified',        'Verificado post-despliegue',  10,   false, false),
    ('requirement', 'closed',          'Cerrado',                     11,   false, true),
    ('requirement', 'rejected',        'Rechazado',                   NULL, false, true),
    ('requirement', 'deferred',        'Diferido a otro paquete',     NULL, false, false),
    -- DEF (§8.3)
    ('defect', 'open',             'Abierto',             1,    true,  false),
    ('defect', 'triaged',          'Triado',              2,    false, false),
    ('defect', 'in_fix',           'En corrección',       3,    false, false),
    ('defect', 'ready_for_retest', 'Listo para reprueba', 4,    false, false),
    ('defect', 'closed',           'Cerrado',             5,    false, true),
    ('defect', 'reopened',         'Reabierto',           NULL, false, false),
    ('defect', 'deferred',         'Diferido',            NULL, false, false),
    -- CHG: el requerimiento no define sus estados. Propuesta [SUPUESTO] (decisión D17).
    ('change', 'registered',  'Registrado', 1,    true,  false),
    ('change', 'approved',    'Aprobado',   2,    false, false),
    ('change', 'in_progress', 'En curso',   3,    false, false),
    ('change', 'released',    'Liberado',   4,    false, false),
    ('change', 'closed',      'Cerrado',    5,    false, true),
    ('change', 'rejected',    'Rechazado',  NULL, false, true)
) AS v (entity_type, key, name, position, is_initial, is_final);

INSERT INTO permission (key, description)
VALUES ('chg.create', 'Crear un CHG'),
       ('chg.edit', 'Editar un CHG'),
       ('chg.approve', 'Aprobar un CHG'),
       ('chg.reject', 'Rechazar un CHG'),
       ('chg.start', 'Poner un CHG en curso'),
       ('chg.release', 'Marcar un CHG como liberado'),
       ('chg.close', 'Cerrar un CHG'),
       ('req.create', 'Registrar un REQ'),
       ('req.edit', 'Editar un REQ'),
       ('risk.assess', 'Evaluar el riesgo de un REQ'),
       ('req.start_design', 'Pasar un REQ a diseño de pruebas'),
       ('req.start_development', 'Pasar un REQ a desarrollo'),
       ('req.deliver', 'Entregar un REQ a QA'),
       ('req.start_testing', 'Iniciar o retomar las pruebas de un REQ'),
       ('req.report_defects', 'Marcar un REQ con defectos'),
       ('req.approve_qa', 'Aprobar un REQ por QA'),
       ('req.package', 'Incluir un REQ en un paquete'),
       ('req.release', 'Marcar un REQ como liberado'),
       ('req.verify', 'Verificar un REQ después del despliegue'),
       ('req.close', 'Cerrar un REQ'),
       ('req.reject', 'Rechazar un REQ'),
       ('req.defer', 'Aceptar que un REQ se difiera a otro paquete'),
       ('entry_contract.check', 'Marcar ítems del contrato de entrada'),
       ('link.manage', 'Agregar, editar y quitar enlaces'),
       ('doc.upload', 'Adjuntar documentos y subir versiones'),
       ('comment.create', 'Comentar'),
       ('testdesign.manage', 'Definir UC y TC'),
       ('run.create', 'Registrar un RUN'),
       ('exp.create', 'Registrar un EXP'),
       ('def.create', 'Abrir un DEF'),
       ('def.edit', 'Editar un DEF'),
       ('def.triage', 'Triar un DEF'),
       ('def.fix', 'Corregir un DEF y dejarlo listo para reprueba'),
       ('def.close', 'Cerrar un DEF'),
       ('def.reopen', 'Reabrir un DEF'),
       ('def.defer', 'Aceptar que un DEF se difiera'),
       ('audit.read', 'Consultar y exportar la auditoría'),
       ('import.run', 'Importar la hoja'),
       ('admin.configure', 'Configurar flujos, reglas de bloqueo, catálogos y permisos');

-- Permisos por rol (requerimiento §4). `qa-admin` recibe además todo lo de `qa`.
INSERT INTO role_permission (role, permission_key)
VALUES ('change-manager', 'chg.create'),
       ('change-manager', 'chg.edit'),
       ('change-manager', 'chg.approve'),
       ('change-manager', 'chg.reject'),
       ('change-manager', 'chg.start'),
       ('change-manager', 'chg.release'),
       ('change-manager', 'chg.close'),
       ('change-manager', 'req.create'),
       ('change-manager', 'req.edit'),
       ('change-manager', 'req.release'),
       ('change-manager', 'req.close'),
       ('change-manager', 'req.reject'),
       ('change-manager', 'req.defer'),
       ('change-manager', 'def.defer'),
       ('change-manager', 'link.manage'),
       ('change-manager', 'doc.upload'),
       ('change-manager', 'comment.create'),
       ('change-manager', 'audit.read'),
       ('qa', 'req.edit'),
       ('qa', 'risk.assess'),
       ('qa', 'req.start_design'),
       ('qa', 'req.start_testing'),
       ('qa', 'req.report_defects'),
       ('qa', 'req.approve_qa'),
       ('qa', 'req.verify'),
       ('qa', 'link.manage'),
       ('qa', 'doc.upload'),
       ('qa', 'comment.create'),
       ('qa', 'testdesign.manage'),
       ('qa', 'run.create'),
       ('qa', 'exp.create'),
       ('qa', 'def.create'),
       ('qa', 'def.edit'),
       ('qa', 'def.triage'),
       ('qa', 'def.close'),
       ('qa', 'def.reopen'),
       ('qa', 'audit.read'),
       ('dev', 'req.edit'),
       ('dev', 'req.start_development'),
       ('dev', 'req.deliver'),
       ('dev', 'req.package'),
       ('dev', 'entry_contract.check'),
       ('dev', 'link.manage'),
       ('dev', 'doc.upload'),
       ('dev', 'comment.create'),
       ('dev', 'def.fix'),
       ('ops', 'doc.upload'),
       ('ops', 'comment.create'),
       ('auditor', 'audit.read'),
       ('qa-admin', 'admin.configure'),
       ('qa-admin', 'import.run');

INSERT INTO role_permission (role, permission_key)
SELECT 'qa-admin', permission_key
FROM role_permission
WHERE role = 'qa';

INSERT INTO workflow_transition (id, from_state_id, to_state_id, permission_key, requires_reason, created_at, updated_at)
SELECT gen_random_uuid(), f.id, t.id, v.permission_key, v.requires_reason, now(), now()
FROM (VALUES
    -- REQ: camino principal
    ('requirement', 'registered',      'risk_assessed',   'risk.assess',           false),
    ('requirement', 'risk_assessed',   'test_design',     'req.start_design',      false),
    ('requirement', 'test_design',     'in_development',  'req.start_development', false),
    ('requirement', 'in_development',  'delivered_to_qa', 'req.deliver',           false),
    ('requirement', 'delivered_to_qa', 'in_testing',      'req.start_testing',     false),
    ('requirement', 'in_testing',      'with_defects',    'req.report_defects',    false),
    ('requirement', 'with_defects',    'in_testing',      'req.start_testing',     false),
    ('requirement', 'in_testing',      'approved_qa',     'req.approve_qa',        false),
    ('requirement', 'approved_qa',     'in_package',      'req.package',           false),
    ('requirement', 'in_package',      'released',        'req.release',           false),
    ('requirement', 'released',        'verified',        'req.verify',            false),
    ('requirement', 'verified',        'closed',          'req.close',             false),
    -- REQ: rechazo, antes de aprobarse
    ('requirement', 'registered',      'rejected',        'req.reject',            true),
    ('requirement', 'risk_assessed',   'rejected',        'req.reject',            true),
    ('requirement', 'test_design',     'rejected',        'req.reject',            true),
    ('requirement', 'in_development',  'rejected',        'req.reject',            true),
    ('requirement', 'delivered_to_qa', 'rejected',        'req.reject',            true),
    ('requirement', 'in_testing',      'rejected',        'req.reject',            true),
    ('requirement', 'with_defects',    'rejected',        'req.reject',            true),
    -- REQ: diferido a otro paquete (lo acepta change management)
    ('requirement', 'in_testing',      'deferred',        'req.defer',             true),
    ('requirement', 'with_defects',    'deferred',        'req.defer',             true),
    ('requirement', 'approved_qa',     'deferred',        'req.defer',             true),
    ('requirement', 'in_package',      'deferred',        'req.defer',             true),
    ('requirement', 'deferred',        'in_testing',      'req.start_testing',     false),
    -- DEF
    ('defect', 'open',             'triaged',          'def.triage', false),
    ('defect', 'triaged',          'in_fix',           'def.fix',    false),
    ('defect', 'in_fix',           'ready_for_retest', 'def.fix',    false),
    ('defect', 'ready_for_retest', 'closed',           'def.close',  false),
    ('defect', 'ready_for_retest', 'reopened',         'def.reopen', true),
    ('defect', 'closed',           'reopened',         'def.reopen', true),
    ('defect', 'reopened',         'in_fix',           'def.fix',    false),
    ('defect', 'triaged',          'deferred',         'def.defer',  true),
    ('defect', 'deferred',         'triaged',          'def.triage', false),
    -- CHG [SUPUESTO]
    ('change', 'registered',  'approved',    'chg.approve', false),
    ('change', 'registered',  'rejected',    'chg.reject',  true),
    ('change', 'approved',    'in_progress', 'chg.start',   false),
    ('change', 'in_progress', 'released',    'chg.release', false),
    ('change', 'released',    'closed',      'chg.close',   false)
) AS v (entity_type, from_key, to_key, permission_key, requires_reason)
JOIN workflow_state f ON f.entity_type = v.entity_type AND f.key = v.from_key
JOIN workflow_state t ON t.entity_type = v.entity_type AND t.key = v.to_key;

-- Reglas de bloqueo. La cobertura mínima por nivel de riesgo está [POR DEFINIR]:
-- se deja en 100 % para los tres niveles hasta que QA la fije.
INSERT INTO blocking_rule (id, transition_id, rule_key, params, route, blocks, created_at, updated_at)
SELECT gen_random_uuid(), tr.id, v.rule_key, v.params::jsonb, v.route, v.blocks, now(), now()
FROM (VALUES
    ('requirement', 'registered',     'risk_assessed',   'risk_assessed',                    '{}',                                     NULL,   true),
    ('requirement', 'registered',     'risk_assessed',   'risk_has_quality_characteristics', '{}',                                     'high', true),
    ('requirement', 'in_development', 'delivered_to_qa', 'entry_contract_complete',          '{}',                                     NULL,   true),
    ('requirement', 'in_testing',     'approved_qa',     'min_coverage',                     '{"high": 100, "medium": 100, "low": 100}', NULL,   true),
    ('requirement', 'in_testing',     'approved_qa',     'no_open_defect_with_severity',     '{"severity": "high"}',                   NULL,   true),
    ('requirement', 'in_testing',     'approved_qa',     'links_have_snapshot',              '{}',                                     NULL,   false),
    ('defect',      'triaged',        'deferred',        'defect_severity_is_not',           '{"severity": "high"}',                   NULL,   true)
) AS v (entity_type, from_key, to_key, rule_key, params, route, blocks)
JOIN workflow_state f ON f.entity_type = v.entity_type AND f.key = v.from_key
JOIN workflow_state t ON t.entity_type = v.entity_type AND t.key = v.to_key
JOIN workflow_transition tr ON tr.from_state_id = f.id AND tr.to_state_id = t.id;

-- +goose Down
DELETE FROM blocking_rule;
DELETE FROM workflow_transition;
DELETE FROM role_permission;
DELETE FROM permission;
DELETE FROM workflow_state;

-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetSystem :one
SELECT id, key, name, default_route
FROM system_under_test
WHERE key = $1;

-- name: LockPackage :one
-- Bloquea el PKG hasta el fin de la transacción: las firmas de un checklist y la revisión de un DIAG
-- son de solo inserción, así que lo que se bloquea es la fila del PKG al que pertenecen.
SELECT id
FROM package
WHERE id = $1
    FOR UPDATE;

-- name: NextTemplateVersion :one
SELECT (coalesce(max(t.version), 0) + 1)::integer AS version
FROM checklist_template t
         JOIN system_under_test s ON s.id = t.system_id
WHERE s.key = $1;

-- name: InsertTemplate :exec
INSERT INTO checklist_template (id, system_id, version, created_by, created_at)
VALUES ($1, (SELECT s.id FROM system_under_test s WHERE s.key = sqlc.arg(system_key)), $2, $3, $4);

-- name: InsertTemplateStep :exec
INSERT INTO checklist_template_step (template_id, position, title, description)
VALUES ($1, $2, $3, $4);

-- name: ListTemplates :many
-- Todas las versiones, la más reciente de cada sistema primero.
SELECT t.id, t.version, t.created_by, u.display_name AS created_by_name, t.created_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route
FROM checklist_template t
         JOIN system_under_test s ON s.id = t.system_id
         JOIN app_user u ON u.id = t.created_by
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
ORDER BY s.key, t.version DESC;

-- name: GetLatestTemplate :one
SELECT t.id, t.version, t.created_by, u.display_name AS created_by_name, t.created_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route
FROM checklist_template t
         JOIN system_under_test s ON s.id = t.system_id
         JOIN app_user u ON u.id = t.created_by
WHERE s.key = $1
ORDER BY t.version DESC
LIMIT 1;

-- name: ListTemplateSteps :many
SELECT template_id, position, title, description
FROM checklist_template_step
WHERE template_id = ANY (sqlc.arg(template_ids)::uuid[])
ORDER BY template_id, position;

-- name: GetPackageRef :one
-- Lee el PKG del checklist: es el sentido de la llave foránea checklist.package_id.
SELECT p.id, p.code, p.title, s.key AS system_key
FROM package p
         JOIN system_under_test s ON s.id = p.system_id
WHERE p.code = $1;

-- name: GetEnvironmentRef :one
-- Lee el ambiente del checklist: es el sentido de la llave foránea checklist.environment_id.
SELECT e.id, e.code, e.name, e.kind, (e.retired_at IS NOT NULL)::boolean AS retired, s.key AS system_key
FROM environment e
         JOIN system_under_test s ON s.id = e.system_id
WHERE e.code = $1;

-- name: InsertChecklist :exec
INSERT INTO checklist (id, code, package_id, environment_id, template_id, started_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: InsertSignature :exec
INSERT INTO checklist_signature (checklist_id, position, comment, signed_by, signed_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetChecklistByCode :one
SELECT c.id, c.code, c.template_id, t.version AS template_version,
       c.started_by, u.display_name AS started_by_name, c.created_at,
       p.id AS package_id, p.code AS package_code, p.title AS package_title,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind
FROM checklist c
         JOIN checklist_template t ON t.id = c.template_id
         JOIN package p ON p.id = c.package_id
         JOIN environment e ON e.id = c.environment_id
         JOIN app_user u ON u.id = c.started_by
WHERE c.code = $1;

-- name: ListChecklists :many
SELECT c.id, c.code, c.template_id, t.version AS template_version,
       c.started_by, u.display_name AS started_by_name, c.created_at,
       p.id AS package_id, p.code AS package_code, p.title AS package_title,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind
FROM checklist c
         JOIN checklist_template t ON t.id = c.template_id
         JOIN package p ON p.id = c.package_id
         JOIN environment e ON e.id = c.environment_id
         JOIN app_user u ON u.id = c.started_by
WHERE (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code))
ORDER BY c.created_at DESC, c.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountChecklists :one
SELECT count(*)
FROM checklist c
         JOIN package p ON p.id = c.package_id
WHERE (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code));

-- name: ListSignatures :many
SELECT g.checklist_id, g.position, g.comment, g.signed_by, u.display_name AS signed_by_name, g.signed_at
FROM checklist_signature g
         JOIN app_user u ON u.id = g.signed_by
WHERE g.checklist_id = ANY (sqlc.arg(checklist_ids)::uuid[])
ORDER BY g.checklist_id, g.position;

-- name: GetDeploymentRef :one
-- Lee el despliegue de un DIAG o de un SMOKE: es el sentido de sus llaves foráneas.
SELECT d.id, d.code, d.executed_at,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind,
       p.id AS package_id, p.code AS package_code, p.title AS package_title
FROM deployment d
         JOIN environment e ON e.id = d.environment_id
         LEFT JOIN package p ON p.id = d.package_id
WHERE d.code = $1;

-- name: InsertDiagnostic :exec
INSERT INTO diagnostic (id, code, deployment_id, notes, submitted_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertDiagnosticReview :exec
INSERT INTO diagnostic_review (diagnostic_id, result, notes, reviewed_by, reviewed_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetDiagnosticByCode :one
SELECT g.id, g.code, g.notes, g.submitted_by, su.display_name AS submitted_by_name, g.created_at,
       d.id AS deployment_id, d.code AS deployment_code, d.executed_at AS deployment_executed_at,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind,
       p.id AS package_id, p.code AS package_code, p.title AS package_title,
       r.result AS review_result, r.notes AS review_notes, r.reviewed_by, ru.display_name AS reviewed_by_name, r.reviewed_at
FROM diagnostic g
         JOIN deployment d ON d.id = g.deployment_id
         JOIN environment e ON e.id = d.environment_id
         JOIN package p ON p.id = d.package_id
         JOIN app_user su ON su.id = g.submitted_by
         LEFT JOIN diagnostic_review r ON r.diagnostic_id = g.id
         LEFT JOIN app_user ru ON ru.id = r.reviewed_by
WHERE g.code = $1;

-- name: ListDiagnostics :many
SELECT g.id, g.code, g.notes, g.submitted_by, su.display_name AS submitted_by_name, g.created_at,
       d.id AS deployment_id, d.code AS deployment_code, d.executed_at AS deployment_executed_at,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind,
       p.id AS package_id, p.code AS package_code, p.title AS package_title,
       r.result AS review_result, r.notes AS review_notes, r.reviewed_by, ru.display_name AS reviewed_by_name, r.reviewed_at
FROM diagnostic g
         JOIN deployment d ON d.id = g.deployment_id
         JOIN environment e ON e.id = d.environment_id
         JOIN package p ON p.id = d.package_id
         JOIN app_user su ON su.id = g.submitted_by
         LEFT JOIN diagnostic_review r ON r.diagnostic_id = g.id
         LEFT JOIN app_user ru ON ru.id = r.reviewed_by
WHERE (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code))
ORDER BY g.created_at DESC, g.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountDiagnostics :one
SELECT count(*)
FROM diagnostic g
         JOIN deployment d ON d.id = g.deployment_id
         JOIN package p ON p.id = d.package_id
WHERE (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code));

-- name: InsertSmokeTest :exec
INSERT INTO smoke_test (id, code, deployment_id, result, notes, executed_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetSmokeTestByCode :one
SELECT k.id, k.code, k.result, k.notes, k.executed_by, u.display_name AS executed_by_name, k.created_at,
       d.id AS deployment_id, d.code AS deployment_code, d.executed_at AS deployment_executed_at,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind,
       p.id AS package_id, p.code AS package_code, p.title AS package_title
FROM smoke_test k
         JOIN deployment d ON d.id = k.deployment_id
         JOIN environment e ON e.id = d.environment_id
         JOIN package p ON p.id = d.package_id
         JOIN app_user u ON u.id = k.executed_by
WHERE k.code = $1;

-- name: ListSmokeTests :many
SELECT k.id, k.code, k.result, k.notes, k.executed_by, u.display_name AS executed_by_name, k.created_at,
       d.id AS deployment_id, d.code AS deployment_code, d.executed_at AS deployment_executed_at,
       e.id AS environment_id, e.code AS environment_code, e.name AS environment_name, e.kind AS environment_kind,
       p.id AS package_id, p.code AS package_code, p.title AS package_title
FROM smoke_test k
         JOIN deployment d ON d.id = k.deployment_id
         JOIN environment e ON e.id = d.environment_id
         JOIN package p ON p.id = d.package_id
         JOIN app_user u ON u.id = k.executed_by
WHERE (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code))
ORDER BY k.created_at DESC, k.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountSmokeTests :one
SELECT count(*)
FROM smoke_test k
         JOIN deployment d ON d.id = k.deployment_id
         JOIN package p ON p.id = d.package_id
WHERE (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code));

-- name: GetPackageFacts :one
-- Lo último de cada cosa para un PKG: su checklist con los pasos que le faltan, su DIAG y su SMOKE.
SELECT (lc.id IS NOT NULL)::boolean AS checklist_exists,
       coalesce((SELECT count(*) FROM checklist_template_step ts WHERE ts.template_id = lc.template_id)
                    - (SELECT count(*) FROM checklist_signature cs WHERE cs.checklist_id = lc.id), 0)::integer AS checklist_pending,
       (CASE
            WHEN ld.id IS NULL THEN ''
            ELSE coalesce((SELECT r.result FROM diagnostic_review r WHERE r.diagnostic_id = ld.id), 'pending')
           END)::text AS diagnostic,
       coalesce(ls.result, '')::text AS smoke
FROM package p
         LEFT JOIN LATERAL (SELECT c.id, c.template_id
                            FROM checklist c
                            WHERE c.package_id = p.id
                            ORDER BY c.created_at DESC, c.code DESC
                            LIMIT 1) lc ON true
         LEFT JOIN LATERAL (SELECT g.id
                            FROM diagnostic g
                                     JOIN deployment d ON d.id = g.deployment_id
                            WHERE d.package_id = p.id
                            ORDER BY g.created_at DESC, g.code DESC
                            LIMIT 1) ld ON true
         LEFT JOIN LATERAL (SELECT k.result
                            FROM smoke_test k
                                     JOIN deployment d ON d.id = k.deployment_id
                            WHERE d.package_id = p.id
                            ORDER BY k.created_at DESC, k.code DESC
                            LIMIT 1) ls ON true
WHERE p.code = $1;

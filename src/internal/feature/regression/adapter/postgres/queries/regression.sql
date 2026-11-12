-- name: GetPackage :one
-- Lee el PKG dueño de la selección: es el sentido de la llave foránea regression_selection.package_id.
SELECT p.id, p.code, s.key AS system_key
FROM package p
         JOIN system_under_test s ON s.id = p.system_id
WHERE p.code = $1;

-- name: LockPackage :one
-- Bloquea el PKG hasta el fin de la transacción: dos escrituras sobre la misma selección van una tras
-- otra, exista ya la selección o no.
SELECT id
FROM package
WHERE code = $1
    FOR UPDATE;

-- name: GetTestCase :one
SELECT tc.id, tc.code, s.key AS system_key, (tc.retired_at IS NOT NULL)::boolean AS retired
FROM test_case tc
         JOIN system_under_test s ON s.id = tc.system_id
WHERE tc.code = $1;

-- name: ListCandidates :many
-- Lo que el sistema propone: los TC activos del sistema del PKG que ejercitan algún componente
-- tocado por un REQ del PKG (impacto), más los críticos.
SELECT tc.id, tc.code,
       EXISTS (SELECT 1
               FROM test_case_component tcc
                        JOIN requirement_component rc ON rc.component_id = tcc.component_id
                        JOIN package_requirement pr ON pr.requirement_id = rc.requirement_id
               WHERE tcc.test_case_id = tc.id
                 AND pr.package_id = p.id)::boolean AS by_impact
FROM package p
         JOIN test_case tc ON tc.system_id = p.system_id
WHERE p.id = $1
  AND tc.retired_at IS NULL
  AND (tc.is_critical
    OR EXISTS (SELECT 1
               FROM test_case_component tcc
                        JOIN requirement_component rc ON rc.component_id = tcc.component_id
                        JOIN package_requirement pr ON pr.requirement_id = rc.requirement_id
               WHERE tcc.test_case_id = tc.id
                 AND pr.package_id = p.id))
ORDER BY tc.code;

-- name: GetSelection :one
SELECT rs.package_id, rs.proposed_by, pu.display_name AS proposed_by_name, rs.proposed_at,
       rs.confirmed_by, cu.display_name AS confirmed_by_name, rs.confirmed_at
FROM regression_selection rs
         JOIN app_user pu ON pu.id = rs.proposed_by
         LEFT JOIN app_user cu ON cu.id = rs.confirmed_by
WHERE rs.package_id = $1;

-- name: ListItems :many
-- Cada TC de la selección con su último RUN sobre un BUILD del PKG: el de ejecución más reciente y,
-- si empatan, el registrado después.
SELECT ri.test_case_id, tc.code AS test_case_code, tc.title AS test_case_title, tc.is_critical,
       ri.origin, ri.proposed, ri.included, ri.reason, ri.decided_by, du.display_name AS decided_by_name, ri.decided_at,
       lr.code AS run_code, lr.result AS run_result, lr.build_version AS run_build_version,
       lr.environment_name AS run_environment_name, lr.executed_at AS run_executed_at,
       lb.code AS run_build_code, le.code AS run_environment_code
FROM regression_item ri
         JOIN test_case tc ON tc.id = ri.test_case_id
         LEFT JOIN app_user du ON du.id = ri.decided_by
         LEFT JOIN test_run lr ON lr.id = (SELECT r.id
                                           FROM test_run r
                                                    JOIN package_build pb ON pb.build_id = r.build_id
                                           WHERE r.test_case_id = ri.test_case_id
                                             AND pb.package_id = ri.package_id
                                           ORDER BY r.executed_at DESC, r.created_at DESC, r.code DESC
                                           LIMIT 1)
         LEFT JOIN build lb ON lb.id = lr.build_id
         LEFT JOIN environment le ON le.id = lr.environment_id
WHERE ri.package_id = $1
ORDER BY tc.code;

-- name: ListItemComponents :many
-- Los componentes de cada TC de la selección que tocan los REQ del PKG.
SELECT tcc.test_case_id, c.code, c.name, c.kind
FROM regression_item ri
         JOIN test_case_component tcc ON tcc.test_case_id = ri.test_case_id
         JOIN component c ON c.id = tcc.component_id
WHERE ri.package_id = $1
  AND EXISTS (SELECT 1
              FROM package_requirement pr
                       JOIN requirement_component rc ON rc.requirement_id = pr.requirement_id
              WHERE pr.package_id = ri.package_id
                AND rc.component_id = tcc.component_id)
ORDER BY c.name, c.code;

-- name: UpsertSelection :exec
INSERT INTO regression_selection (package_id, proposed_by, proposed_at, confirmed_by, confirmed_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (package_id) DO UPDATE SET proposed_by  = excluded.proposed_by,
                                       proposed_at  = excluded.proposed_at,
                                       confirmed_by = excluded.confirmed_by,
                                       confirmed_at = excluded.confirmed_at;

-- name: UpsertItem :exec
INSERT INTO regression_item (package_id, test_case_id, origin, proposed, included, reason, decided_by, decided_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (package_id, test_case_id) DO UPDATE SET origin     = excluded.origin,
                                                     proposed   = excluded.proposed,
                                                     included   = excluded.included,
                                                     reason     = excluded.reason,
                                                     decided_by = excluded.decided_by,
                                                     decided_at = excluded.decided_at;

-- name: DeleteItem :exec
DELETE
FROM regression_item
WHERE package_id = $1
  AND test_case_id = $2;

-- name: DeleteItems :exec
DELETE
FROM regression_item
WHERE package_id = $1;

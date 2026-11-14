-- Consultas de solo lectura que cruzan las tablas de otros features para las métricas (reglas §A.7).

-- name: ListCoverage :many
-- La cobertura de cada REQ con la misma cuenta que la regla min_coverage: TC activos que validan algún UC
-- del REQ y, de ellos, los que tienen su último RUN aprobado. El mínimo es el de la regla activa más exigente
-- para su nivel de riesgo y su ruta; un nivel sin porcentaje exige el 100 %, como en la regla. Sin riesgo o
-- sin regla activa, el mínimo es -1.
WITH req AS (SELECT r.id, r.code, r.title, s.key AS system_key, st.key AS state_key, st.name AS state_name,
                    coalesce(ra.level, '') AS risk_level, coalesce(c.route, s.default_route) AS route
             FROM requirement r
                      JOIN system_under_test s ON s.id = r.system_id
                      JOIN workflow_state st ON st.id = r.state_id
                      LEFT JOIN change c ON c.id = r.change_id
                      LEFT JOIN risk_assessment ra ON ra.requirement_id = r.id
             WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
               AND (sqlc.narg(change_code)::text IS NULL OR c.code = sqlc.narg(change_code))
               AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
               AND (sqlc.narg(package_code)::text IS NULL OR EXISTS (SELECT 1
                                                                     FROM package_requirement pr
                                                                              JOIN package p ON p.id = pr.package_id
                                                                     WHERE pr.requirement_id = r.id
                                                                       AND p.code = sqlc.narg(package_code)))),
     tc AS (SELECT DISTINCT req.id AS requirement_id, t.id AS test_case_id
            FROM req
                     JOIN functional_case f ON f.requirement_id = req.id
                     JOIN test_case_functional_case l ON l.functional_case_id = f.id
                     JOIN test_case t ON t.id = l.test_case_id
            WHERE t.retired_at IS NULL),
     counted AS (SELECT tc.requirement_id, count(*)::integer AS required,
                        count(*) FILTER (WHERE (SELECT x.result
                                                FROM test_run x
                                                WHERE x.test_case_id = tc.test_case_id
                                                ORDER BY x.executed_at DESC, x.created_at DESC, x.code DESC
                                                LIMIT 1) = 'passed')::integer AS passed
                 FROM tc
                 GROUP BY tc.requirement_id)
SELECT req.code, req.title, req.system_key, req.state_key, req.state_name, req.risk_level,
       coalesce(counted.required, 0)::integer AS required, coalesce(counted.passed, 0)::integer AS passed,
       coalesce((SELECT max(CASE WHEN jsonb_typeof(b.params -> req.risk_level) = 'number'
                            THEN (b.params ->> req.risk_level)::float8
                        ELSE 100 END)
        FROM blocking_rule b
                 JOIN workflow_transition wt ON wt.id = b.transition_id
                 JOIN workflow_state ws ON ws.id = wt.to_state_id
        WHERE b.rule_key = 'min_coverage'
          AND b.enabled
          AND ws.entity_type = 'requirement'
          AND (b.route IS NULL OR b.route = req.route)
          AND req.risk_level <> ''), -1)::float8 AS minimum_percent
FROM req
         LEFT JOIN counted ON counted.requirement_id = req.id;

-- name: ListDefectStates :many
SELECT key, name, is_final
FROM workflow_state
WHERE entity_type = 'defect'
ORDER BY position NULLS LAST, key;

-- name: ListDefectsForStates :many
SELECT d.id, d.created_at, st.key AS state_key
FROM defect d
         JOIN workflow_state st ON st.id = d.state_id
         JOIN requirement r ON r.id = d.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(severity)::text IS NULL OR d.severity = sqlc.narg(severity))
  AND d.created_at >= sqlc.arg(created_from)
  AND (sqlc.narg(created_to)::timestamptz IS NULL OR d.created_at < sqlc.narg(created_to));

-- name: ListDefectTransitions :many
-- La historia de estados de los DEF, tal como la guardó la auditoría, en orden.
SELECT entity_id, occurred_at, coalesce(old_value, '') AS from_key, coalesce(new_value, '') AS to_key
FROM audit_entry
WHERE entity_type = 'defect'
  AND action = 'transition'
  AND entity_id = ANY (sqlc.arg(defect_ids)::uuid[])
ORDER BY entity_id, seq;

-- name: ListRegressionSizes :many
-- Una fila por PKG con la regresión confirmada, la más reciente primero.
SELECT p.code, p.title, s.key AS system_key, st.key AS state_key, st.name AS state_name,
       rs.confirmed_at::timestamptz AS confirmed_at,
       count(ri.test_case_id) FILTER (WHERE ri.proposed)::integer                        AS proposed,
       count(ri.test_case_id) FILTER (WHERE ri.included)::integer                        AS final,
       count(ri.test_case_id) FILTER (WHERE ri.included AND NOT ri.proposed)::integer    AS added,
       count(ri.test_case_id) FILTER (WHERE ri.proposed AND NOT ri.included)::integer    AS removed,
       count(ri.test_case_id) FILTER (WHERE ri.included AND ri.origin = 'impact')::integer   AS impact,
       count(ri.test_case_id) FILTER (WHERE ri.included AND ri.origin = 'critical')::integer AS critical,
       count(ri.test_case_id) FILTER (WHERE ri.included AND ri.origin = 'manual')::integer   AS manual,
       count(ri.test_case_id) FILTER (WHERE ri.included AND (SELECT r.result
                                                             FROM test_run r
                                                                      JOIN package_build pb ON pb.build_id = r.build_id
                                                             WHERE r.test_case_id = ri.test_case_id
                                                               AND pb.package_id = p.id
                                                             ORDER BY r.executed_at DESC, r.created_at DESC, r.code DESC
                                                             LIMIT 1) = 'passed')::integer AS passed,
       (SELECT count(*)
        FROM test_case t
        WHERE t.system_id = p.system_id
          AND t.retired_at IS NULL)::integer                                              AS system_test_cases
FROM regression_selection rs
         JOIN package p ON p.id = rs.package_id
         JOIN system_under_test s ON s.id = p.system_id
         JOIN workflow_state st ON st.id = p.state_id
         LEFT JOIN regression_item ri ON ri.package_id = rs.package_id
WHERE rs.confirmed_at IS NOT NULL
  AND (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND rs.confirmed_at >= sqlc.arg(confirmed_from)
  AND (sqlc.narg(confirmed_to)::timestamptz IS NULL OR rs.confirmed_at < sqlc.narg(confirmed_to))
GROUP BY p.id, s.key, st.key, st.name, rs.confirmed_at
ORDER BY rs.confirmed_at DESC, p.code DESC;

-- name: ListDefectReleases :many
-- Cada DEF del período con la primera vez que su REQ llegó a producción: el primer despliegue exitoso,
-- en un ambiente de producción, de un PKG que contiene el REQ. El más reciente primero.
SELECT d.code, d.title, d.severity, d.created_at, st.key AS state_key, st.name AS state_name,
       r.code AS requirement_code, coalesce(rel.package_code, '') AS package_code,
       coalesce(to_char(rel.released_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), '')::text AS released_at
FROM defect d
         JOIN workflow_state st ON st.id = d.state_id
         JOIN requirement r ON r.id = d.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
         LEFT JOIN LATERAL (SELECT p.code AS package_code, dep.executed_at AS released_at
                            FROM package_requirement pr
                                     JOIN package p ON p.id = pr.package_id
                                     JOIN deployment dep ON dep.package_id = p.id
                                     JOIN environment e ON e.id = dep.environment_id
                            WHERE pr.requirement_id = d.requirement_id
                              AND dep.result = 'success'
                              AND e.kind = 'production'
                            ORDER BY dep.executed_at, dep.created_at, dep.code
                            LIMIT 1) rel ON true
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(severity)::text IS NULL OR d.severity = sqlc.narg(severity))
  AND d.created_at >= sqlc.arg(created_from)
  AND (sqlc.narg(created_to)::timestamptz IS NULL OR d.created_at < sqlc.narg(created_to))
ORDER BY d.created_at DESC, d.code DESC;

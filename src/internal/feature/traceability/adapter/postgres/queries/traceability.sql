-- Consultas de solo lectura que cruzan las tablas de todos los features (reglas §A.7).

-- name: ListSystems :many
SELECT key, name, default_route
FROM system_under_test
ORDER BY name;

-- name: GetRequirementHeader :one
SELECT r.id, r.code, r.title, r.description, r.acceptance_criteria, r.created_at, r.updated_at,
       s.key AS system_key, s.name AS system_name,
       st.key AS state_key, st.name AS state_name,
       coalesce(c.code, '') AS change_code, coalesce(c.title, '') AS change_title,
       coalesce(c.route, s.default_route) AS route,
       coalesce(o.id, '') AS owner_id, coalesce(o.display_name, '') AS owner_name,
       coalesce(ra.code, '') AS risk_code, coalesce(ra.level, '') AS risk_level,
       coalesce(ra.justification, '') AS risk_justification
FROM requirement r
         JOIN system_under_test s ON s.id = r.system_id
         JOIN workflow_state st ON st.id = r.state_id
         LEFT JOIN change c ON c.id = r.change_id
         LEFT JOIN app_user o ON o.id = r.owner_id
         LEFT JOIN risk_assessment ra ON ra.requirement_id = r.id
WHERE r.code = $1;

-- name: ListLifecycleStates :many
SELECT key, name, position
FROM workflow_state
WHERE entity_type = $1
  AND position IS NOT NULL
ORDER BY position;

-- name: ListTransitions :many
SELECT occurred_at, coalesce(old_value, '') AS from_key, coalesce(new_value, '') AS to_key
FROM audit_entry
WHERE entity_id = $1
  AND action = 'transition'
ORDER BY seq;

-- name: ListRequirementFunctionalCases :many
SELECT code, title
FROM functional_case
WHERE requirement_id = $1
ORDER BY code;

-- name: ListRequirementTestCases :many
SELECT t.code, t.title, t.is_critical, (t.retired_at IS NOT NULL)::boolean AS retired,
       coalesce(lr.code, '') AS run_code, coalesce(lr.result, '') AS run_result,
       coalesce(lr.build_version, '') AS run_build_version, coalesce(lr.environment_name, '') AS run_environment_name,
       coalesce(to_char(lr.executed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), '')::text AS run_executed_at
FROM test_case t
         LEFT JOIN LATERAL (SELECT x.code, x.result, x.build_version, x.environment_name, x.executed_at
                            FROM test_run x
                            WHERE x.test_case_id = t.id
                            ORDER BY x.executed_at DESC, x.created_at DESC, x.code DESC
                            LIMIT 1) lr ON true
WHERE t.id IN (SELECT l.test_case_id
               FROM test_case_functional_case l
                        JOIN functional_case f ON f.id = l.functional_case_id
               WHERE f.requirement_id = $1)
ORDER BY t.code;

-- name: ListRequirementDefects :many
SELECT d.code, d.title, d.severity, st.key AS state_key, st.name AS state_name, st.is_final
FROM defect d
         JOIN workflow_state st ON st.id = d.state_id
WHERE d.requirement_id = $1
ORDER BY d.code;

-- name: ListMatrix :many
-- Una fila por REQ y TC; un REQ sin TC aparece una vez, sin TC.
SELECT r.code AS requirement_code, r.title AS requirement_title, s.key AS system_key,
       st.key AS state_key, st.name AS state_name,
       coalesce(c.code, '') AS change_code, coalesce(ra.code, '') AS risk_code, coalesce(ra.level, '') AS risk_level,
       coalesce(o.display_name, '') AS owner_name,
       coalesce(t.code, '') AS test_case_code, coalesce(t.title, '') AS test_case_title,
       coalesce(t.is_critical, false) AS test_case_critical, (t.retired_at IS NOT NULL)::boolean AS test_case_retired,
       coalesce(lr.code, '') AS run_code, coalesce(lr.result, '') AS run_result,
       coalesce(lr.build_version, '') AS run_build_version, coalesce(lr.environment_name, '') AS run_environment_name,
       coalesce(to_char(lr.executed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), '')::text AS run_executed_at,
       ARRAY(SELECT p.code
             FROM package p
                      JOIN package_requirement pr ON pr.package_id = p.id
             WHERE pr.requirement_id = r.id
             ORDER BY p.code)::text[] AS packages,
       -- El RPT vigente de cada PKG del REQ: el que ninguna versión posterior reemplazó.
       ARRAY(SELECT cr.code
             FROM conformance_report cr
                      JOIN package_requirement pr ON pr.package_id = cr.package_id
             WHERE pr.requirement_id = r.id
               AND NOT EXISTS (SELECT 1 FROM conformance_report n WHERE n.previous_id = cr.id)
             ORDER BY cr.code)::text[] AS reports,
       ARRAY(SELECT f.code
             FROM functional_case f
                      JOIN test_case_functional_case l ON l.functional_case_id = f.id
             WHERE f.requirement_id = r.id
               AND l.test_case_id = t.id
             ORDER BY f.code)::text[] AS functional_cases,
       (SELECT coalesce(jsonb_agg(jsonb_build_object('code', d.code, 'severity', d.severity,
                                                     'stateKey', ds.key, 'stateName', ds.name) ORDER BY d.code), '[]'::jsonb)
        FROM defect d
                 JOIN workflow_state ds ON ds.id = d.state_id
                 JOIN test_run dr ON dr.id = d.test_run_id
        WHERE d.requirement_id = r.id
          AND dr.test_case_id = t.id)::jsonb AS defects
FROM requirement r
         JOIN system_under_test s ON s.id = r.system_id
         JOIN workflow_state st ON st.id = r.state_id
         LEFT JOIN change c ON c.id = r.change_id
         LEFT JOIN risk_assessment ra ON ra.requirement_id = r.id
         LEFT JOIN app_user o ON o.id = r.owner_id
         LEFT JOIN (SELECT DISTINCT f.requirement_id, l.test_case_id
                    FROM functional_case f
                             JOIN test_case_functional_case l ON l.functional_case_id = f.id) rt ON rt.requirement_id = r.id
         LEFT JOIN test_case t ON t.id = rt.test_case_id
         LEFT JOIN LATERAL (SELECT x.code, x.result, x.build_version, x.environment_name, x.executed_at
                            FROM test_run x
                            WHERE x.test_case_id = t.id
                            ORDER BY x.executed_at DESC, x.created_at DESC, x.code DESC
                            LIMIT 1) lr ON true
WHERE (sqlc.narg(change_code)::text IS NULL OR c.code = sqlc.narg(change_code))
  AND (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(risk_level)::text IS NULL OR ra.level = sqlc.narg(risk_level))
  AND (sqlc.narg(owner_id)::text IS NULL OR r.owner_id = sqlc.narg(owner_id))
  AND (sqlc.narg(package_code)::text IS NULL OR EXISTS (SELECT 1
                                                        FROM package_requirement pr
                                                                 JOIN package p ON p.id = pr.package_id
                                                        WHERE pr.requirement_id = r.id
                                                          AND p.code = sqlc.narg(package_code)))
  AND (sqlc.narg(component_code)::text IS NULL OR EXISTS (SELECT 1
                                                          FROM requirement_component rc
                                                                   JOIN component k ON k.id = rc.component_id
                                                          WHERE rc.requirement_id = r.id
                                                            AND k.code = sqlc.narg(component_code)))
  AND (sqlc.narg(created_from)::timestamptz IS NULL OR r.created_at >= sqlc.narg(created_from))
  AND (sqlc.narg(created_to)::timestamptz IS NULL OR r.created_at < sqlc.narg(created_to))
ORDER BY r.created_at DESC, r.code DESC, t.code
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetMatrixIndicators :one
-- Indicadores de todo lo filtrado, no solo de la página.
SELECT count(*)::integer                                                                    AS total_rows,
       count(*) FILTER (WHERE t.id IS NOT NULL AND t.retired_at IS NULL)::integer           AS required_test_cases,
       count(*) FILTER (WHERE t.id IS NOT NULL AND t.retired_at IS NULL
           AND lr.result = 'passed')::integer                                               AS passed_test_cases,
       count(DISTINCT r.id) FILTER (WHERE EXISTS (SELECT 1
                                                  FROM defect d
                                                           JOIN workflow_state ds ON ds.id = d.state_id
                                                  WHERE d.requirement_id = r.id
                                                    AND NOT ds.is_final
                                                    AND d.severity = 'high'))::integer           AS blocked_requirements
FROM requirement r
         JOIN system_under_test s ON s.id = r.system_id
         JOIN workflow_state st ON st.id = r.state_id
         LEFT JOIN change c ON c.id = r.change_id
         LEFT JOIN risk_assessment ra ON ra.requirement_id = r.id
         LEFT JOIN (SELECT DISTINCT f.requirement_id, l.test_case_id
                    FROM functional_case f
                             JOIN test_case_functional_case l ON l.functional_case_id = f.id) rt ON rt.requirement_id = r.id
         LEFT JOIN test_case t ON t.id = rt.test_case_id
         LEFT JOIN LATERAL (SELECT x.result
                            FROM test_run x
                            WHERE x.test_case_id = t.id
                            ORDER BY x.executed_at DESC, x.created_at DESC, x.code DESC
                            LIMIT 1) lr ON true
WHERE (sqlc.narg(change_code)::text IS NULL OR c.code = sqlc.narg(change_code))
  AND (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(risk_level)::text IS NULL OR ra.level = sqlc.narg(risk_level))
  AND (sqlc.narg(owner_id)::text IS NULL OR r.owner_id = sqlc.narg(owner_id))
  AND (sqlc.narg(package_code)::text IS NULL OR EXISTS (SELECT 1
                                                        FROM package_requirement pr
                                                                 JOIN package p ON p.id = pr.package_id
                                                        WHERE pr.requirement_id = r.id
                                                          AND p.code = sqlc.narg(package_code)))
  AND (sqlc.narg(component_code)::text IS NULL OR EXISTS (SELECT 1
                                                          FROM requirement_component rc
                                                                   JOIN component k ON k.id = rc.component_id
                                                          WHERE rc.requirement_id = r.id
                                                            AND k.code = sqlc.narg(component_code)))
  AND (sqlc.narg(created_from)::timestamptz IS NULL OR r.created_at >= sqlc.narg(created_from))
  AND (sqlc.narg(created_to)::timestamptz IS NULL OR r.created_at < sqlc.narg(created_to));

-- name: CountOpenDefectsBySeverity :many
SELECT d.severity, count(*)::integer AS total
FROM defect d
         JOIN workflow_state ds ON ds.id = d.state_id
         JOIN requirement r ON r.id = d.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
         JOIN workflow_state st ON st.id = r.state_id
         LEFT JOIN change c ON c.id = r.change_id
         LEFT JOIN risk_assessment ra ON ra.requirement_id = r.id
WHERE NOT ds.is_final
  AND (sqlc.narg(change_code)::text IS NULL OR c.code = sqlc.narg(change_code))
  AND (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(risk_level)::text IS NULL OR ra.level = sqlc.narg(risk_level))
  AND (sqlc.narg(owner_id)::text IS NULL OR r.owner_id = sqlc.narg(owner_id))
  AND (sqlc.narg(package_code)::text IS NULL OR EXISTS (SELECT 1
                                                        FROM package_requirement pr
                                                                 JOIN package p ON p.id = pr.package_id
                                                        WHERE pr.requirement_id = r.id
                                                          AND p.code = sqlc.narg(package_code)))
  AND (sqlc.narg(component_code)::text IS NULL OR EXISTS (SELECT 1
                                                          FROM requirement_component rc
                                                                   JOIN component k ON k.id = rc.component_id
                                                          WHERE rc.requirement_id = r.id
                                                            AND k.code = sqlc.narg(component_code)))
  AND (sqlc.narg(created_from)::timestamptz IS NULL OR r.created_at >= sqlc.narg(created_from))
  AND (sqlc.narg(created_to)::timestamptz IS NULL OR r.created_at < sqlc.narg(created_to))
GROUP BY d.severity;

-- name: Search :many
-- Búsqueda global por ID o texto. Las coincidencias exactas de ID van primero.
SELECT x.kind, x.code, x.title, x.state_key, x.state_name
FROM (SELECT 'requirement' AS kind, r.code, r.title, st.key AS state_key, st.name AS state_name
      FROM requirement r
               JOIN workflow_state st ON st.id = r.state_id
      UNION ALL
      SELECT 'change', c.code, c.title, st.key, st.name
      FROM change c
               JOIN workflow_state st ON st.id = c.state_id
      UNION ALL
      SELECT 'defect', d.code, d.title, st.key, st.name
      FROM defect d
               JOIN workflow_state st ON st.id = d.state_id
      UNION ALL
      SELECT 'functional_case', code, title, '', ''
      FROM functional_case
      UNION ALL
      SELECT 'test_case', code, title, '', ''
      FROM test_case
      UNION ALL
      SELECT 'test_run', tr.code, t.title, '', ''
      FROM test_run tr
               JOIN test_case t ON t.id = tr.test_case_id
      UNION ALL
      SELECT 'exploratory_session', e.code, r.title, '', ''
      FROM exploratory_session e
               JOIN requirement r ON r.id = e.requirement_id
      UNION ALL
      SELECT 'risk_assessment', ra.code, r.title, '', ''
      FROM risk_assessment ra
               JOIN requirement r ON r.id = ra.requirement_id
      UNION ALL
      SELECT 'document', code, title, '', ''
      FROM document
      UNION ALL
      SELECT 'component', code, name, '', ''
      FROM component
      UNION ALL
      SELECT 'build', code, version, '', ''
      FROM build
      UNION ALL
      SELECT 'package', p.code, p.title, st.key, st.name
      FROM package p
               JOIN workflow_state st ON st.id = p.state_id
      UNION ALL
      SELECT 'environment', code, name, '', ''
      FROM environment
      UNION ALL
      SELECT 'deployment', d.code, e.name, '', ''
      FROM deployment d
               JOIN environment e ON e.id = d.environment_id
      UNION ALL
      SELECT 'checklist', k.code, p.title, '', ''
      FROM checklist k
               JOIN package p ON p.id = k.package_id
      UNION ALL
      SELECT 'diagnostic', g.code, coalesce(p.title, d.code), '', ''
      FROM diagnostic g
               JOIN deployment d ON d.id = g.deployment_id
               LEFT JOIN package p ON p.id = d.package_id
      UNION ALL
      SELECT 'smoke_test', m.code, coalesce(p.title, d.code), '', ''
      FROM smoke_test m
               JOIN deployment d ON d.id = m.deployment_id
               LEFT JOIN package p ON p.id = d.package_id
      UNION ALL
      SELECT 'report', cr.code, p.title, '', ''
      FROM conformance_report cr
               JOIN package p ON p.id = cr.package_id) x
WHERE x.code = upper(sqlc.arg(query)::text)
   OR x.code ILIKE sqlc.arg(pattern)::text || '%'
   OR x.title ILIKE '%' || sqlc.arg(pattern)::text || '%'
ORDER BY (x.code = upper(sqlc.arg(query)::text)) DESC, x.code
LIMIT sqlc.arg(max_results);

-- name: ListPackageTrace :many
-- La trazabilidad de un PKG para su informe: una fila por REQ del PKG y TC; un REQ sin TC aparece una vez.
-- El RUN es el último del TC sobre un BUILD del PKG: lo que se ejecutó sobre lo que se despliega.
SELECT r.code AS requirement_code, r.title AS requirement_title, st.name AS state_name,
       coalesce(c.code, '') AS change_code, coalesce(c.route, s.default_route) AS route,
       coalesce(ra.code, '') AS risk_code, coalesce(ra.level, '') AS risk_level,
       coalesce(t.code, '') AS test_case_code, coalesce(t.is_critical, false) AS test_case_critical,
       coalesce(lr.code, '') AS run_code, coalesce(lr.result, '') AS run_result,
       ARRAY(SELECT f.code
             FROM functional_case f
                      JOIN test_case_functional_case l ON l.functional_case_id = f.id
             WHERE f.requirement_id = r.id
               AND l.test_case_id = t.id
             ORDER BY f.code)::text[] AS functional_cases,
       ARRAY(SELECT d.code
             FROM defect d
                      JOIN test_run dr ON dr.id = d.test_run_id
             WHERE d.requirement_id = r.id
               AND dr.test_case_id = t.id
             ORDER BY d.code)::text[] AS defects
FROM package p
         JOIN package_requirement pr ON pr.package_id = p.id
         JOIN requirement r ON r.id = pr.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
         JOIN workflow_state st ON st.id = r.state_id
         LEFT JOIN change c ON c.id = r.change_id
         LEFT JOIN risk_assessment ra ON ra.requirement_id = r.id
         LEFT JOIN (SELECT DISTINCT f.requirement_id, l.test_case_id
                    FROM functional_case f
                             JOIN test_case_functional_case l ON l.functional_case_id = f.id) rt ON rt.requirement_id = r.id
         LEFT JOIN test_case t ON t.id = rt.test_case_id
         LEFT JOIN LATERAL (SELECT x.code, x.result
                            FROM test_run x
                                     JOIN package_build pb ON pb.build_id = x.build_id
                            WHERE x.test_case_id = t.id
                              AND pb.package_id = p.id
                            ORDER BY x.executed_at DESC, x.created_at DESC, x.code DESC
                            LIMIT 1) lr ON true
WHERE p.code = $1
ORDER BY r.code, t.code;

-- name: ListPackageRuns :many
-- Todos los RUN sobre un BUILD del PKG, en el orden en que se ejecutaron.
SELECT x.code, t.code AS test_case_code, x.result, b.code AS build_code, x.build_version, x.environment_name,
       x.executed_at, u.display_name AS executed_by_name
FROM package p
         JOIN package_build pb ON pb.package_id = p.id
         JOIN build b ON b.id = pb.build_id
         JOIN test_run x ON x.build_id = b.id
         JOIN test_case t ON t.id = x.test_case_id
         JOIN app_user u ON u.id = x.executed_by
WHERE p.code = $1
ORDER BY x.executed_at, x.code;

-- name: ListPackageDefects :many
-- Los DEF no cerrados de los REQ del PKG. Para los diferidos, quién los difirió, cuándo y por qué,
-- leído de la última transición a «Diferido» en la auditoría.
SELECT d.code, d.title, d.severity, st.key AS state_key, st.name AS state_name, r.code AS requirement_code,
       coalesce(dv.actor_name, '') AS deferred_by_name,
       coalesce(to_char(dv.occurred_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), '')::text AS deferred_at, coalesce(dv.reason, '') AS deferred_reason
FROM package p
         JOIN package_requirement pr ON pr.package_id = p.id
         JOIN requirement r ON r.id = pr.requirement_id
         JOIN defect d ON d.requirement_id = r.id
         JOIN workflow_state st ON st.id = d.state_id
         LEFT JOIN LATERAL (SELECT u.display_name AS actor_name, a.occurred_at, a.reason
                            FROM audit_entry a
                                     JOIN app_user u ON u.id = a.actor_id
                            WHERE a.entity_id = d.id
                              AND a.action = 'transition'
                              AND a.new_value = 'deferred'
                            ORDER BY a.seq DESC
                            LIMIT 1) dv ON st.key = 'deferred'
WHERE p.code = $1
  AND NOT st.is_final
ORDER BY CASE d.severity WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END, d.code;

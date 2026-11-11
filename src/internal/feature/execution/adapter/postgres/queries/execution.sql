-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetTestCaseRef :one
SELECT id, code, title, retired_at
FROM test_case
WHERE code = $1;

-- name: GetRequirementRef :one
SELECT id, code
FROM requirement
WHERE code = $1;

-- name: GetUserName :one
SELECT display_name
FROM app_user
WHERE id = $1;

-- name: InsertRun :exec
INSERT INTO test_run (id, code, test_case_id, build_version, build_checksum, environment_name, result, notes,
                      executed_by, executed_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetRunByCode :one
SELECT r.id, r.code, r.test_case_id, t.code AS test_case_code, t.title AS test_case_title, r.build_version,
       r.build_checksum, r.environment_name, r.result, r.notes, r.executed_by, u.display_name AS executed_by_name,
       r.executed_at, r.created_at
FROM test_run r
         JOIN test_case t ON t.id = r.test_case_id
         JOIN app_user u ON u.id = r.executed_by
WHERE r.code = $1;

-- name: ListRuns :many
SELECT r.id, r.code, r.test_case_id, t.code AS test_case_code, t.title AS test_case_title, r.build_version,
       r.build_checksum, r.environment_name, r.result, r.notes, r.executed_by, u.display_name AS executed_by_name,
       r.executed_at, r.created_at
FROM test_run r
         JOIN test_case t ON t.id = r.test_case_id
         JOIN app_user u ON u.id = r.executed_by
WHERE (NOT sqlc.arg(filter_test_cases)::boolean OR r.test_case_id = ANY (sqlc.arg(test_case_ids)::uuid[]))
  AND (sqlc.narg(build_version)::text IS NULL OR r.build_version = sqlc.narg(build_version))
ORDER BY r.executed_at DESC, r.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountRuns :one
SELECT count(*)
FROM test_run r
WHERE (NOT sqlc.arg(filter_test_cases)::boolean OR r.test_case_id = ANY (sqlc.arg(test_case_ids)::uuid[]))
  AND (sqlc.narg(build_version)::text IS NULL OR r.build_version = sqlc.narg(build_version));

-- name: LatestResults :many
-- El último RUN de cada TC: el de ejecución más reciente y, si empatan, el registrado después.
SELECT DISTINCT ON (test_case_id) test_case_id, result
FROM test_run
WHERE test_case_id = ANY (sqlc.arg(test_case_ids)::uuid[])
ORDER BY test_case_id, executed_at DESC, created_at DESC, code DESC;

-- name: InsertExploration :exec
INSERT INTO exploratory_session (id, code, requirement_id, build_version, environment_name, manual_changes,
                                 findings, executed_by, executed_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateExploration :execrows
UPDATE exploratory_session
SET manual_changes = $2,
    findings       = $3,
    updated_at     = $4
WHERE id = $1;

-- name: GetExplorationByCode :one
SELECT e.id, e.code, e.requirement_id, r.code AS requirement_code, e.build_version, e.environment_name,
       e.manual_changes, e.findings, e.executed_by, u.display_name AS executed_by_name, e.executed_at,
       e.created_at, e.updated_at
FROM exploratory_session e
         JOIN requirement r ON r.id = e.requirement_id
         JOIN app_user u ON u.id = e.executed_by
WHERE e.code = $1;

-- name: ListExplorations :many
SELECT e.id, e.code, e.requirement_id, r.code AS requirement_code, e.build_version, e.environment_name,
       e.manual_changes, e.findings, e.executed_by, u.display_name AS executed_by_name, e.executed_at,
       e.created_at, e.updated_at
FROM exploratory_session e
         JOIN requirement r ON r.id = e.requirement_id
         JOIN app_user u ON u.id = e.executed_by
WHERE (sqlc.narg(requirement_code)::text IS NULL OR r.code = sqlc.narg(requirement_code))
ORDER BY e.executed_at DESC, e.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountExplorations :one
SELECT count(*)
FROM exploratory_session e
         JOIN requirement r ON r.id = e.requirement_id
WHERE (sqlc.narg(requirement_code)::text IS NULL OR r.code = sqlc.narg(requirement_code));

-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetRequirementRef :one
SELECT id, code
FROM requirement
WHERE code = $1;

-- name: GetRunRef :one
SELECT id, code, result, test_case_id
FROM test_run
WHERE code = $1;

-- name: GetUserName :one
SELECT display_name
FROM app_user
WHERE id = $1;

-- name: InsertDefect :exec
INSERT INTO defect (id, code, title, description, requirement_id, test_run_id, severity, state_id, assignee_id,
                    created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: UpdateDefect :execrows
UPDATE defect
SET title       = $2,
    description = $3,
    severity    = $4,
    state_id    = $5,
    assignee_id = $6,
    updated_at  = $7
WHERE id = $1;

-- name: GetDefectByCode :one
SELECT d.id, d.code, d.title, d.description, d.requirement_id, r.code AS requirement_code,
       d.test_run_id, t.code AS run_code, d.severity,
       st.id AS state_id, st.key AS state_key, st.name AS state_name, st.is_final AS state_is_final,
       d.assignee_id, a.display_name AS assignee_name, d.created_by, u.display_name AS created_by_name,
       d.created_at, d.updated_at
FROM defect d
         JOIN requirement r ON r.id = d.requirement_id
         JOIN test_run t ON t.id = d.test_run_id
         JOIN workflow_state st ON st.id = d.state_id
         JOIN app_user u ON u.id = d.created_by
         LEFT JOIN app_user a ON a.id = d.assignee_id
WHERE d.code = $1;

-- name: ListDefects :many
SELECT d.id, d.code, d.title, d.description, d.requirement_id, r.code AS requirement_code,
       d.test_run_id, t.code AS run_code, d.severity,
       st.id AS state_id, st.key AS state_key, st.name AS state_name, st.is_final AS state_is_final,
       d.assignee_id, a.display_name AS assignee_name, d.created_by, u.display_name AS created_by_name,
       d.created_at, d.updated_at
FROM defect d
         JOIN requirement r ON r.id = d.requirement_id
         JOIN test_run t ON t.id = d.test_run_id
         JOIN workflow_state st ON st.id = d.state_id
         JOIN app_user u ON u.id = d.created_by
         LEFT JOIN app_user a ON a.id = d.assignee_id
WHERE (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(requirement_code)::text IS NULL OR r.code = sqlc.narg(requirement_code))
  AND (sqlc.narg(severity)::text IS NULL OR d.severity = sqlc.narg(severity))
  AND (sqlc.narg(assignee_id)::text IS NULL OR d.assignee_id = sqlc.narg(assignee_id))
  AND (NOT sqlc.arg(only_open)::boolean OR NOT st.is_final)
  AND (sqlc.narg(query)::text IS NULL OR d.title ILIKE '%' || sqlc.narg(query) || '%' OR d.code = sqlc.narg(query))
ORDER BY d.created_at DESC, d.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountDefects :one
SELECT count(*)
FROM defect d
         JOIN requirement r ON r.id = d.requirement_id
         JOIN workflow_state st ON st.id = d.state_id
WHERE (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(requirement_code)::text IS NULL OR r.code = sqlc.narg(requirement_code))
  AND (sqlc.narg(severity)::text IS NULL OR d.severity = sqlc.narg(severity))
  AND (sqlc.narg(assignee_id)::text IS NULL OR d.assignee_id = sqlc.narg(assignee_id))
  AND (NOT sqlc.arg(only_open)::boolean OR NOT st.is_final)
  AND (sqlc.narg(query)::text IS NULL OR d.title ILIKE '%' || sqlc.narg(query) || '%' OR d.code = sqlc.narg(query));

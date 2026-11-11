-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetSystem :one
SELECT key, name
FROM system_under_test
WHERE key = $1;

-- name: InsertFunctionalCase :exec
INSERT INTO functional_case (id, code, requirement_id, title, description, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateFunctionalCase :execrows
UPDATE functional_case
SET title       = $2,
    description = $3,
    updated_at  = $4
WHERE id = $1;

-- name: GetFunctionalCaseByCode :one
SELECT f.id, f.code, f.requirement_id, r.code AS requirement_code, s.key AS system_key,
       f.title, f.description, f.created_by, f.created_at, f.updated_at
FROM functional_case f
         JOIN requirement r ON r.id = f.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
WHERE f.code = $1;

-- name: ListFunctionalCasesByRequirement :many
SELECT f.id, f.code, f.requirement_id, r.code AS requirement_code, s.key AS system_key,
       f.title, f.description, f.created_by, f.created_at, f.updated_at
FROM functional_case f
         JOIN requirement r ON r.id = f.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
WHERE f.requirement_id = $1
ORDER BY f.code;

-- name: ListTestCaseCodesByFunctionalCases :many
SELECT l.functional_case_id, t.code
FROM test_case_functional_case l
         JOIN test_case t ON t.id = l.test_case_id
WHERE l.functional_case_id = ANY (sqlc.arg(functional_case_ids)::uuid[])
ORDER BY t.code;

-- name: InsertTestCase :exec
INSERT INTO test_case (id, code, system_id, title, preconditions, steps, expected_result, is_critical,
                       retired_at, created_by, created_at, updated_at)
VALUES ($1, $2, (SELECT s.id FROM system_under_test s WHERE s.key = sqlc.arg(system_key)),
        $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateTestCase :execrows
UPDATE test_case
SET title           = $2,
    preconditions   = $3,
    steps           = $4,
    expected_result = $5,
    is_critical     = $6,
    retired_at      = $7,
    updated_at      = $8
WHERE id = $1;

-- name: DeleteTestCaseLinks :exec
DELETE
FROM test_case_functional_case
WHERE test_case_id = $1;

-- name: InsertTestCaseLink :exec
INSERT INTO test_case_functional_case (test_case_id, functional_case_id)
VALUES ($1, $2);

-- name: GetTestCaseByCode :one
SELECT t.id, t.code, s.key AS system_key, s.name AS system_name, t.title, t.preconditions, t.steps,
       t.expected_result, t.is_critical, t.retired_at, t.created_by, t.created_at, t.updated_at
FROM test_case t
         JOIN system_under_test s ON s.id = t.system_id
WHERE t.code = $1;

-- name: ListTestCases :many
SELECT t.id, t.code, s.key AS system_key, s.name AS system_name, t.title, t.preconditions, t.steps,
       t.expected_result, t.is_critical, t.retired_at, t.created_by, t.created_at, t.updated_at
FROM test_case t
         JOIN system_under_test s ON s.id = t.system_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(critical)::boolean IS NULL OR t.is_critical = sqlc.narg(critical))
  AND (sqlc.narg(query)::text IS NULL OR t.title ILIKE '%' || sqlc.narg(query) || '%' OR t.code = sqlc.narg(query))
  AND (sqlc.narg(requirement_id)::uuid IS NULL OR EXISTS (SELECT 1
                                                         FROM test_case_functional_case l
                                                                  JOIN functional_case f ON f.id = l.functional_case_id
                                                         WHERE l.test_case_id = t.id
                                                           AND f.requirement_id = sqlc.narg(requirement_id)))
ORDER BY t.code
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountTestCases :one
SELECT count(*)
FROM test_case t
         JOIN system_under_test s ON s.id = t.system_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(critical)::boolean IS NULL OR t.is_critical = sqlc.narg(critical))
  AND (sqlc.narg(query)::text IS NULL OR t.title ILIKE '%' || sqlc.narg(query) || '%' OR t.code = sqlc.narg(query))
  AND (sqlc.narg(requirement_id)::uuid IS NULL OR EXISTS (SELECT 1
                                                         FROM test_case_functional_case l
                                                                  JOIN functional_case f ON f.id = l.functional_case_id
                                                         WHERE l.test_case_id = t.id
                                                           AND f.requirement_id = sqlc.narg(requirement_id)));

-- name: ListFunctionalCaseRefsByTestCases :many
SELECT l.test_case_id, f.id, f.code, s.key AS system_key
FROM test_case_functional_case l
         JOIN functional_case f ON f.id = l.functional_case_id
         JOIN requirement r ON r.id = f.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
WHERE l.test_case_id = ANY (sqlc.arg(test_case_ids)::uuid[])
ORDER BY f.code;

-- name: ListFunctionalCaseRefsByCodes :many
SELECT f.id, f.code, s.key AS system_key
FROM functional_case f
         JOIN requirement r ON r.id = f.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
WHERE f.code = ANY (sqlc.arg(codes)::text[])
ORDER BY f.code;

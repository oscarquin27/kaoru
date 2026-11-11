-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetSystem :one
SELECT key, name, default_route
FROM system_under_test
WHERE key = $1;

-- name: GetChangeRef :one
SELECT c.id, c.code, c.title, c.route, s.key AS system_key
FROM change c
         JOIN system_under_test s ON s.id = c.system_id
WHERE c.code = $1;

-- name: GetUserName :one
SELECT display_name
FROM app_user
WHERE id = $1;

-- name: InsertRequirement :exec
INSERT INTO requirement (id, code, title, description, acceptance_criteria, system_id, change_id, state_id,
                         owner_id, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, (SELECT s.id FROM system_under_test s WHERE s.key = sqlc.arg(system_key)),
        $6, $7, $8, $9, $10, $11);

-- name: UpdateRequirement :execrows
UPDATE requirement
SET title               = $2,
    description         = $3,
    acceptance_criteria = $4,
    change_id           = $5,
    state_id            = $6,
    owner_id            = $7,
    updated_at          = $8
WHERE id = $1;

-- name: GetRequirementByCode :one
SELECT r.id, r.code, r.title, r.description, r.acceptance_criteria, r.created_at, r.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       st.id AS state_id, st.key AS state_key, st.name AS state_name,
       c.id AS change_id, c.code AS change_code, c.title AS change_title, c.route AS change_route,
       r.owner_id, o.display_name AS owner_name,
       r.created_by, u.display_name AS created_by_name
FROM requirement r
         JOIN system_under_test s ON s.id = r.system_id
         JOIN workflow_state st ON st.id = r.state_id
         JOIN app_user u ON u.id = r.created_by
         LEFT JOIN change c ON c.id = r.change_id
         LEFT JOIN app_user o ON o.id = r.owner_id
WHERE r.code = $1;

-- name: ListRequirements :many
SELECT r.id, r.code, r.title, r.description, r.acceptance_criteria, r.created_at, r.updated_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route,
       st.id AS state_id, st.key AS state_key, st.name AS state_name,
       c.id AS change_id, c.code AS change_code, c.title AS change_title, c.route AS change_route,
       r.owner_id, o.display_name AS owner_name,
       r.created_by, u.display_name AS created_by_name
FROM requirement r
         JOIN system_under_test s ON s.id = r.system_id
         JOIN workflow_state st ON st.id = r.state_id
         JOIN app_user u ON u.id = r.created_by
         LEFT JOIN change c ON c.id = r.change_id
         LEFT JOIN app_user o ON o.id = r.owner_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(change_code)::text IS NULL OR c.code = sqlc.narg(change_code))
  AND (sqlc.narg(owner_id)::text IS NULL OR r.owner_id = sqlc.narg(owner_id))
  AND (sqlc.narg(query)::text IS NULL OR r.title ILIKE '%' || sqlc.narg(query) || '%' OR r.code = sqlc.narg(query))
ORDER BY r.created_at DESC, r.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountRequirements :one
SELECT count(*)
FROM requirement r
         JOIN system_under_test s ON s.id = r.system_id
         JOIN workflow_state st ON st.id = r.state_id
         LEFT JOIN change c ON c.id = r.change_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(state_key)::text IS NULL OR st.key = sqlc.narg(state_key))
  AND (sqlc.narg(change_code)::text IS NULL OR c.code = sqlc.narg(change_code))
  AND (sqlc.narg(owner_id)::text IS NULL OR r.owner_id = sqlc.narg(owner_id))
  AND (sqlc.narg(query)::text IS NULL OR r.title ILIKE '%' || sqlc.narg(query) || '%' OR r.code = sqlc.narg(query));

-- name: InsertLink :exec
INSERT INTO requirement_link (id, requirement_id, kind, label, url, sheet_id, sheet_gid, sheet_range,
                              is_official, snapshot_document_id, added_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12);

-- name: UpdateLink :execrows
UPDATE requirement_link
SET label                = $2,
    is_official          = $3,
    snapshot_document_id = $4,
    updated_at           = $5
WHERE id = $1;

-- name: DeleteLink :execrows
DELETE
FROM requirement_link
WHERE id = $1;

-- name: GetLink :one
SELECT l.id, l.requirement_id, l.kind, l.label, l.url, l.sheet_id, l.sheet_gid, l.sheet_range, l.is_official,
       l.snapshot_document_id, d.code AS snapshot_code, l.added_by, u.display_name AS added_by_name, l.created_at
FROM requirement_link l
         JOIN app_user u ON u.id = l.added_by
         LEFT JOIN document d ON d.id = l.snapshot_document_id
WHERE l.id = $1
  AND l.requirement_id = $2;

-- name: ListLinks :many
SELECT l.id, l.requirement_id, l.kind, l.label, l.url, l.sheet_id, l.sheet_gid, l.sheet_range, l.is_official,
       l.snapshot_document_id, d.code AS snapshot_code, l.added_by, u.display_name AS added_by_name, l.created_at
FROM requirement_link l
         JOIN app_user u ON u.id = l.added_by
         LEFT JOIN document d ON d.id = l.snapshot_document_id
WHERE l.requirement_id = $1
ORDER BY l.created_at, l.id;

-- name: GetDocumentIdByCode :one
SELECT id
FROM document
WHERE code = $1;

-- name: ListEntryContract :many
SELECT i.key, i.label, k.note, k.checked_by, u.display_name AS checked_by_name, k.checked_at
FROM entry_contract_item i
         LEFT JOIN requirement_entry_check k ON k.item_key = i.key AND k.requirement_id = $1
         LEFT JOIN app_user u ON u.id = k.checked_by
WHERE i.enabled
ORDER BY i.position;

-- name: UpsertEntryCheck :exec
INSERT INTO requirement_entry_check (requirement_id, item_key, note, checked_by, checked_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (requirement_id, item_key) DO UPDATE
    SET note       = excluded.note,
        checked_by = excluded.checked_by,
        checked_at = excluded.checked_at;

-- name: DeleteEntryCheck :execrows
DELETE
FROM requirement_entry_check
WHERE requirement_id = $1
  AND item_key = $2;

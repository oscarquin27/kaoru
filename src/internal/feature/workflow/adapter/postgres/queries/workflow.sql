-- name: ListGrantsByRoles :many
SELECT role, permission_key
FROM role_permission
WHERE role = ANY (sqlc.arg(roles)::text[])
ORDER BY role, permission_key;

-- name: ListStates :many
SELECT id, key, name, position, is_initial, is_final
FROM workflow_state
WHERE entity_type = $1
ORDER BY position NULLS LAST, key;

-- name: ListTransitions :many
SELECT t.id, f.key AS from_key, s.key AS to_key, t.permission_key, t.requires_reason
FROM workflow_transition t
         JOIN workflow_state f ON f.id = t.from_state_id
         JOIN workflow_state s ON s.id = t.to_state_id
WHERE f.entity_type = $1
ORDER BY f.position NULLS LAST, f.key, s.position NULLS LAST, s.key;

-- name: ListRules :many
SELECT r.id, r.transition_id, r.rule_key, r.params, r.route, r.blocks, r.enabled
FROM blocking_rule r
         JOIN workflow_transition t ON t.id = r.transition_id
         JOIN workflow_state f ON f.id = t.from_state_id
WHERE f.entity_type = $1
ORDER BY r.rule_key, r.route NULLS FIRST;

-- name: ListPermissions :many
SELECT key, description
FROM permission
ORDER BY key;

-- name: ListAllGrants :many
SELECT role, permission_key
FROM role_permission
ORDER BY role, permission_key;

-- name: DeleteRoleGrants :exec
DELETE
FROM role_permission
WHERE role = $1;

-- name: InsertGrant :exec
INSERT INTO role_permission (role, permission_key)
VALUES ($1, $2);

-- name: GetRule :one
SELECT r.id, r.rule_key, r.params, r.route, r.blocks, r.enabled
FROM blocking_rule r
WHERE r.id = $1;

-- name: UpdateRule :execrows
UPDATE blocking_rule
SET params     = $2,
    blocks     = $3,
    enabled    = $4,
    updated_at = $5
WHERE id = $1;

-- name: InsertState :exec
INSERT INTO workflow_state (id, entity_type, key, name, position, is_final, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $6);

-- name: UpdateState :execrows
UPDATE workflow_state
SET name       = $3,
    position   = $4,
    updated_at = $5
WHERE entity_type = $1
  AND key = $2;

-- name: DeleteState :execrows
DELETE
FROM workflow_state
WHERE entity_type = $1
  AND key = $2;

-- name: UpsertTransition :execrows
INSERT INTO workflow_transition (id, from_state_id, to_state_id, permission_key, requires_reason, created_at, updated_at)
SELECT gen_random_uuid(), f.id, s.id, sqlc.arg(permission_key), sqlc.arg(requires_reason), sqlc.arg(now), sqlc.arg(now)
FROM workflow_state f
         JOIN workflow_state s ON s.entity_type = f.entity_type
WHERE f.entity_type = sqlc.arg(entity_type)
  AND f.key = sqlc.arg(from_key)
  AND s.key = sqlc.arg(to_key)
ON CONFLICT (from_state_id, to_state_id) DO UPDATE
    SET permission_key  = excluded.permission_key,
        requires_reason = excluded.requires_reason,
        updated_at      = excluded.updated_at;

-- name: DeleteTransition :execrows
DELETE
FROM workflow_transition t
    USING workflow_state f, workflow_state s
WHERE t.from_state_id = f.id
  AND t.to_state_id = s.id
  AND f.entity_type = sqlc.arg(entity_type)
  AND f.key = sqlc.arg(from_key)
  AND s.key = sqlc.arg(to_key);

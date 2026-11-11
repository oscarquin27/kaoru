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

-- name: LockChain :exec
-- Serializa las inserciones para que dos escrituras no lean el mismo hash anterior.
SELECT pg_advisory_xact_lock(hashtext('kaoru.audit_entry'));

-- name: LastHash :one
SELECT hash
FROM audit_entry
ORDER BY seq DESC
LIMIT 1;

-- name: InsertEntry :exec
INSERT INTO audit_entry (id, occurred_at, actor_id, entity_type, entity_id, entity_code, action,
                         field, old_value, new_value, reason, prev_hash, hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: ListEntries :many
SELECT a.id, a.seq, a.occurred_at, a.actor_id, u.display_name AS actor_name, a.entity_type, a.entity_id,
       a.entity_code, a.action, a.field, a.old_value, a.new_value, a.reason, a.prev_hash, a.hash
FROM audit_entry a
         JOIN app_user u ON u.id = a.actor_id
WHERE (sqlc.narg(entity_code)::text IS NULL OR a.entity_code = sqlc.narg(entity_code))
  AND (sqlc.narg(actor_id)::text IS NULL OR a.actor_id = sqlc.narg(actor_id))
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR a.occurred_at >= sqlc.narg(from_time))
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR a.occurred_at < sqlc.narg(to_time))
ORDER BY a.seq DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountEntries :one
SELECT count(*)
FROM audit_entry a
WHERE (sqlc.narg(entity_code)::text IS NULL OR a.entity_code = sqlc.narg(entity_code))
  AND (sqlc.narg(actor_id)::text IS NULL OR a.actor_id = sqlc.narg(actor_id))
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR a.occurred_at >= sqlc.narg(from_time))
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR a.occurred_at < sqlc.narg(to_time));

-- name: ListChainAfter :many
SELECT id, seq, occurred_at, actor_id, entity_type, entity_id, entity_code, action,
       field, old_value, new_value, reason, prev_hash, hash
FROM audit_entry
WHERE seq > sqlc.arg(after_seq)
ORDER BY seq
LIMIT sqlc.arg(batch_size);

-- name: LastTransitions :many
-- Cuándo entró cada entidad a su estado actual: su última transición.
SELECT entity_id, max(occurred_at)::timestamptz AS occurred_at
FROM audit_entry
WHERE action = 'transition'
  AND entity_id = ANY (sqlc.arg(entity_ids)::uuid[])
GROUP BY entity_id;

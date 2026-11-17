-- name: InsertSystem :exec
INSERT INTO system_under_test (id, key, name, default_route, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateSystem :execrows
UPDATE system_under_test
SET name          = $2,
    default_route = $3,
    updated_at    = $4
WHERE id = $1;

-- name: LockSystem :one
-- Bloquea la fila hasta el fin de la transacción: dos ediciones del mismo sistema van una tras otra.
SELECT id, key, name, default_route, created_at, updated_at
FROM system_under_test
WHERE key = $1
FOR UPDATE;

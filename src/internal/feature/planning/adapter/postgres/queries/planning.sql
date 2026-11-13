-- name: InsertMilestone :exec
INSERT INTO milestone (id, title, due_date, notes, owner_id, completed_at, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: UpdateMilestone :exec
UPDATE milestone
SET title = $2, due_date = $3, notes = $4, owner_id = $5, completed_at = $6, updated_at = $7
WHERE id = $1;

-- name: LockMilestone :one
-- Bloquea la fila hasta el fin de la transacción: dos ediciones del mismo hito van una tras otra.
SELECT id
FROM milestone
WHERE id = $1
    FOR UPDATE;

-- name: GetMilestone :one
SELECT m.id, m.title, m.due_date, m.notes, m.owner_id, o.display_name AS owner_name, m.completed_at,
       m.created_by, c.display_name AS created_by_name, m.created_at, m.updated_at
FROM milestone m
         JOIN app_user c ON c.id = m.created_by
         LEFT JOIN app_user o ON o.id = m.owner_id
WHERE m.id = $1;

-- name: ListMilestones :many
-- Por fecha, la más próxima primero. status es pending, completed o nulo para todos.
SELECT m.id, m.title, m.due_date, m.notes, m.owner_id, o.display_name AS owner_name, m.completed_at,
       m.created_by, c.display_name AS created_by_name, m.created_at, m.updated_at
FROM milestone m
         JOIN app_user c ON c.id = m.created_by
         LEFT JOIN app_user o ON o.id = m.owner_id
WHERE (sqlc.narg(status)::text IS NULL
    OR (sqlc.narg(status) = 'pending' AND m.completed_at IS NULL)
    OR (sqlc.narg(status) = 'completed' AND m.completed_at IS NOT NULL))
ORDER BY m.due_date, m.created_at, m.id
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountMilestones :one
SELECT count(*)
FROM milestone m
WHERE (sqlc.narg(status)::text IS NULL
    OR (sqlc.narg(status) = 'pending' AND m.completed_at IS NULL)
    OR (sqlc.narg(status) = 'completed' AND m.completed_at IS NOT NULL));

-- name: ListPendingMilestones :many
SELECT m.id, m.title, m.due_date, m.notes, m.owner_id, o.display_name AS owner_name, m.completed_at,
       m.created_by, c.display_name AS created_by_name, m.created_at, m.updated_at
FROM milestone m
         JOIN app_user c ON c.id = m.created_by
         LEFT JOIN app_user o ON o.id = m.owner_id
WHERE m.completed_at IS NULL
ORDER BY m.due_date, m.created_at, m.id;

-- name: GetUser :one
-- Lee el responsable del hito: es el sentido de la llave foránea milestone.owner_id.
SELECT id, display_name
FROM app_user
WHERE id = $1;

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

-- name: ListMilestonesBetween :many
SELECT m.id, m.title, m.due_date, m.notes, m.owner_id, o.display_name AS owner_name, m.completed_at,
       m.created_by, c.display_name AS created_by_name, m.created_at, m.updated_at
FROM milestone m
         JOIN app_user c ON c.id = m.created_by
         LEFT JOIN app_user o ON o.id = m.owner_id
WHERE m.due_date BETWEEN sqlc.arg(from_date)::date AND sqlc.arg(to_date)::date
ORDER BY m.due_date, m.created_at, m.id;

-- Actividades (fase 6). Los operadores llegan ordenados por nombre en dos arreglos paralelos.

-- name: NextActivityCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ('ACT', $1, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: InsertActivity :exec
INSERT INTO activity (id, code, kind, title, change_id, start_date, end_date, notes, cancelled_at, created_by,
                      created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: UpdateActivity :exec
UPDATE activity
SET kind = $2, title = $3, change_id = $4, start_date = $5, end_date = $6, notes = $7, cancelled_at = $8,
    updated_at = $9
WHERE id = $1;

-- name: DeleteActivityOperators :exec
DELETE FROM activity_operator
WHERE activity_id = $1;

-- name: InsertActivityOperators :exec
INSERT INTO activity_operator (activity_id, user_id)
SELECT sqlc.arg(activity_id)::uuid, unnest(sqlc.arg(user_ids)::text[]);

-- name: LockActivity :one
-- Bloquea la fila hasta el fin de la transacción: dos ediciones de la misma actividad van una tras otra.
SELECT id
FROM activity
WHERE code = $1
    FOR UPDATE;

-- name: GetActivity :one
SELECT a.id, a.code, a.kind, a.title, a.change_id, c.code AS change_code, c.title AS change_title,
       s.name AS system_name, a.start_date, a.end_date, a.notes, a.cancelled_at, a.created_by,
       cb.display_name AS created_by_name, a.created_at, a.updated_at,
       ops.ids::text[] AS operator_ids, ops.names::text[] AS operator_names
FROM activity a
         JOIN app_user cb ON cb.id = a.created_by
         LEFT JOIN change c ON c.id = a.change_id
         LEFT JOIN system_under_test s ON s.id = c.system_id
         CROSS JOIN LATERAL (SELECT coalesce(array_agg(u.id ORDER BY u.display_name, u.id), '{}')           AS ids,
                                    coalesce(array_agg(u.display_name ORDER BY u.display_name, u.id), '{}') AS names
                             FROM activity_operator ao
                                      JOIN app_user u ON u.id = ao.user_id
                             WHERE ao.activity_id = a.id) ops
WHERE a.code = $1;

-- name: ListActivities :many
-- status: planned, active, done o cancelled según today; nulo para todas.
SELECT a.id, a.code, a.kind, a.title, a.change_id, c.code AS change_code, c.title AS change_title,
       s.name AS system_name, a.start_date, a.end_date, a.notes, a.cancelled_at, a.created_by,
       cb.display_name AS created_by_name, a.created_at, a.updated_at,
       ops.ids::text[] AS operator_ids, ops.names::text[] AS operator_names
FROM activity a
         JOIN app_user cb ON cb.id = a.created_by
         LEFT JOIN change c ON c.id = a.change_id
         LEFT JOIN system_under_test s ON s.id = c.system_id
         CROSS JOIN LATERAL (SELECT coalesce(array_agg(u.id ORDER BY u.display_name, u.id), '{}')           AS ids,
                                    coalesce(array_agg(u.display_name ORDER BY u.display_name, u.id), '{}') AS names
                             FROM activity_operator ao
                                      JOIN app_user u ON u.id = ao.user_id
                             WHERE ao.activity_id = a.id) ops
WHERE (sqlc.narg(operator)::text IS NULL
    OR EXISTS (SELECT 1 FROM activity_operator x WHERE x.activity_id = a.id AND x.user_id = sqlc.narg(operator)))
  AND (sqlc.narg(change_code)::text IS NULL OR c.code = sqlc.narg(change_code))
  AND (sqlc.narg(kind)::text IS NULL OR a.kind = sqlc.narg(kind))
  AND (sqlc.narg(status)::text IS NULL
    OR (sqlc.narg(status) = 'cancelled' AND a.cancelled_at IS NOT NULL)
    OR (a.cancelled_at IS NULL AND (
        (sqlc.narg(status) = 'planned' AND a.start_date > sqlc.arg(today)::date)
            OR (sqlc.narg(status) = 'active' AND a.start_date <= sqlc.arg(today)::date AND a.end_date >= sqlc.arg(today)::date)
            OR (sqlc.narg(status) = 'done' AND a.end_date < sqlc.arg(today)::date))))
  AND (sqlc.narg(from_date)::date IS NULL OR a.end_date >= sqlc.narg(from_date)::date)
  AND (sqlc.narg(to_date)::date IS NULL OR a.start_date <= sqlc.narg(to_date)::date)
ORDER BY a.start_date, a.end_date, a.code
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountActivities :one
SELECT count(*)
FROM activity a
         LEFT JOIN change c ON c.id = a.change_id
WHERE (sqlc.narg(operator)::text IS NULL
    OR EXISTS (SELECT 1 FROM activity_operator x WHERE x.activity_id = a.id AND x.user_id = sqlc.narg(operator)))
  AND (sqlc.narg(change_code)::text IS NULL OR c.code = sqlc.narg(change_code))
  AND (sqlc.narg(kind)::text IS NULL OR a.kind = sqlc.narg(kind))
  AND (sqlc.narg(status)::text IS NULL
    OR (sqlc.narg(status) = 'cancelled' AND a.cancelled_at IS NOT NULL)
    OR (a.cancelled_at IS NULL AND (
        (sqlc.narg(status) = 'planned' AND a.start_date > sqlc.arg(today)::date)
            OR (sqlc.narg(status) = 'active' AND a.start_date <= sqlc.arg(today)::date AND a.end_date >= sqlc.arg(today)::date)
            OR (sqlc.narg(status) = 'done' AND a.end_date < sqlc.arg(today)::date))))
  AND (sqlc.narg(from_date)::date IS NULL OR a.end_date >= sqlc.narg(from_date)::date)
  AND (sqlc.narg(to_date)::date IS NULL OR a.start_date <= sqlc.narg(to_date)::date);

-- name: ListActivitiesBetween :many
-- Las no canceladas que tocan los días; con operators, solo las que tienen alguno de ellos.
SELECT a.id, a.code, a.kind, a.title, a.change_id, c.code AS change_code, c.title AS change_title,
       s.name AS system_name, a.start_date, a.end_date, a.notes, a.cancelled_at, a.created_by,
       cb.display_name AS created_by_name, a.created_at, a.updated_at,
       ops.ids::text[] AS operator_ids, ops.names::text[] AS operator_names
FROM activity a
         JOIN app_user cb ON cb.id = a.created_by
         LEFT JOIN change c ON c.id = a.change_id
         LEFT JOIN system_under_test s ON s.id = c.system_id
         CROSS JOIN LATERAL (SELECT coalesce(array_agg(u.id ORDER BY u.display_name, u.id), '{}')           AS ids,
                                    coalesce(array_agg(u.display_name ORDER BY u.display_name, u.id), '{}') AS names
                             FROM activity_operator ao
                                      JOIN app_user u ON u.id = ao.user_id
                             WHERE ao.activity_id = a.id) ops
WHERE a.cancelled_at IS NULL
  AND a.end_date >= sqlc.arg(from_date)::date
  AND a.start_date <= sqlc.arg(to_date)::date
  AND (sqlc.narg(operators)::text[] IS NULL
    OR EXISTS (SELECT 1
               FROM activity_operator x
               WHERE x.activity_id = a.id AND x.user_id = ANY (sqlc.narg(operators)::text[])))
ORDER BY a.start_date, a.end_date, a.code;

-- name: ListCertifications :many
SELECT a.id, a.code, a.kind, a.title, a.change_id, c.code AS change_code, c.title AS change_title,
       s.name AS system_name, a.start_date, a.end_date, a.notes, a.cancelled_at, a.created_by,
       cb.display_name AS created_by_name, a.created_at, a.updated_at,
       ops.ids::text[] AS operator_ids, ops.names::text[] AS operator_names
FROM activity a
         JOIN app_user cb ON cb.id = a.created_by
         LEFT JOIN change c ON c.id = a.change_id
         LEFT JOIN system_under_test s ON s.id = c.system_id
         CROSS JOIN LATERAL (SELECT coalesce(array_agg(u.id ORDER BY u.display_name, u.id), '{}')           AS ids,
                                    coalesce(array_agg(u.display_name ORDER BY u.display_name, u.id), '{}') AS names
                             FROM activity_operator ao
                                      JOIN app_user u ON u.id = ao.user_id
                             WHERE ao.activity_id = a.id) ops
WHERE a.kind = 'certification'
  AND c.code = $1
ORDER BY a.start_date, a.end_date, a.code;

-- name: GetChange :one
-- El CHG de una actividad: es el sentido de la llave foránea activity.change_id.
SELECT c.id, c.code, c.title, s.name AS system_name
FROM change c
         JOIN system_under_test s ON s.id = c.system_id
WHERE c.code = $1;

-- name: GetUsers :many
SELECT id, display_name
FROM app_user
WHERE id = ANY (sqlc.arg(ids)::text[]);

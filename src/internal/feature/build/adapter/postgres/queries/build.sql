-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetSystem :one
SELECT id, key, name, default_route
FROM system_under_test
WHERE key = $1;

-- name: InsertBuild :exec
INSERT INTO build (id, code, system_id, version, checksum, migrations, dependencies, notes, built_by, created_at)
VALUES ($1, $2, (SELECT s.id FROM system_under_test s WHERE s.key = sqlc.arg(system_key)), $3, $4, $5, $6, $7, $8, $9);

-- name: InsertBuildRequirement :exec
INSERT INTO build_requirement (build_id, requirement_id)
VALUES ($1, $2);

-- name: GetBuildByCode :one
SELECT b.id, b.code, b.version, b.checksum, b.migrations, b.dependencies, b.notes,
       b.built_by, u.display_name AS built_by_name, b.created_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route
FROM build b
         JOIN system_under_test s ON s.id = b.system_id
         JOIN app_user u ON u.id = b.built_by
WHERE b.code = $1;

-- name: GetBuildCodeByChecksum :one
SELECT code
FROM build
WHERE checksum = $1;

-- name: ListBuilds :many
SELECT b.id, b.code, b.version, b.checksum, b.migrations, b.dependencies, b.notes,
       b.built_by, u.display_name AS built_by_name, b.created_at,
       s.key AS system_key, s.name AS system_name, s.default_route AS system_default_route
FROM build b
         JOIN system_under_test s ON s.id = b.system_id
         JOIN app_user u ON u.id = b.built_by
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(query)::text IS NULL OR b.version ILIKE '%' || sqlc.narg(query) || '%')
  AND (sqlc.narg(requirement_code)::text IS NULL OR EXISTS (SELECT 1
                                                            FROM build_requirement br
                                                                     JOIN requirement r ON r.id = br.requirement_id
                                                            WHERE br.build_id = b.id
                                                              AND r.code = sqlc.narg(requirement_code)))
ORDER BY b.created_at DESC, b.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountBuilds :one
SELECT count(*)
FROM build b
         JOIN system_under_test s ON s.id = b.system_id
WHERE (sqlc.narg(system_key)::text IS NULL OR s.key = sqlc.narg(system_key))
  AND (sqlc.narg(query)::text IS NULL OR b.version ILIKE '%' || sqlc.narg(query) || '%')
  AND (sqlc.narg(requirement_code)::text IS NULL OR EXISTS (SELECT 1
                                                            FROM build_requirement br
                                                                     JOIN requirement r ON r.id = br.requirement_id
                                                            WHERE br.build_id = b.id
                                                              AND r.code = sqlc.narg(requirement_code)));

-- name: ListBuildRequirements :many
-- Los REQ del manifiesto de varios BUILD, en orden de código.
SELECT br.build_id, r.id, r.code, r.title, s.key AS system_key
FROM build_requirement br
         JOIN requirement r ON r.id = br.requirement_id
         JOIN system_under_test s ON s.id = r.system_id
WHERE br.build_id = ANY (sqlc.arg(build_ids)::uuid[])
ORDER BY r.code;

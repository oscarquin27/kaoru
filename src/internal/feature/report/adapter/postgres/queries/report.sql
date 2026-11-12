-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetPackageRef :one
-- Lee el PKG del informe: es el sentido de la llave foránea conformance_report.package_id.
SELECT id, code, title
FROM package
WHERE code = $1;

-- name: LockPackage :one
-- Bloquea el PKG hasta el fin de la transacción: un RPT es de solo inserción, así que lo que
-- protege la cadena de versiones es la fila del PKG.
SELECT id
FROM package
WHERE id = $1
    FOR UPDATE;

-- name: GetUserName :one
SELECT display_name
FROM app_user
WHERE id = $1;

-- name: InsertReport :exec
INSERT INTO conformance_report (id, code, package_id, version, previous_id, content, checksum, issued_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetLatestReport :one
SELECT r.id, r.code, r.version, r.previous_id, prev.code AS previous_code, ''::text AS superseded_by,
       r.content, r.checksum, r.issued_by, u.display_name AS issued_by_name, r.created_at,
       p.id AS package_id, p.code AS package_code, p.title AS package_title
FROM conformance_report r
         JOIN package p ON p.id = r.package_id
         JOIN app_user u ON u.id = r.issued_by
         LEFT JOIN conformance_report prev ON prev.id = r.previous_id
WHERE r.package_id = $1
ORDER BY r.version DESC
LIMIT 1;

-- name: GetReportByCode :one
SELECT r.id, r.code, r.version, r.previous_id, prev.code AS previous_code,
       coalesce(next.code, '')::text AS superseded_by,
       r.content, r.checksum, r.issued_by, u.display_name AS issued_by_name, r.created_at,
       p.id AS package_id, p.code AS package_code, p.title AS package_title
FROM conformance_report r
         JOIN package p ON p.id = r.package_id
         JOIN app_user u ON u.id = r.issued_by
         LEFT JOIN conformance_report prev ON prev.id = r.previous_id
         LEFT JOIN conformance_report next ON next.previous_id = r.id
WHERE r.code = $1;

-- name: ListReports :many
-- Sin el contenido: la lista solo muestra la cadena de versiones y sus huellas.
SELECT r.id, r.code, r.version, r.previous_id, prev.code AS previous_code,
       coalesce(next.code, '')::text AS superseded_by,
       r.checksum, r.issued_by, u.display_name AS issued_by_name, r.created_at,
       p.id AS package_id, p.code AS package_code, p.title AS package_title
FROM conformance_report r
         JOIN package p ON p.id = r.package_id
         JOIN app_user u ON u.id = r.issued_by
         LEFT JOIN conformance_report prev ON prev.id = r.previous_id
         LEFT JOIN conformance_report next ON next.previous_id = r.id
WHERE (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code))
ORDER BY r.created_at DESC, r.code DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountReports :one
SELECT count(*)
FROM conformance_report r
         JOIN package p ON p.id = r.package_id
WHERE (sqlc.narg(package_code)::text IS NULL OR p.code = sqlc.narg(package_code));

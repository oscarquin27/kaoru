-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: InsertDocument :exec
INSERT INTO document (id, code, title, kind, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6);

-- name: LockDocument :one
-- Bloquea el documento mientras se le agrega una versión, para que dos subidas no tomen el mismo número.
SELECT id
FROM document
WHERE code = $1
    FOR UPDATE;

-- name: TouchDocument :exec
UPDATE document
SET updated_at = $2
WHERE id = $1;

-- name: NextVersion :one
SELECT (coalesce(max(version), 0) + 1)::integer
FROM document_version
WHERE document_id = $1;

-- name: InsertVersion :exec
INSERT INTO document_version (id, document_id, version, file_name, media_type, size_bytes, checksum, uploaded_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: InsertLink :execrows
INSERT INTO document_link (document_id, entity_type, entity_id, entity_code, linked_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING;

-- name: GetDocumentByCode :one
SELECT id, code, title, kind, created_by, created_at, updated_at
FROM document
WHERE code = $1;

-- name: ListDocumentsByEntity :many
SELECT d.id, d.code, d.title, d.kind, d.created_by, d.created_at, d.updated_at
FROM document d
         JOIN document_link l ON l.document_id = d.id
WHERE l.entity_type = $1
  AND l.entity_id = $2
ORDER BY d.code;

-- name: ListVersions :many
SELECT v.document_id, v.version, v.file_name, v.media_type, v.size_bytes, v.checksum, v.uploaded_by,
       u.display_name AS uploaded_by_name, v.created_at
FROM document_version v
         JOIN app_user u ON u.id = v.uploaded_by
WHERE v.document_id = ANY (sqlc.arg(document_ids)::uuid[])
ORDER BY v.document_id, v.version DESC;

-- name: ListLinks :many
SELECT document_id, entity_type, entity_id, entity_code
FROM document_link
WHERE document_id = ANY (sqlc.arg(document_ids)::uuid[])
ORDER BY document_id, entity_code;

-- name: ReferencedChecksums :many
SELECT DISTINCT checksum
FROM document_version
WHERE checksum = ANY (sqlc.arg(checksums)::text[]);

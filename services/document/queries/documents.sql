-- name: CreateDocument :one
INSERT INTO document.documents (id, customer_id, document_type, object_key, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, customer_id, document_type, object_key, content_type, size_bytes, sha256, status, created_at, confirmed_at;

-- name: GetDocumentByID :one
SELECT id, customer_id, document_type, object_key, content_type, size_bytes, sha256, status, created_at, confirmed_at
FROM document.documents
WHERE id = $1;

-- name: ConfirmDocument :one
UPDATE document.documents
SET status = 'CONFIRMED', content_type = $2, size_bytes = $3, sha256 = $4, confirmed_at = now()
WHERE id = $1 AND status = 'PENDING'
RETURNING id, customer_id, document_type, object_key, content_type, size_bytes, sha256, status, created_at, confirmed_at;

-- name: RejectDocument :one
UPDATE document.documents
SET status = 'REJECTED', content_type = $2, size_bytes = $3
WHERE id = $1 AND status = 'PENDING'
RETURNING id, customer_id, document_type, object_key, content_type, size_bytes, sha256, status, created_at, confirmed_at;

-- name: ListDocuments :many
SELECT id, customer_id, document_type, object_key, content_type, size_bytes, sha256, status, created_at, confirmed_at
FROM document.documents
WHERE customer_id = @customer_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('cursor')::uuid IS NULL OR id < sqlc.narg('cursor')::uuid)
ORDER BY id DESC  -- UUID v7 монотонны по времени
LIMIT @lim;

-- name: ListConfirmedDocumentTypes :many
SELECT DISTINCT document_type
FROM document.documents
WHERE customer_id = $1 AND status = 'CONFIRMED';

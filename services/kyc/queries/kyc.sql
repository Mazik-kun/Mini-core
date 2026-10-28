-- name: CreateApplication :one
INSERT INTO kyc.applications (
    id, customer_id, status, full_name_snapshot, birth_date_snapshot
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetApplicationByID :one
SELECT * FROM kyc.applications WHERE id = $1;

-- name: GetActiveByCustomerID :one
SELECT * FROM kyc.applications 
WHERE customer_id = $1 
  AND status NOT IN ('APPROVED', 'REJECTED', 'AUTO_REJECTED')
LIMIT 1;

-- name: ClaimApplication :one
UPDATE kyc.applications
SET officer_id = $2, updated_at = now()
WHERE id = $1 AND status = 'IN_REVIEW' AND officer_id IS NULL
RETURNING *;

-- name: UpdateApplicationStatus :one
UPDATE kyc.applications
SET status = $2, 
    reason = $3, 
    updated_at = now(),
    decided_at = CASE WHEN $2 IN ('APPROVED', 'REJECTED', 'AUTO_REJECTED') THEN now() ELSE decided_at END
WHERE id = $1 AND status = $4
RETURNING *;

-- name: ListApplications :many
SELECT * FROM kyc.applications
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('cursor')::uuid IS NULL OR id < sqlc.narg('cursor')::uuid)
ORDER BY id DESC
LIMIT $1;

-- name: ListForAutoChecks :many
SELECT * FROM kyc.applications
WHERE status = 'AUTO_CHECKS'
ORDER BY updated_at ASC
FOR UPDATE SKIP LOCKED
LIMIT $1;

-- name: IncrementAttempts :one
UPDATE kyc.applications
SET attempts = attempts + 1, updated_at = now()
WHERE id = $1
RETURNING attempts;

-- name: InsertEvent :one
INSERT INTO kyc.kyc_events (
    id, application_id, actor_id, actor_role, from_status, to_status, comment
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: SaveCheck :one
INSERT INTO kyc.kyc_checks (
    id, application_id, check_type, status, details
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ListSanctions :many
SELECT * FROM kyc.sanctions_list;
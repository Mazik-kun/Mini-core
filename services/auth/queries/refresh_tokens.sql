-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(id, user_id, token_hash, expires_at, revoked_at) VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at;

-- name: GetByHash :one
SELECT id, user_id,token_hash ,expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE token_hash = $1;

-- name: Revoke :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1;

-- name: RevokeAllForUser :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE user_id = $1 and revoked_at IS NULL;
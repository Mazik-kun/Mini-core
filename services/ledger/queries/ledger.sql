-- name: CreateTransaction :one
INSERT INTO transactions (id, status, type, idempotency_key, created_at)
VALUES ($1, $2, $3, $4, now())
RETURNING *;

-- name: GetTransactionByIdempotencyKey :one
SELECT * FROM transactions WHERE idempotency_key = $1;

-- name: CreateEntry :one
INSERT INTO entries (id, transaction_id, account_id, direction, amount, currency, created_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
RETURNING *;

-- name: GetBalancesForUpdate :many
SELECT * FROM balances 
WHERE account_id IN ($1, $2) 
ORDER BY account_id ASC 
FOR UPDATE;

-- name: UpsertBalance :exec
INSERT INTO balances (account_id, amount, currency, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (account_id) DO UPDATE 
SET amount = balances.amount + EXCLUDED.amount, updated_at = now();

-- name: GetBalance :one
SELECT * FROM balances WHERE account_id = $1;

-- name: CreateOutboxEvent :one
INSERT INTO outbox (id, event_type, payload, status, created_at)
VALUES ($1, $2, $3, 'PENDING', now())
RETURNING *;

-- name: GetEntriesByTransactionID :many
SELECT * FROM entries WHERE transaction_id = $1;

-- name: GetTransactionByAccountID :many
SElECT DISTINCT t.* FROM transactions t
JOIN entries e ON t.id = e.transaction_id
WHERE e.account_id = $1 AND (sqlc.narg('from_time')::timestampz IS NULL OR t.created_at >= sqlc.narg('from_time')) AND (sqlc.narg('to_time')::timestampz IS NULL OR t.created_at >= sqlc.narg('to_time'))
ORDER BY t.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPendingOutboxEvents :many
SELECT * FROM outbox
WHERE status = 'PENDING'
ORDER BY created_at ASC
LIMIT 10;

-- name: MarkOutboxEventAsSent :exec
UPDATE outbox SET status = 'SENT' WHERE id = $1;
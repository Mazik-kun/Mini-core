-- name: CreateAccount :one
INSERT INTO accounts (id, customer_id, account_number, currency, account_type, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now(), now())
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: GetAccountByNumber :one
SELECT * FROM accounts WHERE account_number = $1;

-- name: ListAccountsByCustomer :many
SELECT * FROM accounts WHERE customer_id = $1 ORDER BY created_at DESC;

-- name: CountAccountsByCurrency :one
SELECT COUNT(*) FROM accounts WHERE customer_id = $1 AND currency = $2;

-- name: UpdateAccountStatus :one
UPDATE accounts SET status = $2, updated_at = now() WHERE id = $1 RETURNING *;
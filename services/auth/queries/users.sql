-- name: CreateUser :one
INSERT INTO users(id, email, password_hash, role) VALUES ($1, $2, $3, $4)
RETURNING id, email, password_hash, role, customer_id, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, customer_id, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, role, customer_id, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ExistsUserByRole :one
SELECT EXISTS(SELECT 1 FROM users WHERE role = $1);

-- name: UpdateUserCustomerID :exec
UPDATE users SET customer_id = $2, updated_at = now() WHERE id = $1;
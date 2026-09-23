-- name: CreateUser :one
INSERT INTO users(email, password_hash, role) VALUES ($1, $2, $3)
RETURNING id, email, password_hash, role, customer_id, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, customer_id, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, role, customer_id, created_at, updated_at
FROM users
WHERE id = $1;
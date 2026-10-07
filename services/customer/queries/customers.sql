-- name: CreateCustomer :one
INSERT INTO customers(user_id, full_name, birth_date, address, phone_number, citizenship) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, full_name, birth_date, address, phone_number, citizenship, status, created_at, updated_at;

-- name: GetCustomerByID :one
SELECT id, user_id, full_name, birth_date, address, phone_number, citizenship, status, created_at, updated_at
FROM customers
WHERE id = $1;

-- name: GetCustomerByUserID :one
SELECT id, user_id, full_name, birth_date, address, phone_number, citizenship, status, created_at, updated_at
FROM customers
WHERE user_id = $1;

-- name: UpdateCustomer :one
UPDATE customers 
SET full_name = $2, birth_date = $3, address = $4, phone_number = $5, citizenship = $6, status = $7, updated_at = now() 
WHERE id = $1
RETURNING id, user_id, full_name, birth_date, address, phone_number, citizenship, status, created_at, updated_at;

-- name: UpdateCustomerStatus :one
UPDATE customers 
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING id, user_id, full_name, birth_date, address, phone_number, citizenship, status, created_at, updated_at;

-- name: ListCustomers :many
SELECT id, user_id, full_name, birth_date, address, phone_number, citizenship, status, created_at, updated_at
FROM customers
WHERE (@statuses::text[] IS NULL OR status = ANY(@statuses::text[]))
ORDER BY created_at DESC
LIMIT @lim OFFSET @off;

-- name: CountCustomers :one
SELECT count(*) FROM customers
WHERE (@statuses::text[] IS NULL OR status = ANY(@statuses::text[]));
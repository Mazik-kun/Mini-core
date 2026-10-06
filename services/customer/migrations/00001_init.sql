-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE, -- внешний ключ из users
    full_name TEXT NOT NULL,
    birth_date DATE NOT NULL,
    address TEXT NOT NULL,
    phone_number TEXT NOT NULL,
    citizenship TEXT NOT NULL, -- гражданство
    status TEXT NOT NULL DEFAULT 'NEW' 
    CHECK (status IN ('NEW', 'PROFILE_FILLED', 'ON_KYC', 'ACTIVE', 'REJECTED', 'BLOCKED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE customers;
-- +goose StatementEnd


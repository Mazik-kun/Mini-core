-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE, -- внешний ключ из users
    full_name TEXT NOT NULL DEFAULT '',
    birth_date DATE,               -- NULL пока клиент не заполнил профиль
    address TEXT NOT NULL DEFAULT '',
    phone_number TEXT NOT NULL DEFAULT '',
    citizenship TEXT NOT NULL DEFAULT '', -- гражданство
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
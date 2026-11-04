-- +goose Up
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    account_number VARCHAR(34) NOT NULL UNIQUE,
    currency VARCHAR(3) NOT NULL,
    account_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_accounts_customer ON accounts(customer_id);

-- +goose Down
DROP TABLE accounts;
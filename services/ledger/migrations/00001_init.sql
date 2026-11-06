-- +goose Up
CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    status VARCHAR(20) NOT NULL,
    type VARCHAR(20) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE, -- Защита от повторных запросов
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE entries (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    account_id UUID NOT NULL,
    direction VARCHAR(10) NOT NULL, -- DEBIT или CREDIT
    amount BIGINT NOT NULL CHECK (amount > 0), -- Только int64! Никаких float!
    currency VARCHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Индекс для быстрого поиска проводок по счету
CREATE INDEX idx_entries_account_id ON entries(account_id);

CREATE TABLE balances (
    account_id UUID PRIMARY KEY,
    amount BIGINT NOT NULL DEFAULT 0, -- Текущий баланс
    currency VARCHAR(3) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE outbox (
    id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- PENDING, SENT
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE outbox;
DROP TABLE balances;
DROP TABLE entries;
DROP TABLE transactions;
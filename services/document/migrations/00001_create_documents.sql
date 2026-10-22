-- +goose Up
CREATE SCHEMA IF NOT EXISTS document;

CREATE TABLE document.documents (
    id            uuid PRIMARY KEY,
    customer_id   uuid        NOT NULL,
    document_type text        NOT NULL,
    object_key    text        NOT NULL UNIQUE,
    content_type  text,
    size_bytes    bigint,
    sha256        text,
    status        text        NOT NULL CHECK (status IN ('PENDING', 'CONFIRMED', 'REJECTED')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    confirmed_at  timestamptz
);

CREATE INDEX idx_documents_customer_type_status
    ON document.documents (customer_id, document_type, status);

-- +goose Down
DROP TABLE IF EXISTS document.documents;
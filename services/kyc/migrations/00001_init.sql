-- +goose Up
CREATE SCHEMA IF NOT EXISTS kyc;

CREATE TABLE kyc.applications (
    id uuid PRIMARY KEY,
    customer_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('DRAFT', 'SUBMITTED', 'AUTO_CHECKS', 'IN_REVIEW', 'APPROVED', 'REJECTED', 'AUTO_REJECTED', 'DOCS_REQUESTED')),
    officer_id uuid,
    
    -- Snapshot данных клиента на момент подачи (чтобы воркер не зависел от customer-service)
    full_name_snapshot text NOT NULL,
    birth_date_snapshot date NOT NULL,
    
    attempts int NOT NULL DEFAULT 0,
    reason text,
    
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz
);

-- Защита от одновременных активных заявок от одного клиента
CREATE UNIQUE INDEX uniq_active_app_per_customer
    ON kyc.applications(customer_id) 
    WHERE status NOT IN ('APPROVED', 'REJECTED', 'AUTO_REJECTED');

CREATE TABLE kyc.kyc_checks (
    id uuid PRIMARY KEY,
    application_id uuid NOT NULL REFERENCES kyc.applications(id),
    check_type text NOT NULL,
    status text NOT NULL,
    details jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE kyc.kyc_events (
    id uuid PRIMARY KEY,
    application_id uuid NOT NULL REFERENCES kyc.applications(id),
    actor_id uuid,
    actor_role text,
    from_status text,
    to_status text,
    comment text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE kyc.sanctions_list (
    full_name_normalized text NOT NULL,
    birth_date date NOT NULL,
    reason text NOT NULL,
    PRIMARY KEY (full_name_normalized, birth_date)
);

-- Тестовые данные для санкций (Иван Иванов)
INSERT INTO kyc.sanctions_list (full_name_normalized, birth_date, reason)
VALUES ('ivan ivanov', '1980-01-01', 'bad financing');

-- +goose Down
DROP SCHEMA IF EXISTS kyc CASCADE;
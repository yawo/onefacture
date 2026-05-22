CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE organizations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    siren        VARCHAR(9),
    pa_id        TEXT,
    settings     JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    key_hash        BYTEA NOT NULL UNIQUE,
    last_four       CHAR(4) NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX api_keys_org_idx ON api_keys(organization_id);

CREATE TYPE invoice_status AS ENUM (
    'DRAFT','VALIDATED','SUBMITTED','RECEIVED','ACCEPTED','REJECTED','PAID','CANCELLED'
);

CREATE TABLE invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    direction       TEXT NOT NULL CHECK (direction IN ('OUTBOUND','INBOUND')),
    status          invoice_status NOT NULL DEFAULT 'DRAFT',
    profile         TEXT NOT NULL,
    type_code       TEXT NOT NULL,
    number          TEXT NOT NULL,
    currency        CHAR(3) NOT NULL,
    issue_date      DATE NOT NULL,
    due_date        DATE,
    seller_siren    VARCHAR(14),
    buyer_siren     VARCHAR(14),
    pa_id           TEXT,
    pa_ref          TEXT,
    payload         JSONB NOT NULL,
    raw_xml         BYTEA,
    raw_pdf         BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, number)
);

CREATE INDEX invoices_org_status_idx ON invoices(organization_id, status);
CREATE INDEX invoices_buyer_idx ON invoices(buyer_siren) WHERE buyer_siren IS NOT NULL;
CREATE INDEX invoices_issue_idx ON invoices(issue_date);

CREATE TABLE idempotency_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key             TEXT NOT NULL,
    method          TEXT NOT NULL,
    path            TEXT NOT NULL,
    request_hash    TEXT NOT NULL,
    status_code     INT,
    response_body   JSONB,
    resource_type   TEXT,
    resource_id     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, key)
);

CREATE INDEX idempotency_keys_org_idx ON idempotency_keys(organization_id, created_at DESC);

CREATE TABLE lifecycle_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id      UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    from_status     invoice_status,
    to_status       invoice_status NOT NULL,
    pa_code         TEXT,
    pa_message      TEXT,
    payload         JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX lifecycle_events_invoice_idx ON lifecycle_events(invoice_id, occurred_at DESC);

CREATE TABLE audit_log (
    id              BIGSERIAL PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    actor           TEXT NOT NULL,
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT,
    metadata        JSONB,
    prev_hash       BYTEA,
    record_hash     BYTEA NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_org_idx ON audit_log(organization_id, occurred_at DESC);

CREATE TABLE webhook_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    secret_hash     BYTEA NOT NULL,
    events          TEXT[] NOT NULL DEFAULT ARRAY['*']::TEXT[],
    ip_allowlist    TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    mtls_required   BOOLEAN NOT NULL DEFAULT FALSE,
    mtls_cert_ref   TEXT,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX webhook_endpoints_org_idx ON webhook_endpoints(organization_id);

CREATE TABLE webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id     UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    attempts        INT NOT NULL DEFAULT 0,
    last_error      TEXT,
    next_attempt_at TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX webhook_deliveries_status_idx ON webhook_deliveries(status, next_attempt_at);

CREATE TABLE submission_dlq (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invoice_id      UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    pa_id           TEXT NOT NULL,
    error           TEXT NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}'::JSONB,
    status          TEXT NOT NULL DEFAULT 'FAILED',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    replayed_at     TIMESTAMPTZ
);

CREATE INDEX submission_dlq_org_status_idx ON submission_dlq(organization_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS shop_withdrawals (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shop_id            UUID NOT NULL,
    amount_myr_cents   BIGINT NOT NULL,
    bank_account_id    UUID NOT NULL,
    status             VARCHAR(20) NOT NULL,
    requested_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at            TIMESTAMPTZ,
    failure_reason     TEXT,
    idempotency_key    VARCHAR(128) NOT NULL UNIQUE,
    CONSTRAINT chk_shop_withdrawal_amount_positive CHECK (amount_myr_cents > 0),
    CONSTRAINT chk_shop_withdrawal_status CHECK (status IN ('pending', 'processing', 'paid', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_shop_withdrawals_shop_requested
    ON shop_withdrawals(shop_id, requested_at DESC);

CREATE TABLE IF NOT EXISTS shop_wallet_ledger (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shop_id            UUID NOT NULL,
    direction          VARCHAR(10) NOT NULL,
    amount_myr_cents   BIGINT NOT NULL,
    source_type        VARCHAR(40) NOT NULL,
    source_id          UUID NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_shop_wallet_direction CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT chk_shop_wallet_amount_positive CHECK (amount_myr_cents > 0),
    CONSTRAINT uq_shop_wallet_source UNIQUE (source_type, source_id)
);

CREATE INDEX IF NOT EXISTS idx_shop_wallet_ledger_shop_created
    ON shop_wallet_ledger(shop_id, created_at DESC);

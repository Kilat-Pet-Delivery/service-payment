CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id UUID UNIQUE NOT NULL,
    owner_id UUID NOT NULL,
    runner_id UUID,
    escrow_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    amount_cents BIGINT NOT NULL,
    platform_fee_cents BIGINT NOT NULL,
    runner_payout_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'MYR',
    payment_method VARCHAR(50),
    stripe_payment_id VARCHAR(255),
    escrow_held_at TIMESTAMPTZ,
    escrow_released_at TIMESTAMPTZ,
    refunded_at TIMESTAMPTZ,
    refund_reason TEXT,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_booking ON payments(booking_id);
CREATE INDEX idx_payments_owner ON payments(owner_id);
CREATE INDEX idx_payments_status ON payments(escrow_status);

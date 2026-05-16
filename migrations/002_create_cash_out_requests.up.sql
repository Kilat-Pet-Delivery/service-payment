-- cash_out_requests tracks every runner cash-out from request through completion.
-- runner_id references service-identity (cross-service, no FK constraint).
-- destination_id references a future payout_destinations table (cross-service, no FK constraint).

CREATE TABLE cash_out_requests (
    id                  UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    runner_id           UUID         NOT NULL,                      -- ref: service-identity runners
    amount_myr_cents    BIGINT       NOT NULL,
    fee_myr_cents       BIGINT       NOT NULL DEFAULT 50,
    destination_id      UUID         NOT NULL,                      -- ref: payout_destinations (future)
    status              VARCHAR(20)  NOT NULL DEFAULT 'pending',
    requested_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ  NULL,
    simulated_rail_id   TEXT         NULL,                          -- DuitNow-style transaction reference

    CONSTRAINT chk_cash_out_status
        CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

CREATE INDEX idx_cash_out_requests_runner ON cash_out_requests(runner_id);
CREATE INDEX idx_cash_out_requests_status ON cash_out_requests(status);

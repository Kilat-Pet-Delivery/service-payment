CREATE TABLE bank_accounts (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id               UUID NOT NULL,
    bank_code             VARCHAR(20) NOT NULL,
    account_number_masked VARCHAR(40) NOT NULL,
    account_holder_name   VARCHAR(255) NOT NULL,
    is_default            BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at            TIMESTAMPTZ NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bank_accounts_user ON bank_accounts(user_id);
CREATE INDEX idx_bank_accounts_user_default ON bank_accounts(user_id, is_default) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_bank_accounts_one_default
    ON bank_accounts(user_id)
    WHERE is_default = TRUE AND deleted_at IS NULL;

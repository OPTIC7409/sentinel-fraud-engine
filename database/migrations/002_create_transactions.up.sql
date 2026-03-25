-- Create transactions table for all financial transactions
CREATE TABLE IF NOT EXISTS transactions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             INTEGER NOT NULL,
    amount              NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    merchant_id         VARCHAR(100) NOT NULL,
    merchant_category   VARCHAR(50) NOT NULL,
    location_lat        NUMERIC(9, 6),
    location_lng        NUMERIC(9, 6),
    timestamp           TIMESTAMP NOT NULL,
    metadata            JSONB,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX idx_transactions_user_timestamp ON transactions(user_id, timestamp DESC);
CREATE INDEX idx_transactions_merchant ON transactions(merchant_id, timestamp DESC);
CREATE INDEX idx_transactions_timestamp ON transactions(timestamp DESC);
CREATE INDEX idx_transactions_merchant_category ON transactions(merchant_category);
CREATE INDEX idx_transactions_amount ON transactions(amount DESC);

COMMENT ON TABLE transactions IS 'Immutable record of all financial transactions';
COMMENT ON COLUMN transactions.timestamp IS 'Business timestamp (when transaction occurred)';
COMMENT ON COLUMN transactions.created_at IS 'System timestamp (when we ingested it)';
COMMENT ON COLUMN transactions.metadata IS 'Extensible JSON field for device fingerprint, IP, etc.';

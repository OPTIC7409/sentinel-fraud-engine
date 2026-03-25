-- Create alert priority and status enums (idempotent for partial reruns)
DO $$ BEGIN
    CREATE TYPE alert_priority AS ENUM ('medium', 'high', 'critical');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TYPE alert_status AS ENUM ('open', 'investigating', 'resolved', 'false_positive');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Create alerts table for high-risk transactions
CREATE TABLE IF NOT EXISTS alerts (
    id              SERIAL PRIMARY KEY,
    transaction_id  UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    risk_score      INTEGER NOT NULL,
    priority        alert_priority NOT NULL,
    status          alert_status NOT NULL DEFAULT 'open',
    assigned_to     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMP,
    notes           TEXT,
    CONSTRAINT check_resolved CHECK (
        (status IN ('resolved', 'false_positive') AND resolved_at IS NOT NULL) OR
        (status IN ('open', 'investigating') AND resolved_at IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_alerts_status_created ON alerts(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_assigned ON alerts(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_alerts_transaction ON alerts(transaction_id);
CREATE INDEX IF NOT EXISTS idx_alerts_priority ON alerts(priority, created_at DESC);

COMMENT ON TABLE alerts IS 'Fraud alerts triggered by high-risk transactions';
COMMENT ON COLUMN alerts.priority IS 'Auto-assigned: 75-84=medium, 85-94=high, 95+=critical';
COMMENT ON COLUMN alerts.status IS 'Workflow: open → investigating → resolved/false_positive';
COMMENT ON CONSTRAINT check_resolved ON alerts IS 'Ensures resolved_at is set when status is terminal';

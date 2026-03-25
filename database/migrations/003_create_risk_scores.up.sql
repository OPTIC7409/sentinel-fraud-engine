-- Create risk_scores table for ML model outputs
CREATE TABLE IF NOT EXISTS risk_scores (
    id                  SERIAL PRIMARY KEY,
    transaction_id      UUID NOT NULL UNIQUE REFERENCES transactions(id) ON DELETE CASCADE,
    risk_score          INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    fraud_probability   NUMERIC(5, 4) NOT NULL CHECK (fraud_probability >= 0 AND fraud_probability <= 1),
    feature_vector      JSONB NOT NULL,
    model_version       VARCHAR(50) NOT NULL,
    processing_time_ms  INTEGER,
    scored_at           TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_risk_scores_transaction ON risk_scores(transaction_id);
CREATE INDEX idx_risk_scores_score ON risk_scores(risk_score DESC);
CREATE INDEX idx_risk_scores_scored_at ON risk_scores(scored_at DESC);
CREATE INDEX idx_risk_scores_model_version ON risk_scores(model_version);

COMMENT ON TABLE risk_scores IS 'Audit trail of all fraud risk assessments';
COMMENT ON COLUMN risk_scores.transaction_id IS 'One-to-one with transactions (idempotency)';
COMMENT ON COLUMN risk_scores.risk_score IS 'Display score 0-100 derived from fraud_probability';
COMMENT ON COLUMN risk_scores.fraud_probability IS 'Raw ML model output 0.0-1.0';
COMMENT ON COLUMN risk_scores.feature_vector IS 'JSON of input features for debugging';
COMMENT ON COLUMN risk_scores.model_version IS 'Model version for A/B testing and rollback';

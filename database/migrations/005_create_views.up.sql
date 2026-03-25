-- Create view for common query: transactions with their risk scores
CREATE OR REPLACE VIEW transaction_with_risk AS
SELECT 
    t.id,
    t.user_id,
    t.amount,
    t.currency,
    t.merchant_id,
    t.merchant_category,
    t.location_lat,
    t.location_lng,
    t.timestamp,
    t.metadata,
    t.created_at,
    r.risk_score,
    r.fraud_probability,
    r.model_version,
    r.scored_at
FROM transactions t
LEFT JOIN risk_scores r ON t.id = r.transaction_id;

COMMENT ON VIEW transaction_with_risk IS 'Joins transactions and risk_scores for efficient API queries';

package models

import "time"

// TransactionEvent is the event published to Kafka after transaction ingestion.
// This is the payload sent through the event stream.
type TransactionEvent struct {
	TransactionID    string                 `json:"transaction_id"`
	UserID           int64                  `json:"user_id"`
	Amount           float64                `json:"amount"`
	Currency         string                 `json:"currency"`
	MerchantID       string                 `json:"merchant_id"`
	MerchantCategory string                 `json:"merchant_category"`
	LocationLat      *float64               `json:"location_lat,omitempty"`
	LocationLng      *float64               `json:"location_lng,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	EventID          string                 `json:"event_id"`     // For idempotency
	PublishedAt      time.Time              `json:"published_at"` // When event was published
}

// ScoredTransactionEvent is published after risk scoring.
type ScoredTransactionEvent struct {
	TransactionEvent
	RiskScore        int                `json:"risk_score"`
	FraudProbability float64            `json:"fraud_probability"`
	FeatureVector    map[string]float64 `json:"feature_vector"`
	ModelVersion     string             `json:"model_version"`
	ScoredAt         time.Time          `json:"scored_at"`
}

// AlertEvent is published when alert is triggered.
type AlertEvent struct {
	AlertID       int64     `json:"alert_id"`
	TransactionID string    `json:"transaction_id"`
	RiskScore     int       `json:"risk_score"`
	Priority      string    `json:"priority"`
	CreatedAt     time.Time `json:"created_at"`
}

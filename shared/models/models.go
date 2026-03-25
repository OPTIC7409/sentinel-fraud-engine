package models

import (
	"time"

	"github.com/google/uuid"
)

// Transaction represents a financial transaction in the system.
// This is the core domain model that flows through all services.
type Transaction struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	UserID           int64                  `json:"user_id" db:"user_id"`
	Amount           float64                `json:"amount" db:"amount"`
	Currency         string                 `json:"currency" db:"currency"`
	MerchantID       string                 `json:"merchant_id" db:"merchant_id"`
	MerchantCategory string                 `json:"merchant_category" db:"merchant_category"`
	LocationLat      *float64               `json:"location_lat,omitempty" db:"location_lat"`
	LocationLng      *float64               `json:"location_lng,omitempty" db:"location_lng"`
	Timestamp        time.Time              `json:"timestamp" db:"timestamp"`
	Metadata         map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
}

// RiskScore represents the output of fraud detection ML model.
// Stored for audit trail and analytics.
type RiskScore struct {
	ID               uuid.UUID          `json:"id" db:"id"`
	TransactionID    uuid.UUID          `json:"transaction_id" db:"transaction_id"`
	RiskScore        int                `json:"risk_score" db:"risk_score"`               // 0-100
	FraudProbability float64            `json:"fraud_probability" db:"fraud_probability"` // 0.0-1.0
	FeatureVector    map[string]float64 `json:"feature_vector" db:"feature_vector"`
	ModelVersion     string             `json:"model_version" db:"model_version"`
	ProcessingTimeMs int                `json:"processing_time_ms,omitempty" db:"processing_time_ms"`
	ScoredAt         time.Time          `json:"scored_at" db:"scored_at"`
}

// Alert represents a fraud alert triggered by high-risk transaction.
type Alert struct {
	ID            int64         `json:"id" db:"id"`
	TransactionID uuid.UUID     `json:"transaction_id" db:"transaction_id"`
	RiskScore     int           `json:"risk_score" db:"risk_score"`
	Priority      AlertPriority `json:"priority" db:"priority"`
	Status        AlertStatus   `json:"status" db:"status"`
	AssignedTo    *int64        `json:"assigned_to,omitempty" db:"assigned_to"`
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
	ResolvedAt    *time.Time    `json:"resolved_at,omitempty" db:"resolved_at"`
	Notes         string        `json:"notes,omitempty" db:"notes"`
}

// AlertPriority defines the urgency level of an alert.
type AlertPriority string

const (
	PriorityMedium   AlertPriority = "medium"   // Risk score 75-84
	PriorityHigh     AlertPriority = "high"     // Risk score 85-94
	PriorityCritical AlertPriority = "critical" // Risk score 95-100
)

// AlertStatus defines the lifecycle state of an alert.
type AlertStatus string

const (
	StatusOpen          AlertStatus = "open"
	StatusInvestigating AlertStatus = "investigating"
	StatusResolved      AlertStatus = "resolved"
	StatusFalsePositive AlertStatus = "false_positive"
)

// User represents a dashboard user (fraud analyst or admin).
type User struct {
	ID           int64      `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"` // Never send in JSON
	FullName     string     `json:"full_name" db:"full_name"`
	Role         UserRole   `json:"role" db:"role"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	LastLogin    *time.Time `json:"last_login,omitempty" db:"last_login"`
}

// UserRole defines access level for dashboard users.
type UserRole string

const (
	RoleAnalyst UserRole = "analyst"
	RoleAdmin   UserRole = "admin"
)

// ScoredTransaction combines Transaction and RiskScore for API responses.
// Used by API Gateway to avoid multiple queries.
type ScoredTransaction struct {
	Transaction
	RiskScore        *int     `json:"risk_score,omitempty"`
	FraudProbability *float64 `json:"fraud_probability,omitempty"`
	ModelVersion     *string  `json:"model_version,omitempty"`
}

// Risk thresholds used across services
const (
	LowRiskThreshold    = 0  // 0-40
	MediumRiskThreshold = 41 // 41-74
	HighRiskThreshold   = 75 // 75-100 (triggers alert)
	CriticalThreshold   = 95 // 95-100 (critical priority)
)

// DeterminePriority calculates alert priority from risk score.
func DeterminePriority(riskScore int) AlertPriority {
	switch {
	case riskScore >= CriticalThreshold:
		return PriorityCritical
	case riskScore >= 85:
		return PriorityHigh
	default:
		return PriorityMedium
	}
}

// RiskCategory returns human-readable risk category.
func RiskCategory(score int) string {
	switch {
	case score >= HighRiskThreshold:
		return "high"
	case score >= MediumRiskThreshold:
		return "medium"
	default:
		return "low"
	}
}

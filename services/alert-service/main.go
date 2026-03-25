package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/sentinel-fraud-engine/monorepo/shared/kafka"
	"github.com/sentinel-fraud-engine/monorepo/shared/logger"
	"github.com/sentinel-fraud-engine/monorepo/shared/models"
)

const (
	ServiceName = "alert-service"
)

var (
	kafkaBrokers = getEnv("KAFKA_BROKERS", "localhost:9092")
	databaseURL  = getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/sentinel_fraud?sslmode=disable")
)

func main() {
	logger.InitLogger(ServiceName, getEnv("ENV", "development") == "development")

	log.Info().
		Str("service", ServiceName).
		Str("kafka_brokers", kafkaBrokers).
		Msg("Starting alert service")

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("Database ping failed")
	}
	log.Info().Msg("Database connection established")

	// Create Kafka consumer
	consumer := kafka.NewConsumer(
		[]string{kafkaBrokers},
		kafka.TopicScoredTransactions,
		"alert-service-consumers",
	)
	defer consumer.Close()

	// Create alert handler
	handler := NewAlertHandler(db)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start consuming
	go func() {
		if err := consumer.Consume(ctx, handler.ProcessMessage); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("Consumer stopped with error")
		}
	}()

	log.Info().Msg("Alert service running, monitoring for high-risk transactions...")

	<-sigChan
	log.Info().Msg("Shutdown signal received, stopping service...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Info().Msg("Service stopped")
}

// AlertHandler processes scored transactions and creates alerts
type AlertHandler struct {
	db            *sql.DB
	alertsCreated int64
	alertsSkipped int64
}

func NewAlertHandler(db *sql.DB) *AlertHandler {
	return &AlertHandler{
		db: db,
	}
}

// ProcessMessage handles incoming scored transaction events
func (h *AlertHandler) ProcessMessage(key, value []byte) error {
	// Parse scored transaction event
	var event models.ScoredTransactionEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Debug().
		Str("transaction_id", event.TransactionID).
		Int("risk_score", event.RiskScore).
		Msg("Processing scored transaction")

	// Check if risk score meets alert threshold
	if event.RiskScore < models.HighRiskThreshold {
		h.alertsSkipped++
		log.Debug().
			Str("transaction_id", event.TransactionID).
			Int("risk_score", event.RiskScore).
			Msg("Risk score below threshold, no alert created")
		return nil
	}

	// Determine alert priority based on risk score
	priority := models.DeterminePriority(event.RiskScore)

	// Create alert
	alert, err := h.createAlert(event, priority)
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	h.alertsCreated++

	log.Warn().
		Int64("alert_id", alert.ID).
		Str("transaction_id", event.TransactionID).
		Int("risk_score", event.RiskScore).
		Str("priority", string(priority)).
		Int64("total_alerts", h.alertsCreated).
		Msg("⚠️  FRAUD ALERT CREATED")

	// Dispatch alert (webhook, notification, etc.)
	if err := h.dispatchAlert(alert, event); err != nil {
		log.Error().Err(err).Int64("alert_id", alert.ID).Msg("Failed to dispatch alert")
		// Don't return error - alert is saved in DB
	}

	return nil
}

func (h *AlertHandler) createAlert(event models.ScoredTransactionEvent, priority models.AlertPriority) (*models.Alert, error) {
	txnID, err := uuid.Parse(event.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction ID: %w", err)
	}

	query := `
		INSERT INTO alerts (transaction_id, risk_score, priority, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var alertID int64
	err = h.db.QueryRow(query,
		txnID,
		event.RiskScore,
		priority,
		models.StatusOpen,
		time.Now(),
	).Scan(&alertID)

	if err != nil {
		return nil, err
	}

	return &models.Alert{
		ID:            alertID,
		TransactionID: txnID,
		RiskScore:     event.RiskScore,
		Priority:      priority,
		Status:        models.StatusOpen,
		CreatedAt:     time.Now(),
	}, nil
}

func (h *AlertHandler) dispatchAlert(alert *models.Alert, event models.ScoredTransactionEvent) error {
	// In production, this would:
	// 1. Send webhook to fraud investigation system
	// 2. Send notification to on-call analyst
	// 3. Create ticket in case management system

	// For now, log structured alert data
	log.Warn().
		Int64("alert_id", alert.ID).
		Str("transaction_id", event.TransactionID).
		Int64("user_id", event.UserID).
		Float64("amount", event.Amount).
		Str("merchant_id", event.MerchantID).
		Str("merchant_category", event.MerchantCategory).
		Int("risk_score", event.RiskScore).
		Float64("fraud_probability", event.FraudProbability).
		Str("priority", string(alert.Priority)).
		Interface("feature_vector", event.FeatureVector).
		Msg("Alert dispatched for investigation")

	// Simulate webhook call (would be actual HTTP POST in production)
	webhookURL := getEnv("ALERT_WEBHOOK_URL", "")
	if webhookURL != "" {
		log.Info().Str("webhook_url", webhookURL).Msg("Would send webhook to fraud investigation system")
		// http.Post(webhookURL, "application/json", alertPayload)
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

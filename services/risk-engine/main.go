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
	"github.com/sentinel-fraud-engine/monorepo/shared/fraudmodel"
	"github.com/sentinel-fraud-engine/monorepo/shared/kafka"
	"github.com/sentinel-fraud-engine/monorepo/shared/logger"
	"github.com/sentinel-fraud-engine/monorepo/shared/models"
)

const (
	ServiceName = "risk-engine"
)

var (
	kafkaBrokers = getEnv("KAFKA_BROKERS", "localhost:9092")
	databaseURL  = getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/sentinel_fraud?sslmode=disable")
	modelDir     = getEnv("MODEL_DIR", "../../ml/model")
)

func main() {
	logger.InitLogger(ServiceName, getEnv("ENV", "development") == "development")

	log.Info().
		Str("service", ServiceName).
		Str("kafka_brokers", kafkaBrokers).
		Str("model_dir", modelDir).
		Msg("Starting risk engine service")

	// Initialize fraud scorer
	scorer, err := fraudmodel.NewFraudScorer(modelDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize fraud scorer")
	}
	log.Info().Str("model_version", scorer.GetModelVersion()).Msg("Fraud scorer loaded")

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

	// Create Kafka consumer and producer
	consumer := kafka.NewConsumer(
		[]string{kafkaBrokers},
		kafka.TopicRawTransactions,
		"risk-engine-consumers",
	)
	defer consumer.Close()

	producer := kafka.NewProducer([]string{kafkaBrokers}, kafka.TopicScoredTransactions)
	defer producer.Close()

	// Ensure topics exist
	kafka.EnsureTopicsExist([]string{kafkaBrokers}, []string{
		kafka.TopicRawTransactions,
		kafka.TopicScoredTransactions,
	})

	// Create processor
	processor := NewRiskProcessor(db, scorer, producer)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start consuming in goroutine
	go func() {
		if err := consumer.Consume(ctx, processor.ProcessMessage); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("Consumer stopped with error")
		}
	}()

	log.Info().Msg("Risk engine running, consuming transactions...")

	<-sigChan
	log.Info().Msg("Shutdown signal received, stopping service...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Info().Msg("Service stopped")
}

// RiskProcessor processes transactions and assigns risk scores
type RiskProcessor struct {
	db             *sql.DB
	scorer         fraudmodel.FraudScorer
	producer       *kafka.Producer
	processedCount int64
}

func NewRiskProcessor(db *sql.DB, scorer fraudmodel.FraudScorer, producer *kafka.Producer) *RiskProcessor {
	return &RiskProcessor{
		db:       db,
		scorer:   scorer,
		producer: producer,
	}
}

// ProcessMessage handles incoming transaction events
func (p *RiskProcessor) ProcessMessage(key, value []byte) error {
	startTime := time.Now()

	// Parse transaction event
	var event models.TransactionEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Info().
		Str("transaction_id", event.TransactionID).
		Int64("user_id", event.UserID).
		Float64("amount", event.Amount).
		Msg("Processing transaction")

	// Convert event to transaction model
	txn, err := p.eventToTransaction(event)
	if err != nil {
		return fmt.Errorf("failed to convert event: %w", err)
	}

	// Save transaction to database first (idempotency check)
	if err := p.saveTransaction(txn); err != nil {
		// Check if already processed (duplicate event)
		if isDuplicateError(err) {
			log.Warn().Str("transaction_id", txn.ID.String()).Msg("Transaction already processed, skipping")
			return nil
		}
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	// Run fraud scoring
	prediction, err := p.scorer.PredictRisk(*txn)
	if err != nil {
		return fmt.Errorf("fraud scoring failed: %w", err)
	}

	// Save risk score to database
	if err := p.saveRiskScore(txn.ID, prediction); err != nil {
		return fmt.Errorf("failed to save risk score: %w", err)
	}

	// Publish scored transaction event
	if err := p.publishScoredTransaction(event, prediction); err != nil {
		return fmt.Errorf("failed to publish scored transaction: %w", err)
	}

	processingTime := time.Since(startTime)
	p.processedCount++

	log.Info().
		Str("transaction_id", txn.ID.String()).
		Int("risk_score", prediction.RiskScore).
		Float64("fraud_probability", prediction.FraudProbability).
		Int64("processing_ms", processingTime.Milliseconds()).
		Int64("total_processed", p.processedCount).
		Msg("Transaction scored successfully")

	return nil
}

func (p *RiskProcessor) eventToTransaction(event models.TransactionEvent) (*models.Transaction, error) {
	txnID, err := uuid.Parse(event.TransactionID)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction ID: %w", err)
	}

	return &models.Transaction{
		ID:               txnID,
		UserID:           event.UserID,
		Amount:           event.Amount,
		Currency:         event.Currency,
		MerchantID:       event.MerchantID,
		MerchantCategory: event.MerchantCategory,
		LocationLat:      event.LocationLat,
		LocationLng:      event.LocationLng,
		Timestamp:        event.Timestamp,
		Metadata:         event.Metadata,
		CreatedAt:        event.PublishedAt,
	}, nil
}

func (p *RiskProcessor) saveTransaction(txn *models.Transaction) error {
	query := `
		INSERT INTO transactions (id, user_id, amount, currency, merchant_id, merchant_category, 
			location_lat, location_lng, timestamp, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO NOTHING
	`

	metadataJSON, _ := json.Marshal(txn.Metadata)

	_, err := p.db.Exec(query,
		txn.ID, txn.UserID, txn.Amount, txn.Currency, txn.MerchantID, txn.MerchantCategory,
		txn.LocationLat, txn.LocationLng, txn.Timestamp, metadataJSON, txn.CreatedAt,
	)

	return err
}

func (p *RiskProcessor) saveRiskScore(txnID uuid.UUID, prediction *fraudmodel.RiskPrediction) error {
	query := `
		INSERT INTO risk_scores (transaction_id, risk_score, fraud_probability, feature_vector, 
			model_version, processing_time_ms, scored_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (transaction_id) DO NOTHING
	`

	featuresJSON, _ := json.Marshal(prediction.FeatureVector)

	_, err := p.db.Exec(query,
		txnID, prediction.RiskScore, prediction.FraudProbability, featuresJSON,
		prediction.ModelVersion, prediction.ProcessingTimeMs, time.Now(),
	)

	return err
}

func (p *RiskProcessor) publishScoredTransaction(event models.TransactionEvent, prediction *fraudmodel.RiskPrediction) error {
	scoredEvent := models.ScoredTransactionEvent{
		TransactionEvent: event,
		RiskScore:        prediction.RiskScore,
		FraudProbability: prediction.FraudProbability,
		FeatureVector:    prediction.FeatureVector,
		ModelVersion:     prediction.ModelVersion,
		ScoredAt:         time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("%d", event.UserID)
	return p.producer.Publish(ctx, key, scoredEvent)
}

func isDuplicateError(err error) bool {
	// PostgreSQL duplicate key error has specific error code
	return err != nil && (err.Error() == "sql: no rows affected" ||
		err.Error() == "pq: duplicate key value violates unique constraint")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

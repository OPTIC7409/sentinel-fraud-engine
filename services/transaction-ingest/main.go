package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sentinel-fraud-engine/monorepo/shared/kafka"
	"github.com/sentinel-fraud-engine/monorepo/shared/logger"
	"github.com/sentinel-fraud-engine/monorepo/shared/models"
)

const (
	ServiceName         = "transaction-ingest"
	TransactionsPerSec  = 10 // Configurable load
	NumWorkers          = 50  // Concurrent workers
)

var (
	kafkaBrokers = getEnv("KAFKA_BROKERS", "localhost:9092")
)

func main() {
	// Initialize structured logging
	logger.InitLogger(ServiceName, getEnv("ENV", "development") == "development")
	
	log.Info().
		Str("service", ServiceName).
		Str("kafka_brokers", kafkaBrokers).
		Int("target_tps", TransactionsPerSec).
		Msg("Starting transaction ingest service")

	// Create Kafka producer
	producer := kafka.NewProducer([]string{kafkaBrokers}, kafka.TopicRawTransactions)
	defer producer.Close()

	// Ensure topic exists
	if err := kafka.EnsureTopicsExist([]string{kafkaBrokers}, []string{kafka.TopicRawTransactions}); err != nil {
		log.Warn().Err(err).Msg("Topic creation warning")
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start transaction generator
	generator := NewTransactionGenerator(producer)
	
	go func() {
		if err := generator.Run(ctx); err != nil {
			log.Error().Err(err).Msg("Generator stopped with error")
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info().Msg("Shutdown signal received, stopping service...")
	cancel()
	
	time.Sleep(2 * time.Second) // Allow in-flight transactions to complete
	log.Info().Msg("Service stopped")
}

// TransactionGenerator generates synthetic transactions
type TransactionGenerator struct {
	producer      *kafka.Producer
	rand          *rand.Rand
	merchantCategories []string
	currencies    []string
	locations     []Location
}

type Location struct {
	Lat float64
	Lng float64
}

func NewTransactionGenerator(producer *kafka.Producer) *TransactionGenerator {
	return &TransactionGenerator{
		producer: producer,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		merchantCategories: []string{
			"groceries", "restaurants", "gas_stations", "utilities",
			"healthcare", "retail", "electronics", "jewelry",
			"wire_transfer", "crypto", "travel", "entertainment",
		},
		currencies: []string{"USD", "EUR", "GBP"},
		locations: []Location{
			{40.7128, -74.0060},  // New York
			{34.0522, -118.2437}, // Los Angeles
			{41.8781, -87.6298},  // Chicago
			{29.7604, -95.3698},  // Houston
		},
	}
}

// Run starts generating transactions at target rate
func (g *TransactionGenerator) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second / time.Duration(TransactionsPerSec))
	defer ticker.Stop()

	transactionCount := 0
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Int("total_transactions", transactionCount).
				Float64("duration_seconds", time.Since(startTime).Seconds()).
				Msg("Generator stopped")
			return ctx.Err()
			
		case <-ticker.C:
			txn := g.generateTransaction()
			if err := g.publishTransaction(ctx, txn); err != nil {
				log.Error().Err(err).Str("transaction_id", txn.ID.String()).Msg("Failed to publish transaction")
				continue
			}
			
			transactionCount++
			if transactionCount%100 == 0 {
				tps := float64(transactionCount) / time.Since(startTime).Seconds()
				log.Info().
					Int("count", transactionCount).
					Float64("tps", tps).
					Msg("Transaction generation progress")
			}
		}
	}
}

// generateTransaction creates a realistic synthetic transaction
func (g *TransactionGenerator) generateTransaction() *models.Transaction {
	category := g.merchantCategories[g.rand.Intn(len(g.merchantCategories))]
	location := g.locations[g.rand.Intn(len(g.locations))]
	
	// Amount varies by category
	var amount float64
	switch category {
	case "groceries", "restaurants", "gas_stations":
		amount = 10 + g.rand.Float64()*190 // $10-$200
	case "electronics", "jewelry":
		amount = 100 + g.rand.Float64()*2900 // $100-$3000
	case "wire_transfer", "crypto":
		amount = 500 + g.rand.Float64()*9500 // $500-$10000
	default:
		amount = 20 + g.rand.Float64()*280 // $20-$300
	}
	
	lat := location.Lat + (g.rand.Float64()-0.5)*0.4 // ~20km variance
	lng := location.Lng + (g.rand.Float64()-0.5)*0.4

	return &models.Transaction{
		ID:               uuid.New(),
		UserID:           int64(g.rand.Intn(10000) + 1), // 10k users
		Amount:           float64(int(amount*100)) / 100, // Round to cents
		Currency:         "USD",
		MerchantID:       fmt.Sprintf("merchant_%s_%d", category, g.rand.Intn(1000)),
		MerchantCategory: category,
		LocationLat:      &lat,
		LocationLng:      &lng,
		Timestamp:        time.Now(),
		Metadata:         map[string]interface{}{"source": "generator"},
		CreatedAt:        time.Now(),
	}
}

// publishTransaction sends transaction to Kafka
func (g *TransactionGenerator) publishTransaction(ctx context.Context, txn *models.Transaction) error {
	// Convert to event format
	event := models.TransactionEvent{
		TransactionID:    txn.ID.String(),
		UserID:           txn.UserID,
		Amount:           txn.Amount,
		Currency:         txn.Currency,
		MerchantID:       txn.MerchantID,
		MerchantCategory: txn.MerchantCategory,
		LocationLat:      txn.LocationLat,
		LocationLng:      txn.LocationLng,
		Timestamp:        txn.Timestamp,
		Metadata:         txn.Metadata,
		EventID:          uuid.New().String(),
		PublishedAt:      time.Now(),
	}

	// Publish to Kafka (key by user_id for ordering)
	key := fmt.Sprintf("%d", txn.UserID)
	if err := g.producer.Publish(ctx, key, event); err != nil {
		return fmt.Errorf("kafka publish failed: %w", err)
	}

	log.Debug().
		Str("transaction_id", txn.ID.String()).
		Int64("user_id", txn.UserID).
		Float64("amount", txn.Amount).
		Str("category", txn.MerchantCategory).
		Msg("Transaction published")

	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

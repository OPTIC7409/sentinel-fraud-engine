package fraudmodel

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sentinel-fraud-engine/monorepo/shared/models"
)

func TestFeatureExtraction(t *testing.T) {
	scorer := &LogisticRegressionScorer{
		userPatterns:  make(map[int64]*UserPattern),
		merchantRisks: initializeMerchantRisks(),
	}

	lat := 40.7128
	lng := -74.0060

	txn := models.Transaction{
		ID:               uuid.New(),
		UserID:           12345,
		Amount:           150.00,
		Currency:         "USD",
		MerchantID:       "merchant_electronics_123",
		MerchantCategory: "electronics",
		LocationLat:      &lat,
		LocationLng:      &lng,
		Timestamp:        time.Now(),
		CreatedAt:        time.Now(),
	}

	features := scorer.extractFeatures(txn)

	// Test amount normalization
	if features["amount_normalized"] <= 0 || features["amount_normalized"] > 1 {
		t.Errorf("Amount normalized out of range: %f", features["amount_normalized"])
	}

	// Test merchant category risk
	if features["merchant_category_risk"] != 0.7 { // electronics = 0.7
		t.Errorf("Expected merchant risk 0.7, got %f", features["merchant_category_risk"])
	}

	// Test time anomaly (should be 0 during normal hours)
	if txn.Timestamp.Hour() >= 6 && txn.Timestamp.Hour() <= 23 {
		if features["time_anomaly"] != 0 {
			t.Errorf("Expected time_anomaly 0 during normal hours, got %f", features["time_anomaly"])
		}
	}
}

func TestVelocityScoring(t *testing.T) {
	scorer := &LogisticRegressionScorer{
		userPatterns:  make(map[int64]*UserPattern),
		merchantRisks: initializeMerchantRisks(),
	}

	userID := int64(12345)

	// First transaction should have 0 velocity
	velocity1 := scorer.calculateVelocityScore(userID)
	if velocity1 != 0 {
		t.Errorf("First transaction should have 0 velocity, got %f", velocity1)
	}

	// Simulate user pattern with recent transactions
	scorer.userPatterns[userID] = &UserPattern{
		RecentTxnCount: 5,
		LastResetTime:  time.Now(),
	}

	velocity2 := scorer.calculateVelocityScore(userID)
	if velocity2 != 0.5 { // 5/10 = 0.5
		t.Errorf("Expected velocity 0.5, got %f", velocity2)
	}

	// Test velocity cap at 1.0
	scorer.userPatterns[userID].RecentTxnCount = 15
	velocity3 := scorer.calculateVelocityScore(userID)
	if velocity3 > 1.0 {
		t.Errorf("Velocity should be capped at 1.0, got %f", velocity3)
	}
}

func TestHaversineDistance(t *testing.T) {
	// New York to Los Angeles (approximate)
	nyLat, nyLng := 40.7128, -74.0060
	laLat, laLng := 34.0522, -118.2437

	distance := haversineDistance(nyLat, nyLng, laLat, laLng)

	// Should be approximately 3944 km
	if distance < 3900 || distance > 4000 {
		t.Errorf("Expected distance ~3944 km, got %f", distance)
	}

	// Same location should be 0
	distance2 := haversineDistance(nyLat, nyLng, nyLat, nyLng)
	if distance2 != 0 {
		t.Errorf("Same location distance should be 0, got %f", distance2)
	}
}

func TestMerchantRiskInitialization(t *testing.T) {
	risks := initializeMerchantRisks()

	// Test known high-risk categories
	if risks["wire_transfer"] != 0.9 {
		t.Errorf("Expected wire_transfer risk 0.9, got %f", risks["wire_transfer"])
	}

	if risks["crypto"] != 0.85 {
		t.Errorf("Expected crypto risk 0.85, got %f", risks["crypto"])
	}

	// Test low-risk categories
	if risks["groceries"] != 0.1 {
		t.Errorf("Expected groceries risk 0.1, got %f", risks["groceries"])
	}
}

func TestUserPatternUpdate(t *testing.T) {
	scorer := &LogisticRegressionScorer{
		userPatterns:  make(map[int64]*UserPattern),
		merchantRisks: initializeMerchantRisks(),
	}

	lat := 40.7128
	lng := -74.0060

	txn := models.Transaction{
		ID:          uuid.New(),
		UserID:      12345,
		Amount:      100.00,
		LocationLat: &lat,
		LocationLng: &lng,
		Timestamp:   time.Now(),
		CreatedAt:   time.Now(),
	}

	scorer.updateUserPattern(txn)

	pattern, exists := scorer.userPatterns[txn.UserID]
	if !exists {
		t.Fatal("User pattern should be created")
	}

	if pattern.TotalTransactions != 1 {
		t.Errorf("Expected 1 transaction, got %d", pattern.TotalTransactions)
	}

	if pattern.RecentTxnCount != 1 {
		t.Errorf("Expected recent count 1, got %d", pattern.RecentTxnCount)
	}

	// Update again - should increment counters
	scorer.updateUserPattern(txn)

	if pattern.TotalTransactions != 2 {
		t.Errorf("Expected 2 total transactions, got %d", pattern.TotalTransactions)
	}
}

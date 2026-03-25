package fraudmodel

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sentinel-fraud-engine/monorepo/shared/models"
)

const (
	ModelVersion = "v1.0.0"
)

// FraudScorer interface for fraud risk scoring
type FraudScorer interface {
	PredictRisk(tx models.Transaction) (*RiskPrediction, error)
	GetModelVersion() string
}

// RiskPrediction holds the output of fraud prediction
type RiskPrediction struct {
	FraudProbability float64            `json:"fraud_probability"`
	RiskScore        int                `json:"risk_score"`
	FeatureVector    map[string]float64 `json:"feature_vector"`
	ModelVersion     string             `json:"model_version"`
	ProcessingTimeMs int64              `json:"processing_time_ms"`
}

// LogisticRegressionScorer implements fraud scoring using trained model
type LogisticRegressionScorer struct {
	modelPath     string
	pythonScript  string
	mu            sync.RWMutex
	userPatterns  map[int64]*UserPattern
	merchantRisks map[string]float64
}

// UserPattern tracks user behavior for velocity and location features
type UserPattern struct {
	TypicalLat        float64
	TypicalLng        float64
	RecentTxnCount    int
	LastResetTime     time.Time
	TotalTransactions int
}

// NewFraudScorer creates a new fraud scorer instance
func NewFraudScorer(modelDir string) (FraudScorer, error) {
	modelPath := filepath.Join(modelDir, fmt.Sprintf("fraud_model_%s.joblib", ModelVersion))

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s", modelPath)
	}

	scriptPath := filepath.Join(modelDir, "inference.py")

	scorer := &LogisticRegressionScorer{
		modelPath:     modelPath,
		pythonScript:  scriptPath,
		userPatterns:  make(map[int64]*UserPattern),
		merchantRisks: initializeMerchantRisks(),
	}

	log.Info().Str("model_path", modelPath).Str("model_version", ModelVersion).Msg("Fraud scorer initialized")
	return scorer, nil
}

// PredictRisk performs fraud risk prediction on a transaction
func (s *LogisticRegressionScorer) PredictRisk(tx models.Transaction) (*RiskPrediction, error) {
	startTime := time.Now()
	features := s.extractFeatures(tx)
	fraudProb, err := s.runInference(features)
	if err != nil {
		return nil, fmt.Errorf("model inference failed: %w", err)
	}

	riskScore := int(math.Round(fraudProb * 100))
	if riskScore < 0 {
		riskScore = 0
	} else if riskScore > 100 {
		riskScore = 100
	}

	processingTime := time.Since(startTime)
	s.updateUserPattern(tx)

	return &RiskPrediction{
		FraudProbability: fraudProb,
		RiskScore:        riskScore,
		FeatureVector:    features,
		ModelVersion:     ModelVersion,
		ProcessingTimeMs: processingTime.Milliseconds(),
	}, nil
}

func (s *LogisticRegressionScorer) extractFeatures(tx models.Transaction) map[string]float64 {
	features := make(map[string]float64)
	maxAmount := 10000.0
	features["amount_normalized"] = math.Log(tx.Amount+1) / math.Log(maxAmount)
	features["velocity_score"] = s.calculateVelocityScore(tx.UserID)
	features["location_deviation"] = s.calculateLocationDeviation(tx)
	features["time_anomaly"] = s.calculateTimeAnomaly(tx.Timestamp)
	features["merchant_category_risk"] = s.getMerchantCategoryRisk(tx.MerchantCategory)
	return features
}

func (s *LogisticRegressionScorer) calculateVelocityScore(userID int64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pattern, exists := s.userPatterns[userID]
	if !exists {
		return 0.0
	}

	if time.Since(pattern.LastResetTime) > time.Hour {
		return 0.0
	}

	velocity := float64(pattern.RecentTxnCount) / 10.0
	if velocity > 1.0 {
		velocity = 1.0
	}
	return velocity
}

func (s *LogisticRegressionScorer) calculateLocationDeviation(tx models.Transaction) float64 {
	if tx.LocationLat == nil || tx.LocationLng == nil {
		return 0.0
	}

	s.mu.RLock()
	pattern, exists := s.userPatterns[tx.UserID]
	s.mu.RUnlock()

	if !exists || pattern.TotalTransactions < 5 {
		return 0.0
	}

	distKm := haversineDistance(pattern.TypicalLat, pattern.TypicalLng, *tx.LocationLat, *tx.LocationLng)
	deviation := distKm / 1000.0
	if deviation > 1.0 {
		deviation = 1.0
	}
	return deviation
}

func (s *LogisticRegressionScorer) calculateTimeAnomaly(timestamp time.Time) float64 {
	hour := timestamp.Hour()
	if hour < 6 || hour > 23 {
		return 1.0
	}
	return 0.0
}

func (s *LogisticRegressionScorer) getMerchantCategoryRisk(category string) float64 {
	s.mu.RLock()
	risk, exists := s.merchantRisks[category]
	s.mu.RUnlock()

	if !exists {
		return 0.5
	}
	return risk
}

func (s *LogisticRegressionScorer) updateUserPattern(tx models.Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pattern, exists := s.userPatterns[tx.UserID]
	if !exists {
		pattern = &UserPattern{LastResetTime: time.Now()}
		s.userPatterns[tx.UserID] = pattern
	}

	if tx.LocationLat != nil && tx.LocationLng != nil {
		if pattern.TotalTransactions == 0 {
			pattern.TypicalLat = *tx.LocationLat
			pattern.TypicalLng = *tx.LocationLng
		} else {
			alpha := 0.1
			pattern.TypicalLat = pattern.TypicalLat*(1-alpha) + *tx.LocationLat*alpha
			pattern.TypicalLng = pattern.TypicalLng*(1-alpha) + *tx.LocationLng*alpha
		}
	}

	if time.Since(pattern.LastResetTime) > time.Hour {
		pattern.RecentTxnCount = 1
		pattern.LastResetTime = time.Now()
	} else {
		pattern.RecentTxnCount++
	}

	pattern.TotalTransactions++
}

func (s *LogisticRegressionScorer) runInference(features map[string]float64) (float64, error) {
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal features: %w", err)
	}

	cmd := exec.Command("python3", s.pythonScript, string(featuresJSON))
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("python inference failed: %w", err)
	}

	var result struct {
		Probability float64 `json:"probability"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("failed to parse inference result: %w", err)
	}

	return result.Probability, nil
}

func (s *LogisticRegressionScorer) GetModelVersion() string {
	return ModelVersion
}

func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func initializeMerchantRisks() map[string]float64 {
	return map[string]float64{
		"groceries":     0.1,
		"restaurants":   0.2,
		"gas_stations":  0.15,
		"utilities":     0.05,
		"healthcare":    0.08,
		"retail":        0.25,
		"electronics":   0.7,
		"jewelry":       0.75,
		"wire_transfer": 0.9,
		"crypto":        0.85,
		"travel":        0.4,
		"entertainment": 0.3,
	}
}

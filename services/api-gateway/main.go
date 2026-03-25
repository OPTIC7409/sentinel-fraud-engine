package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/sentinel-fraud-engine/monorepo/shared/logger"
	"golang.org/x/time/rate"
)

const (
	ServiceName = "api-gateway"
	Port        = 8000
)

var (
	databaseURL = getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/sentinel_fraud?sslmode=disable")
	jwtSecret   = []byte(getEnv("JWT_SECRET", "sentinel-secret-change-in-production"))
)

func main() {
	logger.InitLogger(ServiceName, getEnv("ENV", "development") == "development")

	log.Info().
		Str("service", ServiceName).
		Int("port", Port).
		Msg("Starting API gateway")

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

	// Create router
	router := mux.NewRouter()

	// Create API handler
	api := NewAPIHandler(db)

	// Public routes
	router.HandleFunc("/health", api.HealthCheck).Methods("GET")
	router.HandleFunc("/api/auth/login", api.Login).Methods("POST")

	// Protected routes (require JWT)
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(api.AuthMiddleware)
	protected.Use(api.RateLimitMiddleware)

	protected.HandleFunc("/transactions", api.GetTransactions).Methods("GET")
	protected.HandleFunc("/transactions/{id}", api.GetTransaction).Methods("GET")
	protected.HandleFunc("/alerts", api.GetAlerts).Methods("GET")
	protected.HandleFunc("/alerts/{id}", api.GetAlert).Methods("GET")
	protected.HandleFunc("/alerts/{id}/resolve", api.ResolveAlert).Methods("POST")
	protected.HandleFunc("/metrics", api.GetMetrics).Methods("GET")

	// CORS must wrap the whole router: OPTIONS preflight does not match route-specific
	// Methods("POST"|"GET"), so mux would 404 without headers if CORS were only router.Use.
	handler := corsMiddleware(router)

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Info().Int("port", Port).Msg("API gateway listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}
	log.Info().Msg("Server stopped")
}

// APIHandler handles HTTP requests
type APIHandler struct {
	db          *sql.DB
	rateLimiter *rate.Limiter
}

func NewAPIHandler(db *sql.DB) *APIHandler {
	return &APIHandler{
		db:          db,
		rateLimiter: rate.NewLimiter(rate.Limit(100), 200), // 100 req/s, burst 200
	}
}

// HealthCheck returns service health
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   ServiceName,
		"timestamp": time.Now().Unix(),
	}

	// Check database
	if err := h.db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["database"] = "down"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
}

// Login authenticates user and returns JWT token
func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Demo: issue JWT without checking password_hash (replace with real auth for prod).

	token, err := generateJWT(creds.Email, "analyst")
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"token": token,
		"user": map[string]string{
			"email": creds.Email,
			"role":  "analyst",
		},
	}

	json.NewEncoder(w).Encode(response)
	log.Info().Str("email", creds.Email).Msg("User logged in")
}

// GetTransactions returns paginated transactions with risk scores
func (h *APIHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT t.id, t.user_id, t.amount, t.currency, t.merchant_id, t.merchant_category,
			   t.timestamp, r.risk_score, r.fraud_probability
		FROM transactions t
		LEFT JOIN risk_scores r ON t.id = r.transaction_id
		ORDER BY t.timestamp DESC
		LIMIT $1
	`

	rows, err := h.db.Query(query, limit)
	if err != nil {
		log.Error().Err(err).Msg("Query failed")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	transactions := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, userID, currency, merchantID, merchantCategory string
			amount                                             float64
			timestamp                                          time.Time
			riskScore                                          *int
			fraudProb                                          *float64
		)

		if err := rows.Scan(&id, &userID, &amount, &currency, &merchantID, &merchantCategory,
			&timestamp, &riskScore, &fraudProb); err != nil {
			continue
		}

		txn := map[string]interface{}{
			"id":                id,
			"user_id":           userID,
			"amount":            amount,
			"currency":          currency,
			"merchant_id":       merchantID,
			"merchant_category": merchantCategory,
			"timestamp":         timestamp.Unix(),
		}

		if riskScore != nil {
			txn["risk_score"] = *riskScore
			txn["fraud_probability"] = *fraudProb
		}

		transactions = append(transactions, txn)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

// GetTransaction returns a single transaction by ID
func (h *APIHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	query := `
		SELECT t.id, t.user_id, t.amount, t.currency, t.merchant_id, t.merchant_category,
			   t.timestamp, r.risk_score, r.fraud_probability, r.feature_vector
		FROM transactions t
		LEFT JOIN risk_scores r ON t.id = r.transaction_id
		WHERE t.id = $1
	`

	var (
		txnID, userID, currency, merchantID, merchantCategory string
		amount                                                float64
		timestamp                                             time.Time
		riskScore                                             *int
		fraudProb                                             *float64
		features                                              []byte
	)

	err := h.db.QueryRow(query, id).Scan(
		&txnID, &userID, &amount, &currency, &merchantID, &merchantCategory,
		&timestamp, &riskScore, &fraudProb, &features,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("Query failed")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	txn := map[string]interface{}{
		"id":                txnID,
		"user_id":           userID,
		"amount":            amount,
		"currency":          currency,
		"merchant_id":       merchantID,
		"merchant_category": merchantCategory,
		"timestamp":         timestamp.Unix(),
	}

	if riskScore != nil {
		txn["risk_score"] = *riskScore
		txn["fraud_probability"] = *fraudProb

		var featVec map[string]float64
		json.Unmarshal(features, &featVec)
		txn["feature_vector"] = featVec
	}

	json.NewEncoder(w).Encode(txn)
}

// GetAlerts returns alerts filtered by status
func (h *APIHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "open"
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT a.id, a.transaction_id, a.risk_score, a.priority, a.status, a.created_at
		FROM alerts a
		WHERE a.status = $1
		ORDER BY a.priority DESC, a.created_at DESC
		LIMIT $2
	`

	rows, err := h.db.Query(query, status, limit)
	if err != nil {
		log.Error().Err(err).Msg("Query failed")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	alerts := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, riskScore                int64
			txnID, priority, alertStatus string
			createdAt                    time.Time
		)

		if err := rows.Scan(&id, &txnID, &riskScore, &priority, &alertStatus, &createdAt); err != nil {
			continue
		}

		alert := map[string]interface{}{
			"id":             id,
			"transaction_id": txnID,
			"risk_score":     riskScore,
			"priority":       priority,
			"status":         alertStatus,
			"created_at":     createdAt.Unix(),
		}

		alerts = append(alerts, alert)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// GetAlert returns single alert details
func (h *APIHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	query := `SELECT id, transaction_id, risk_score, priority, status, created_at FROM alerts WHERE id = $1`

	var (
		alertID, riskScore      int64
		txnID, priority, status string
		createdAt               time.Time
	)

	err := h.db.QueryRow(query, id).Scan(&alertID, &txnID, &riskScore, &priority, &status, &createdAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Alert not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	alert := map[string]interface{}{
		"id":             alertID,
		"transaction_id": txnID,
		"risk_score":     riskScore,
		"priority":       priority,
		"status":         status,
		"created_at":     createdAt.Unix(),
	}

	json.NewEncoder(w).Encode(alert)
}

// ResolveAlert marks an alert as resolved
func (h *APIHandler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	query := `UPDATE alerts SET status = 'resolved', resolved_at = $1 WHERE id = $2`

	_, err := h.db.Exec(query, time.Now(), id)
	if err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "resolved"})
	log.Info().Str("alert_id", id).Msg("Alert resolved")
}

// GetMetrics returns system metrics
func (h *APIHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	// Query basic stats
	var totalTxns, totalAlerts, openAlerts int64

	h.db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&totalTxns)
	h.db.QueryRow("SELECT COUNT(*) FROM alerts").Scan(&totalAlerts)
	h.db.QueryRow("SELECT COUNT(*) FROM alerts WHERE status = 'open'").Scan(&openAlerts)

	metrics := map[string]interface{}{
		"total_transactions": totalTxns,
		"total_alerts":       totalAlerts,
		"open_alerts":        openAlerts,
		"timestamp":          time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(metrics)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

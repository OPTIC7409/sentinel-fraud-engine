# Sentinel Fraud Engine - System Architecture

## Executive Summary

The Sentinel Fraud Engine is a production-grade, real-time fraud detection and risk scoring platform designed for high-throughput financial transaction processing. The system processes 1000+ transactions per second through a distributed, event-driven microservices architecture.

## Design Philosophy

This system is architected using principles employed by tier-1 financial institutions:

- **Event-Driven Architecture**: Decoupled services communicate via message streams, enabling independent scaling and fault isolation
- **Domain-Driven Design**: Each service encapsulates specific business capabilities with clear boundaries
- **Fail-Fast with Observability**: Explicit error handling, structured logging, and metrics at every layer
- **Data Integrity First**: ACID guarantees for financial data, idempotency for event processing

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          FRONTEND LAYER (Next.js)                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │ Live Feed    │  │ Risk Viz     │  │ Alerts Panel │  │ Trends/Charts│   │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
└────────────────────────────────┬────────────────────────────────────────────┘
                                 │ WebSocket / REST
                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        API GATEWAY (Go)                                      │
│  • JWT Authentication                                                        │
│  • Rate Limiting (Token Bucket)                                             │
│  • Input Validation                                                          │
│  • Request/Response Transformation                                           │
└────────────────────────────────┬────────────────────────────────────────────┘
                                 │
                ┌────────────────┼────────────────┐
                ▼                ▼                ▼
        ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
        │ PostgreSQL  │  │   Kafka/    │  │   Metrics   │
        │  Database   │  │   Redis     │  │  (Prom)     │
        └─────────────┘  │  Streams    │  └─────────────┘
                         └──────┬──────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
        ▼                       ▼                       ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ TRANSACTION      │  │  RISK ENGINE     │  │  ALERT SERVICE   │
│ INGEST SERVICE   │  │  SERVICE         │  │                  │
│                  │  │                  │  │                  │
│ • Generate/      │  │ • Consume Events │  │ • Consume Scored │
│   Receive Txns   │  │ • Feature        │  │   Transactions   │
│ • Validation     │──┼─→ Extraction     │──┼─→• Threshold     │
│ • Publish Events │  │ • ML Inference   │  │   Detection      │
│                  │  │ • Risk Scoring   │  │ • Alert Dispatch │
│                  │  │ • Publish Scores │  │ • Audit Logging  │
└──────────────────┘  └──────────────────┘  └──────────────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │  ML MODEL       │
                      │  (Inference)    │
                      │                 │
                      │ • Load Model    │
                      │ • Feature Vec   │
                      │ • Predict 0-1   │
                      │ • Scale to 0-100│
                      └─────────────────┘
```

---

## Service Responsibilities

### 1. Transaction Ingest Service

**Purpose**: Entry point for all financial transactions into the fraud detection pipeline

**Responsibilities**:
- Accept transaction events from external systems (or generate synthetic load)
- Perform initial validation (schema, required fields, data types)
- Assign transaction ID and timestamp if not present
- Publish validated transactions to `raw-transactions` stream
- Handle backpressure when downstream systems are saturated

**Key Design Decisions**:
- **Stateless**: No transaction state stored, purely a gateway
- **High Concurrency**: Worker pool pattern with goroutines to achieve 1000+ TPS
- **Fail Fast**: Invalid transactions rejected immediately with clear error codes

**Data Flow**:
```
External Source → Validate → Enrich → Publish to Kafka → ACK
```

---

### 2. Risk Engine Service

**Purpose**: Core fraud detection brain - scores every transaction for fraud risk

**Responsibilities**:
- Consume transactions from `raw-transactions` stream
- Extract features required by ML model:
  - Transaction amount (normalized)
  - Velocity metrics (transactions per user in time window)
  - Location deviation (distance from typical user locations)
  - Time anomaly score (unusual hour/day patterns)
  - Merchant category risk rating
- Invoke ML model for fraud probability prediction
- Convert probability (0-1) to risk score (0-100)
- Enrich transaction with risk score and feature values
- Publish to `scored-transactions` stream
- Store risk score in PostgreSQL for audit and analytics

**Key Design Decisions**:
- **Feature Store**: Maintain in-memory cache of user velocity and location patterns with TTL
- **Model Loading**: Load trained model artifact at startup, keep in memory
- **Idempotency**: Use transaction ID to detect and skip duplicate processing
- **Explicit Thresholds**: 
  - `LowRisk` = 0-40
  - `MediumRisk` = 41-74
  - `HighRisk` = 75-100

**Performance Considerations**:
- Model inference must complete in <10ms for throughput target
- Feature extraction parallelized where possible
- Database writes batched every 100ms to reduce I/O

---

### 3. Alert Service

**Purpose**: Detect high-risk transactions and trigger immediate alerts

**Responsibilities**:
- Consume transactions from `scored-transactions` stream
- Apply alerting rules:
  - Risk score ≥ 75 → immediate alert
  - Risk score ≥ 90 → critical priority alert
- Generate alert record with context (transaction details, risk factors)
- Dispatch alerts via:
  - Webhook to external fraud investigation system
  - Write to `alerts` table in PostgreSQL
  - Structured log for audit trail
- Track alert resolution status

**Key Design Decisions**:
- **Threshold-Based**: Clear, auditable rules (no ML black box in alerting)
- **Guaranteed Delivery**: Alert dispatch retries with exponential backoff
- **Deduplication**: Prevent alert spam from duplicate events using transaction ID

**Alert Schema**:
```
Alert {
  ID, TransactionID, RiskScore, Timestamp, 
  Priority (medium|high|critical),
  Status (open|investigating|resolved|false_positive),
  AssignedTo (optional)
}
```

---

### 4. API Gateway

**Purpose**: Unified REST API for frontend and external consumers

**Responsibilities**:
- **Authentication**: JWT token validation on all protected endpoints
- **Rate Limiting**: Per-user and per-IP limits to prevent abuse
- **Input Sanitization**: Validate and sanitize all incoming requests
- **Query Endpoints**:
  - `GET /transactions` - Paginated transaction history
  - `GET /transactions/:id` - Single transaction with risk details
  - `GET /alerts` - Active and historical alerts
  - `GET /metrics` - System health and statistics
- **Command Endpoints**:
  - `POST /auth/login` - User authentication
  - `POST /alerts/:id/resolve` - Mark alert as resolved
- **WebSocket**: Real-time push for live transaction feed

**Key Design Decisions**:
- **Middleware Chain**: Auth → Rate Limit → Validation → Handler → Error Handler
- **Pagination**: Cursor-based for large result sets
- **Response Times**: p95 < 100ms for read queries

---

## Data Flow

### Complete Transaction Pipeline

```
1. Transaction Created
   └─→ Ingest Service validates and publishes

2. Kafka: raw-transactions topic
   └─→ Risk Engine consumes

3. Risk Engine Processing:
   - Extract features from transaction
   - Query feature store for user patterns
   - Invoke ML model → fraud probability
   - Scale to risk score (0-100)
   - Write to PostgreSQL
   - Publish enriched event

4. Kafka: scored-transactions topic
   └─→ Alert Service consumes

5. Alert Service Processing:
   - Check if score ≥ 75
   - Generate alert record
   - Dispatch to webhook
   - Write to alerts table
   - Log audit trail

6. API Gateway serves:
   - Dashboard queries transactions table
   - WebSocket pushes new scored transactions
   - Alert panel queries alerts table
```

**Latency Budget**:
- Ingest → Kafka: <5ms
- Risk Engine processing: <15ms
- Alert Service: <10ms
- **End-to-end: <50ms (p95)**

---

## Database Schema (PostgreSQL)

### Design Principles
- **Normalized for Integrity**: Proper foreign keys and constraints
- **Indexed for Performance**: All query patterns have supporting indexes
- **Time-Series Optimized**: Partitioning by date for transactions table
- **Audit Trail**: All mutations tracked with timestamps

### Tables Overview

**1. transactions**
```sql
- id (UUID, primary key)
- user_id (integer, indexed)
- amount (decimal)
- currency (varchar)
- merchant_id (varchar, indexed)
- merchant_category (varchar)
- location_lat (decimal)
- location_lng (decimal)
- timestamp (timestamp, indexed)
- metadata (jsonb)
```
*Purpose*: Source of truth for all financial transactions

**2. risk_scores**
```sql
- id (serial, primary key)
- transaction_id (UUID, FK → transactions, unique)
- risk_score (integer 0-100)
- fraud_probability (decimal)
- feature_vector (jsonb)
- model_version (varchar)
- scored_at (timestamp)
```
*Purpose*: Audit trail of all risk assessments

**3. alerts**
```sql
- id (serial, primary key)
- transaction_id (UUID, FK → transactions)
- risk_score (integer)
- priority (enum: medium|high|critical)
- status (enum: open|investigating|resolved|false_positive)
- created_at (timestamp, indexed)
- resolved_at (timestamp, nullable)
- assigned_to (integer, FK → users, nullable)
- notes (text)
```
*Purpose*: Track fraud alerts and investigation workflow

**4. users**
```sql
- id (serial, primary key)
- email (varchar, unique)
- password_hash (varchar)
- full_name (varchar)
- role (enum: analyst|admin)
- created_at (timestamp)
- last_login (timestamp)
```
*Purpose*: Dashboard users (fraud analysts)

### Indexing Strategy

```sql
-- High-cardinality queries
CREATE INDEX idx_transactions_user_timestamp ON transactions(user_id, timestamp DESC);
CREATE INDEX idx_transactions_merchant ON transactions(merchant_id, timestamp DESC);

-- Alert dashboard queries
CREATE INDEX idx_alerts_status_created ON alerts(status, created_at DESC);

-- Time-range scans
CREATE INDEX idx_transactions_timestamp ON transactions(timestamp DESC);
```

---

## Event Streaming (Kafka)

### Topic Design

**1. raw-transactions**
- **Partitions**: 8 (allows parallel consumption)
- **Retention**: 7 days
- **Key**: user_id (ensures ordering per user)
- **Schema**:
  ```json
  {
    "transaction_id": "uuid",
    "user_id": 12345,
    "amount": 150.00,
    "currency": "USD",
    "merchant_id": "merchant_789",
    "merchant_category": "electronics",
    "location": {"lat": 40.7128, "lng": -74.0060},
    "timestamp": "2026-03-25T21:00:00Z"
  }
  ```

**2. scored-transactions**
- **Partitions**: 8
- **Retention**: 7 days
- **Key**: user_id
- **Schema**: raw-transaction + risk_score, fraud_probability, features

**3. alerts**
- **Partitions**: 4
- **Retention**: 30 days (compliance)
- **Key**: transaction_id

### Consumer Groups
- `risk-engine-consumers`: Parallel processing of raw transactions
- `alert-service-consumers`: Process scored transactions

---

## ML Model Architecture

### Model Type
**Logistic Regression** (initial version)
- Simple, interpretable, fast inference (<1ms)
- Upgradeable to XGBoost/Neural Network without changing API

### Features (5 core features)

1. **Transaction Amount (normalized)**
   - Log-scale normalization: `log(amount + 1) / log(max_amount)`
   - Rationale: Large amounts more suspicious

2. **Velocity Score**
   - Count of transactions by user in last 60 minutes
   - Formula: `min(count / 10, 1.0)` (capped at 10)
   - Rationale: Rapid successive transactions indicate fraud

3. **Location Deviation**
   - Distance from user's typical location (Haversine)
   - Normalized: `min(distance_km / 1000, 1.0)`
   - Rationale: Geographically distant transactions suspicious

4. **Time Anomaly Score**
   - Unusual hour (3am purchases) or day (holiday activity)
   - Binary: 1 if time is outside 8am-10pm or weekend
   - Rationale: Fraudsters act during off-hours

5. **Merchant Category Risk**
   - Pre-assigned risk rating per category
   - Electronics: 0.7, Groceries: 0.1, Wire Transfer: 0.9
   - Rationale: Domain knowledge of fraud-prone categories

### Training Data
- 100,000 synthetic transactions
- 5% labeled as fraud (realistic fraud rate)
- Features engineered from transaction attributes
- 80/20 train/test split

### Inference API
```go
type FraudScorer interface {
    PredictRisk(tx Transaction) (fraudProb float64, riskScore int, err error)
}
```

### Model Versioning
- Models stored as artifacts: `ml/model/fraud_model_v1.pkl`
- Version tracked in risk_scores table
- Allows A/B testing and rollback

---

## Security Architecture

### Authentication (JWT)
- **Token Structure**: Header.Payload.Signature
- **Claims**: user_id, email, role, exp (expiry)
- **Secret**: Environment variable (rotated monthly)
- **Expiry**: 1 hour (short-lived for security)
- **Refresh**: Separate refresh token with 7-day expiry

### Authorization
- **Role-Based Access Control (RBAC)**:
  - `analyst`: Read transactions/alerts, resolve alerts
  - `admin`: Full access + user management

### Rate Limiting
- **Algorithm**: Token Bucket
- **Limits**:
  - Per user: 100 requests/minute
  - Per IP: 1000 requests/minute (prevents DDoS)
- **Response**: 429 Too Many Requests with Retry-After header

### Input Validation
- **All Inputs Sanitized**: SQL injection, XSS prevention
- **Type Validation**: Strong typing + runtime checks
- **Range Checks**: Transaction amount must be > 0 and < $1M

---

## Observability

### Structured Logging
- **Format**: JSON lines (parseable by log aggregators)
- **Fields**: timestamp, level, service, request_id, message, context
- **Libraries**: zerolog (Go), pino (Next.js)
- **Levels**: DEBUG (dev only), INFO, WARN, ERROR, FATAL

**Example**:
```json
{
  "time": "2026-03-25T21:00:00Z",
  "level": "info",
  "service": "risk-engine",
  "request_id": "abc123",
  "transaction_id": "txn_456",
  "risk_score": 82,
  "processing_time_ms": 12,
  "message": "transaction scored"
}
```

### Metrics (Prometheus)
- **Request Rate**: Requests per second per service
- **Latency**: p50, p95, p99 processing times
- **Error Rate**: Errors per second + error types
- **Queue Depth**: Kafka consumer lag
- **Database**: Query times, connection pool utilization

### Alerts (via metrics)
- Risk engine processing time p95 > 20ms
- Alert service consumer lag > 1000 messages
- API gateway error rate > 1%

---

## Scalability & Performance

### Target Performance
- **Throughput**: 1000+ transactions/second
- **Latency**: p95 end-to-end < 50ms
- **Availability**: 99.9% uptime

### Horizontal Scaling Strategy

**Stateless Services** (can scale linearly):
- Transaction Ingest: Add instances, load balance
- Risk Engine: Add consumer instances (Kafka rebalances)
- Alert Service: Add consumer instances
- API Gateway: Add instances behind load balancer

**Stateful Components**:
- PostgreSQL: Read replicas for queries, write to primary
- Kafka: Add partitions and brokers as needed

### Concurrency Patterns (Go)

**Worker Pool Pattern**:
```go
// Transaction Ingest Service
for i := 0; i < NumWorkers; i++ {
    go worker(transactionChan)
}
```

**Graceful Shutdown**:
```go
// Ensure in-flight transactions complete before shutdown
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### Database Optimization
- **Connection Pooling**: Max 50 connections per service
- **Prepared Statements**: Reduce query parse time
- **Batch Writes**: Commit every 100ms or 500 records
- **Partitioning**: Transactions table partitioned monthly

---

## Failure Modes & Mitigation

| Component | Failure Mode | Impact | Mitigation |
|-----------|--------------|--------|------------|
| Kafka Down | No event flow | All processing stops | Dead letter queue, service retries |
| PostgreSQL Down | No persistence | Read-only mode, alerts fail | Read replica failover, circuit breaker |
| Risk Engine Crash | No scoring | Transactions pile in queue | Auto-restart, consumer group rebalancing |
| ML Model Error | Scoring fails | Alert on model errors, default to high risk | Fallback to rule-based scoring |
| API Gateway Down | Dashboard unavailable | Frontend can't query data | Multiple gateway instances, load balancer |

### Circuit Breaker Pattern
- After 5 consecutive failures, open circuit for 30 seconds
- Prevents cascading failures across services

### Idempotency
- All event consumers use transaction ID to detect duplicates
- Database upserts instead of inserts where applicable

---

## Technology Trade-offs

### Why Kafka over RabbitMQ?
- **Higher throughput**: Kafka designed for log streaming at scale
- **Persistence**: Messages stored on disk, not just in-memory
- **Replay**: Can reprocess events by resetting consumer offset
- **Drawback**: More complex to operate

### Why PostgreSQL over NoSQL?
- **ACID guarantees**: Financial data requires consistency
- **Relational queries**: Complex joins for analytics
- **Mature tooling**: Battle-tested in fintech
- **Drawback**: Harder to horizontally scale writes

### Why Logistic Regression initially?
- **Interpretability**: Can explain why transaction was flagged
- **Speed**: <1ms inference time
- **Simplicity**: Easy to debug and validate
- **Upgrade path**: Can swap to XGBoost/NN without changing API

---

## Deployment Architecture (Local Development)

```
Docker Compose Stack:
├── PostgreSQL (port 5432)
├── Kafka + Zookeeper (port 9092)
├── Transaction Ingest Service (port 8001)
├── Risk Engine Service (port 8002)
├── Alert Service (port 8003)
├── API Gateway (port 8000)
└── Next.js Dashboard (port 3000)
```

**Service Dependencies**:
- All services wait for PostgreSQL healthy
- Event-driven services wait for Kafka healthy
- Frontend waits for API Gateway healthy

---

## Future Enhancements (Out of Scope v1)

1. **Graph-based fraud detection**: Analyze networks of connected accounts
2. **Real-time feature engineering**: Apache Flink for complex aggregations
3. **Model retraining pipeline**: Automated retraining on new fraud patterns
4. **Multi-region deployment**: Geo-distributed for global transactions
5. **Advanced alerting**: Slack/PagerDuty integration

---

## Conclusion

This architecture balances **simplicity with production-readiness**. Every design decision prioritizes:

1. **Correctness**: Financial data integrity is non-negotiable
2. **Observability**: Every operation is traceable
3. **Performance**: Meets real-world bank throughput requirements
4. **Maintainability**: Clear boundaries, explicit error handling

The system is designed to be **explainable in a technical review** and **deployable in a regulated environment**.


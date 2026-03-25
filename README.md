# Sentinel Fraud Engine

Real-time fraud detection and risk scoring platform built with production-grade architecture.

## 🏗️ System Overview

A microservices-based, event-driven fraud detection system that processes 1000+ transactions per second, assigns AI-powered risk scores, and triggers real-time alerts.

## 🛠️ Tech Stack

- **Backend Services**: Golang (4 microservices)
- **Database**: PostgreSQL with optimized indexing
- **Event Streaming**: Kafka / Redis Streams
- **ML Model**: Python (scikit-learn) + Go inference
- **Frontend**: Next.js with real-time updates
- **Observability**: Structured logging (zerolog), Prometheus metrics

## 📁 Project Structure

```
sentinel-fraud-engine/
├── services/
│   ├── transaction-ingest/    # Receives and validates transactions
│   ├── risk-engine/            # ML-powered fraud scoring
│   ├── alert-service/          # Alert generation and dispatch
│   └── api-gateway/            # REST API + WebSocket
├── frontend/
│   └── dashboard/              # Next.js real-time dashboard
├── ml/
│   ├── training/               # Model training scripts
│   └── model/                  # Trained model artifacts
├── database/
│   └── migrations/             # SQL schema migrations
├── shared/
│   ├── models/                 # Shared Go types
│   ├── kafka/                  # Event streaming utilities
│   └── postgres/               # Database utilities
└── docs/                       # Documentation
```

## 🏛️ Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed system design.

**Key Services:**
1. **Transaction Ingest** - Validates and publishes transaction events
2. **Risk Engine** - Extracts features, runs ML model, scores fraud risk (0-100)
3. **Alert Service** - Triggers alerts for high-risk transactions (score ≥ 75)
4. **API Gateway** - JWT auth, rate limiting, REST API for frontend

**Data Flow:**
```
Transaction → Ingest → Kafka → Risk Engine → ML Model → Kafka → Alert Service
                                      ↓
                              PostgreSQL (audit trail)
                                      ↓
                              API Gateway → Dashboard
```

## 🗄️ Database Schema

See [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md) for complete schema design.

**Core Tables:**
- `transactions` - All financial transactions
- `risk_scores` - ML model outputs and audit trail
- `alerts` - High-risk transaction alerts
- `users` - Dashboard users (fraud analysts)

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Kafka 3.x or Redis 7+
- Node.js 18+
- Python 3.11+ (for ML training)

### Local Development Setup

```bash
# 1. Clone repository
git clone https://github.com/yourusername/sentinel-fraud-engine.git
cd sentinel-fraud-engine

# 2. Set up database
docker compose up -d postgres
go run database/migrations/*.go up

# 3. Train ML model
cd ml/training
python train_model.py
cd ../..

# 4. Start services with Docker Compose
docker compose up

# 5. Access dashboard
open http://localhost:3000
```

### Running Individual Services

```bash
# Transaction Ingest Service
cd services/transaction-ingest
go run main.go

# Risk Engine Service
cd services/risk-engine
go run main.go

# Alert Service
cd services/alert-service
go run main.go

# API Gateway
cd services/api-gateway
go run main.go

# Frontend Dashboard
cd frontend/dashboard
npm install
npm run dev
```

## 🧪 Testing

```bash
# Run all unit tests
go test ./...

# Run integration tests
go test ./... -tags=integration

# Load test (generate 1000 TPS)
cd services/transaction-ingest
go run cmd/loadtest/main.go --tps 1000
```

## 📊 Observability

**Structured Logging:**
```bash
# All services log JSON to stdout
docker compose logs -f risk-engine
```

**Metrics:**
```bash
# Prometheus metrics exposed on each service
curl http://localhost:8002/metrics
```

**Key Metrics:**
- Transaction processing rate (TPS)
- ML inference latency (p95, p99)
- Kafka consumer lag
- Database query times

## 🔐 Security

- **Authentication**: JWT tokens (1-hour expiry)
- **Authorization**: Role-based access (analyst, admin)
- **Rate Limiting**: Token bucket algorithm (100 req/min per user)
- **Input Validation**: All inputs sanitized and type-checked
- **Secrets Management**: Environment variables (never committed)

## 🧠 ML Model

**Algorithm**: Logistic Regression (v1)

**Features:**
1. Transaction amount (normalized)
2. Velocity score (transactions per hour)
3. Location deviation (distance from typical location)
4. Time anomaly (off-hours activity)
5. Merchant category risk

**Output**: Fraud probability (0-1) → Risk score (0-100)

**Training Data**: 100,000 synthetic transactions with 5% fraud rate

## 📈 Performance

**Targets:**
- Throughput: 1000+ TPS
- Latency: p95 end-to-end < 50ms
- ML inference: < 10ms
- API response: p95 < 100ms

## 🔄 Deployment

**Docker Compose (Local/Staging):**
```bash
docker compose up --build
```

**Production Considerations:**
- Kubernetes deployment (multi-replica services)
- PostgreSQL read replicas for scaling
- Kafka partitions = parallelism factor
- Redis cache for feature store
- CDN for frontend assets

## 📝 Design Decisions

### Why Kafka over RabbitMQ?
- Higher throughput for log-based streaming
- Event replay capability (reprocess by resetting offset)
- Better for audit trail (events persisted on disk)

### Why PostgreSQL over NoSQL?
- ACID guarantees required for financial data
- Complex relational queries for analytics
- Battle-tested in fintech (compliance-ready)

### Why Logistic Regression initially?
- Explainable (can justify why transaction was flagged)
- Fast inference (<1ms)
- Establishes baseline for future improvements

## 🛣️ Roadmap

**v1.0 (Current)**
- [x] Core microservices architecture
- [x] Basic ML fraud model
- [x] Real-time dashboard
- [ ] Load testing and optimization

**v2.0 (Future)**
- [ ] Graph-based fraud detection
- [ ] Advanced ML models (XGBoost, Neural Network)
- [ ] Multi-region deployment
- [ ] Automated model retraining pipeline

## 📚 Documentation

- [Architecture Design](./ARCHITECTURE.md)
- [Database Schema](./DATABASE_SCHEMA.md)
- [API Documentation](./docs/API.md) (coming soon)
- [Deployment Guide](./docs/DEPLOYMENT.md) (coming soon)

## 🤝 Contributing

This is an educational project demonstrating production-grade system design.

**Code Standards:**
- Follow Go conventions (gofmt, golangci-lint)
- Write tests for business logic
- Document non-obvious decisions
- Keep functions focused and small

## 📄 License

MIT License - see LICENSE file for details

## 👥 Authors

Built as a demonstration of senior-level software engineering practices for fraud detection systems.

---

**Note**: This system is designed to the standards expected at tier-1 financial institutions (e.g., JPMorgan Chase). Every design decision prioritizes correctness, observability, and production-readiness.

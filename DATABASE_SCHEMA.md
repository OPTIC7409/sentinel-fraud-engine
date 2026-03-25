# Sentinel Fraud Engine - Database Schema Design

## Schema Philosophy

This database schema is designed for a **high-write, high-read financial system** with the following priorities:

1. **Data Integrity**: ACID compliance, foreign keys, constraints
2. **Query Performance**: Strategic indexing for all access patterns  
3. **Audit Trail**: Immutable transaction records, full timestamp tracking
4. **Scalability**: Partitioning strategy for time-series data

---

## Schema Diagram

```
┌─────────────┐         ┌──────────────────┐         ┌─────────────┐
│   users     │         │   transactions   │         │ risk_scores │
├─────────────┤         ├──────────────────┤         ├─────────────┤
│ id (PK)     │         │ id (PK)          │◄────────│ id (PK)     │
│ email       │         │ user_id (FK)     │         │ txn_id (FK) │
│ password_   │         │ amount           │         │ risk_score  │
│  hash       │         │ currency         │         │ fraud_prob  │
│ full_name   │         │ merchant_id      │         │ features    │
│ role        │         │ merchant_cat     │         │ model_ver   │
│ created_at  │         │ location_lat     │         │ scored_at   │
│ last_login  │         │ location_lng     │         └─────────────┘
└─────────────┘         │ timestamp        │
      ▲                 │ metadata         │
      │                 │ created_at       │
      │                 └──────────────────┘
      │                          ▲
      │                          │
      │                 ┌────────────────┐
      │                 │    alerts      │
      │                 ├────────────────┤
      │                 │ id (PK)        │
      │                 │ txn_id (FK)    │
      │                 │ risk_score     │
      └─────────────────│ assigned_to(FK)│
                        │ priority       │
                        │ status         │
                        │ created_at     │
                        │ resolved_at    │
                        │ notes          │
                        └────────────────┘
```

---

## Table Definitions

### 1. users

**Purpose**: Dashboard users (fraud analysts and admins)

**Schema**:
```sql
CREATE TABLE users (
    id              SERIAL PRIMARY KEY,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL CHECK (role IN ('analyst', 'admin')),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login      TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
```

**Design Decisions**:
- `email` is unique constraint for login
- `password_hash` stores bcrypt hash (never plain text)
- `role` is enum-like CHECK constraint (not separate roles table - YAGNI)
- `last_login` tracks user activity for audit

**Sample Data**:
```sql
INSERT INTO users (email, password_hash, full_name, role) VALUES
('alice@sentinel.com', '$2a$10$...', 'Alice Johnson', 'admin'),
('bob@sentinel.com', '$2a$10$...', 'Bob Smith', 'analyst');
```

---

### 2. transactions

**Purpose**: Immutable record of all financial transactions entering the system

**Schema**:
```sql
CREATE TABLE transactions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             INTEGER NOT NULL,
    amount              NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    merchant_id         VARCHAR(100) NOT NULL,
    merchant_category   VARCHAR(50) NOT NULL,
    location_lat        NUMERIC(9, 6),
    location_lng        NUMERIC(9, 6),
    timestamp           TIMESTAMP NOT NULL,
    metadata            JSONB,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX idx_transactions_user_timestamp ON transactions(user_id, timestamp DESC);
CREATE INDEX idx_transactions_merchant ON transactions(merchant_id, timestamp DESC);
CREATE INDEX idx_transactions_timestamp ON transactions(timestamp DESC);
CREATE INDEX idx_transactions_merchant_category ON transactions(merchant_category);

-- Partitioning (for high volume, partition by month)
-- This will be implemented in migrations as native partitioning
```

**Design Decisions**:
- `id` is UUID for distributed generation (services can generate IDs independently)
- `amount` is NUMERIC for exact decimal precision (never use FLOAT for money)
- `timestamp` is the transaction's business timestamp (when it occurred)
- `created_at` is when we ingested it (for debugging clock skew)
- `metadata` is JSONB for extensibility without schema changes (e.g, device fingerprint)
- No `user_id` foreign key to users table - users here are customers, not dashboard users

**Access Patterns**:
1. "Get recent transactions for user X": `WHERE user_id = ? ORDER BY timestamp DESC LIMIT 100`
2. "Get all transactions at merchant Y": `WHERE merchant_id = ? AND timestamp > ?`
3. "Get high-value transactions today": `WHERE amount > 10000 AND timestamp >= CURRENT_DATE`

**Partitioning Strategy** (for scaling):
```sql
-- Partition by month for time-series queries
CREATE TABLE transactions (
    -- columns as above
) PARTITION BY RANGE (timestamp);

CREATE TABLE transactions_2026_03 PARTITION OF transactions
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE TABLE transactions_2026_04 PARTITION OF transactions
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
-- etc.
```

**Sample Data**:
```sql
INSERT INTO transactions VALUES
('a1b2c3d4...', 12345, 150.00, 'USD', 'merchant_amazon', 'electronics', 
 40.7128, -74.0060, '2026-03-25 14:30:00', '{"device": "iPhone"}', NOW());
```

---

### 3. risk_scores

**Purpose**: Audit trail of all fraud risk assessments. One-to-one with transactions.

How `risk_score` and `fraud_probability` are produced (feature engineering, logistic regression, and rounding) is documented in [ARCHITECTURE.md](./ARCHITECTURE.md) in **How the risk score is computed (mathematics and ML)**.

**Schema**:
```sql
CREATE TABLE risk_scores (
    id                  SERIAL PRIMARY KEY,
    transaction_id      UUID NOT NULL UNIQUE REFERENCES transactions(id) ON DELETE CASCADE,
    risk_score          INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    fraud_probability   NUMERIC(5, 4) NOT NULL CHECK (fraud_probability >= 0 AND fraud_probability <= 1),
    feature_vector      JSONB NOT NULL,
    model_version       VARCHAR(50) NOT NULL,
    processing_time_ms  INTEGER,
    scored_at           TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_risk_scores_transaction ON risk_scores(transaction_id);
CREATE INDEX idx_risk_scores_score ON risk_scores(risk_score DESC);
CREATE INDEX idx_risk_scores_scored_at ON risk_scores(scored_at DESC);
```

**Design Decisions**:
- `transaction_id` is UNIQUE - each transaction scored exactly once
- `risk_score` is 0-100 integer for display (derived from fraud_probability)
- `fraud_probability` is the raw ML model output (0.0 to 1.0)
- `feature_vector` stores inputs to model as JSON for debugging:
  ```json
  {
    "amount_normalized": 0.23,
    "velocity_score": 0.8,
    "location_deviation": 0.15,
    "time_anomaly": 0,
    "merchant_category_risk": 0.7
  }
  ```
- `model_version` enables A/B testing and rollback (e.g., "v1.0.0", "v1.1.0")
- `processing_time_ms` tracks inference latency for performance monitoring
- CASCADE delete if transaction deleted (but transactions should never be deleted)

**Access Patterns**:
1. "Get risk score for transaction X": `WHERE transaction_id = ?`
2. "Get all high-risk transactions": `WHERE risk_score >= 75 ORDER BY scored_at DESC`
3. "Analyse model performance": `SELECT model_version, AVG(risk_score), COUNT(*) GROUP BY model_version`

**Sample Data**:
```sql
INSERT INTO risk_scores VALUES
(1, 'a1b2c3d4...', 82, 0.8234, 
 '{"amount_normalized": 0.6, "velocity_score": 0.9, ...}',
 'v1.0.0', 12, NOW());
```

---

### 4. alerts

**Purpose**: Track fraud alerts triggered by high-risk transactions

**Schema**:
```sql
CREATE TYPE alert_priority AS ENUM ('medium', 'high', 'critical');
CREATE TYPE alert_status AS ENUM ('open', 'investigating', 'resolved', 'false_positive');

CREATE TABLE alerts (
    id              SERIAL PRIMARY KEY,
    transaction_id  UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    risk_score      INTEGER NOT NULL,
    priority        alert_priority NOT NULL,
    status          alert_status NOT NULL DEFAULT 'open',
    assigned_to     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMP,
    notes           TEXT,
    CONSTRAINT check_resolved CHECK (
        (status IN ('resolved', 'false_positive') AND resolved_at IS NOT NULL) OR
        (status IN ('open', 'investigating') AND resolved_at IS NULL)
    )
);

CREATE INDEX idx_alerts_status_created ON alerts(status, created_at DESC);
CREATE INDEX idx_alerts_assigned ON alerts(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_alerts_transaction ON alerts(transaction_id);
CREATE INDEX idx_alerts_priority ON alerts(priority, created_at DESC);
```

**Design Decisions**:
- **Priority levels** (auto-assigned based on risk_score):
  - 75-84: `medium`
  - 85-94: `high`
  - 95-100: `critical`
- `status` workflow: `open` → `investigating` → `resolved`/`false_positive`
- `assigned_to` is nullable (alerts start unassigned)
- `resolved_at` is NULL until status changes to resolved/false_positive (enforced by CHECK)
- No foreign key to transaction (allows deleting transactions without cascade)
- Actually, YES foreign key - we want referential integrity in fraud system

**Access Patterns**:
1. "Get open alerts": `WHERE status = 'open' ORDER BY priority DESC, created_at`
2. "Get my assigned alerts": `WHERE assigned_to = ? AND status != 'resolved'`
3. "Get critical alerts in last hour": `WHERE priority = 'critical' AND created_at > NOW() - INTERVAL '1 hour'`
4. "Alert resolution metrics": `SELECT AVG(EXTRACT(EPOCH FROM (resolved_at - created_at))) WHERE status = 'resolved'`

**Sample Data**:
```sql
INSERT INTO alerts VALUES
(1, 'a1b2c3d4...', 82, 'high', 'investigating', 2, NOW(), NULL, 'Reviewing transaction pattern');
```

---

## Derived Views (for API efficiency)

### transaction_with_risk

**Purpose**: Join transactions and risk_scores for common API queries

```sql
CREATE VIEW transaction_with_risk AS
SELECT 
    t.id,
    t.user_id,
    t.amount,
    t.currency,
    t.merchant_id,
    t.merchant_category,
    t.timestamp,
    r.risk_score,
    r.fraud_probability,
    r.model_version,
    r.scored_at
FROM transactions t
LEFT JOIN risk_scores r ON t.id = r.transaction_id;
```

**Usage**: `SELECT * FROM transaction_with_risk WHERE risk_score > 75 LIMIT 100`

---

## Indexing Strategy Summary

| Table | Index | Purpose | Type |
|-------|-------|---------|------|
| users | email | Login lookup | Unique |
| transactions | (user_id, timestamp) | User transaction history | B-tree composite |
| transactions | timestamp | Time-range scans | B-tree |
| transactions | merchant_id | Merchant analysis | B-tree |
| risk_scores | transaction_id | Score lookup | Unique |
| risk_scores | risk_score DESC | High-risk queries | B-tree |
| alerts | (status, created_at) | Alert dashboard | B-tree composite |
| alerts | assigned_to | Analyst workload | B-tree partial (WHERE assigned_to IS NOT NULL) |

**Why these indexes?**
- **Composite indexes** match WHERE + ORDER BY patterns (user_id + timestamp)
- **Partial indexes** on alerts reduce index size (only index assigned alerts)
- **DESC indexes** optimise ORDER BY DESC queries (most recent first)

---

## Data Retention & Archival

### Hot Data (PostgreSQL)
- **Transactions**: Last 3 months in primary tables
- **Risk Scores**: Last 3 months
- **Alerts**: Last 6 months

### Cold Data (Archive)
- **Transactions**: Older than 3 months → S3/data warehouse
- **Compliance**: Keep audit trail for 7 years (regulatory requirement)

### Archival Strategy
```sql
-- Monthly batch job
INSERT INTO transactions_archive 
SELECT * FROM transactions WHERE timestamp < NOW() - INTERVAL '3 months';

DELETE FROM transactions WHERE timestamp < NOW() - INTERVAL '3 months';
```

---

## Performance Targets

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Transaction INSERT | <5ms p95 | Single row |
| Risk score INSERT | <10ms p95 | Single row |
| Alert query (dashboard) | <50ms p95 | 100 rows |
| Transaction history query | <100ms p95 | 1000 rows |
| Batch INSERT (100 txns) | <50ms p95 | Using COPY or multi-value INSERT |

---

## Migration Strategy

### Version Control
- Use `golang-migrate` for schema versioning
- Each migration is two files: `NNN_description.up.sql` and `NNN_description.down.sql`

### Migration Order
1. `001_create_users.up.sql`
2. `002_create_transactions.up.sql`
3. `003_create_risk_scores.up.sql`
4. `004_create_alerts.up.sql`
5. `005_create_indexes.up.sql`
6. `006_create_views.up.sql`

### Zero-Downtime Migrations
- **Add columns**: Safe (default values)
- **Drop columns**: Dangerous (requires multi-phase deployment)
- **Change types**: Requires new column + backfill + drop old

---

## Consistency & Constraints

### Foreign Key Enforcement
- All relationships use FK constraints (data integrity > performance)
- `ON DELETE CASCADE` where child is meaningless without parent (risk_scores)
- `ON DELETE SET NULL` where orphans acceptable (alerts.assigned_to)

### CHECK Constraints
- `amount > 0` - prevent negative transactions
- `risk_score BETWEEN 0 AND 100` - enforce valid range
- `fraud_probability BETWEEN 0 AND 1` - ML output validation
- `resolved_at` logic - prevent invalid state combinations

### NOT NULL Enforcement
- All business-critical fields are NOT NULL
- Nullable only when truly optional (assigned_to, resolved_at, notes)

---

## Database Configuration

### Connection Pool Settings
```ini
# Per service
max_connections = 50
min_idle_connections = 10
max_connection_lifetime = 5m
connection_timeout = 10s
```

### Performance Tuning (PostgreSQL)
```ini
# For write-heavy workload
shared_buffers = 4GB
effective_cache_size = 12GB
maintenance_work_mem = 1GB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1  # SSD assumed
```

---

## Backup & Recovery

### Backup Strategy
- **Continuous archiving**: WAL (Write-Ahead Log) streaming to S3
- **Daily snapshots**: Full database backup at 2am UTC
- **Retention**: 30 days of daily backups

### Recovery Time Objective (RTO)
- Target: <15 minutes to restore from snapshot
- Critical: Transactions and risk_scores tables

### Point-in-Time Recovery (PITR)
- Can restore to any point within last 30 days
- Required for fraud investigation audits

---

## Security

### Row-Level Security (RLS)
Not implemented in v1, but future consideration:
```sql
-- Analysts can only see alerts assigned to them
CREATE POLICY analyst_alerts ON alerts
    FOR SELECT
    USING (assigned_to = current_user_id() OR role = 'admin');
```

### Encryption
- **At rest**: PostgreSQL transparent data encryption (TDE)
- **In transit**: TLS 1.3 for all connections
- **Password hashing**: bcrypt with cost factor 12

### Audit Logging
- Enable PostgreSQL audit extension (`pgaudit`)
- Log all DDL changes, DELETE operations, and user auth

---

## Testing Data

### Seed Data Script
```sql
-- Create test users
INSERT INTO users (email, password_hash, full_name, role) VALUES
('admin@test.com', '$2a$12$...', 'Admin User', 'admin'),
('analyst@test.com', '$2a$12$...', 'Test Analyst', 'analyst');

-- Create sample transactions (1000 rows)
INSERT INTO transactions (user_id, amount, currency, merchant_id, merchant_category, timestamp)
SELECT 
    floor(random() * 10000)::int,
    (random() * 1000)::numeric(12,2),
    'USD',
    'merchant_' || floor(random() * 100),
    (ARRAY['groceries', 'electronics', 'travel', 'restaurants'])[floor(random() * 4 + 1)],
    NOW() - (random() * INTERVAL '30 days')
FROM generate_series(1, 1000);
```

---

## Conclusion

This schema is designed for **correctness, performance, and observability**:

- **Normalised** - No redundancy, clear relationships  
- **Indexed** - Query patterns covered  
- **Constrained** - Invalid states blocked at DB level  
- **Auditable** - Timestamps and foreign keys throughout  
- **Scalable** - Partitioning when volume warrants it  

Every decision prioritises **production-readiness over convenience**.

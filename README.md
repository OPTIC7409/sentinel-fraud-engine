# Sentinel Fraud Engine

A sample fraud detection and risk scoring platform: event-driven microservices, PostgreSQL for durable state, Kafka between services, and a small Next.js dashboard for analysts. I built it to show how I think about correctness, operability, and the kind of constraints you care about in banking technology (auditability, clear failure modes, honest metrics).

If you are hiring for a backend or full-stack role in financial services, the sections on architecture, how to run it, and **challenges I faced** are the ones worth reading first.

## What this is

Transactions flow through ingest, scoring, alerting, and persistence. The risk engine applies a trained model (logistic regression in v1) and writes scores to Postgres. High scores surface as alerts. The API gateway exposes authenticated HTTP APIs; the dashboard polls for updates (no WebSocket on the gateway today).

Design goals: explainable scoring, structured logs, metrics hooks, and a schema that supports investigation and reporting, not just real-time display.

## Tech stack

- Go: four services (ingest, risk engine, alerts, API gateway)
- PostgreSQL 15 (migrations under `database/migrations/`)
- Kafka (Confluent images in Compose) for decoupling producers and consumers
- Python / scikit-learn for training; inference path integrated in the risk engine
- Next.js dashboard under `frontend/dashboard/`
- Zerolog-style JSON logging; services expose Prometheus-style endpoints where implemented

## Repository layout

```
sentinel-fraud-engine/
├── services/
│   ├── transaction-ingest/
│   ├── risk-engine/
│   ├── alert-service/
│   └── api-gateway/
├── frontend/dashboard/
├── ml/training/          model training
├── ml/model/             serialised artifacts
├── database/migrations/  SQL up/down
├── tools/loadtest/       Kafka producer load tool
├── shared/               Go packages shared by services
└── docs/                 extra design notes
```

More detail: [ARCHITECTURE.md](./ARCHITECTURE.md), [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md).

## Architecture (short)

1. **Transaction ingest** validates and publishes to Kafka.
2. **Risk engine** consumes, scores (model + features), writes to Postgres, publishes scored events.
3. **Alert service** consumes scored events and raises alerts when policy says so (for example high risk score).
4. **API gateway** issues JWTs, rate limits, and serves REST for the dashboard and automation.

Data flow in one line: ingest to Kafka, risk engine to Postgres and Kafka, alerts downstream, gateway reads Postgres for the UI.

## Getting started

**Prerequisites:** Docker and Docker Compose, Go 1.21+ for local tooling, Node 18+ for the dashboard. Python 3.11+ if you retrain the model.

**Quick path (recommended):**

```bash
git clone <your-fork-or-url> sentinel-fraud-engine
cd sentinel-fraud-engine
./start.sh
curl -s http://localhost:8000/health
```

`start.sh` builds images, starts Postgres (host port **5433** mapped to container 5432), Zookeeper/Kafka, runs migrations, seeds users, then starts all application containers.

**Dashboard:**

```bash
cd frontend/dashboard
cp .env.example .env.local   # optional; default API is http://localhost:8000
npm install
npm run dev
```

Open http://localhost:3000, sign in. Seeded analyst: `analyst@sentinel.com` / `sentinel123` (see `database/seeds/001_users.sql`). Full dashboard notes: [frontend/dashboard/README.md](./frontend/dashboard/README.md).

**Manual service startup (no app containers):**

```bash
docker compose up -d postgres zookeeper kafka
cd database && go run migrate.go up && cd ..
docker exec -i sentinel-postgres psql -U postgres -d sentinel_fraud < database/seeds/001_users.sql
# then run each service with go run from its directory, with env vars for Kafka and DATABASE_URL
```

## API smoke tests

```bash
curl -s http://localhost:8000/health

curl -s -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"analyst@sentinel.com","password":"sentinel123"}'

# use the token from the response:
curl -s "http://localhost:8000/api/transactions?limit=10" \
  -H "Authorization: Bearer <token>"

curl -s "http://localhost:8000/api/alerts?status=open" \
  -H "Authorization: Bearer <token>"

curl -s http://localhost:8000/api/metrics \
  -H "Authorization: Bearer <token>"
```

## Testing and load generation

```bash
go test ./...
```

There are few tests today; the fraud model package has coverage. Integration-tagged tests are not wired up yet (`go test -tags=integration` is a placeholder for pipeline work).

**Load tool (publishes synthetic messages to Kafka):**

```bash
go run ./tools/loadtest/main.go --tps 1000 --duration 60 --brokers localhost:9092
```

### Sample load run (local Docker)

One run on my machine with `--tps 1000 --duration 60` and Kafka on `localhost:9092`:

| Metric | Value |
|--------|------:|
| Duration | 60 s |
| Publishes succeeded | 4,499 |
| Publish errors | 1 |
| Publish success rate | 99.98% |
| Measured producer TPS | ~75 |
| Requested TPS flag | 1,000 |

That success rate is **Kafka client and broker reliability** for the producer, not classifier precision or recall. Achieved TPS is dominated by a single-process ticker on a laptop, not by the theoretical ceiling of the services.

## Observability

```bash
docker compose logs -f risk-engine
curl -s http://localhost:8002/metrics   # example; port depends on service exposure
```

Useful signals to watch in a real deployment: consumer lag, scoring latency, error rates on ingest, and DB connection health.

## Security (intentional limits of this repo)

- JWT auth on protected routes (secret from env in Compose; rotate for anything beyond local demo).
- Rate limiting on the gateway (token bucket).
- Passwords in seeds are bcrypt; the demo login handler still issues tokens without hitting the users table (documented in code as a shortcut). In a bank interview I would say: replace with real credential verification and audit login events.

## ML model (v1)

Logistic regression on synthetic data. The risk engine builds five numeric features in Go (amount, velocity, location deviation, time flag, merchant-category prior), passes them to a trained **sklearn** model in Python, gets **P(fraud)**, then sets **risk score = round(100 × P(fraud))** clamped to 0–100. Exact formulas, the sigmoid, and training hyperparameters are in [ARCHITECTURE.md](./ARCHITECTURE.md) under **How the risk score is computed (mathematics and ML)**.

The small `ml/model/fraud_model_v1.0.0.joblib` is in git so Docker and local runs work after clone. Large CSVs under `ml/training/data/` stay out of the repo (`.gitignore`); regenerate with `python ml/training/generate_data.py` then `python ml/training/train_model.py` from a venv with `pandas`, `scikit-learn`, `joblib` installed.

## Performance targets (design intent)

Throughput in the 1k TPS class, low scoring latency, API p95 targets in the README of a typical internal tool. The numbers above separate **what I measured on a laptop** from **what the architecture is aiming for** under proper hardware and tuning.

## Challenges I faced and how I resolved them

These are real issues I hit while making the stack pleasant to run on a developer machine and from the browser. I am including them because they mirror the kind of debugging you do in production engineering: follow the failure, fix the root cause, make it hard to repeat the mistake.

**1. Postgres port collision.** Another project already owned `localhost:5432`. Binding Sentinel’s container to the same host port failed. I mapped Postgres to **5433** on the host while keeping `postgres:5432` inside the Docker network so service env vars stayed unchanged. I aligned the Go migration default and local dev defaults with the host port so `start.sh` could run migrations from the laptop against the published port.

**2. Migrations appeared to run but created nothing.** `start.sh` runs `go run migrate.go` from inside `database/`, while the migrator looked for `database/migrations` relative to the current working directory. That path did not exist from that cwd, so the tool logged “no migration files” and exited zero. Seeds then failed (`users` missing), and because the script uses `set -e`, **application containers never started**. The dashboard showed “Failed to fetch” simply because nothing was listening on port 8000. I fixed the migrator to resolve the migrations directory **next to the source file** and to **fail hard** if no migration files match, so silent no-ops cannot happen again.

**3. Repeat runs against a dirty local database.** After partial applies, rerunning SQL could fail on “index already exists”. I made indexes **idempotent** (`CREATE INDEX IF NOT EXISTS`) and wrapped enum creation in exception-safe blocks where Postgres requires it, so recovery on a shared laptop does not mean wiping the volume every time.

**4. A bad comment in SQL blocked migration 004.** I had used `COMMENT ON COLUMN` for something that was a **table constraint**, not a column. Postgres correctly rejected it. Switching to `COMMENT ON CONSTRAINT` fixed the migration.

**5. Browser login blocked by CORS.** The dashboard on port 3000 calls the API on port 8000. Browsers send an **OPTIONS** preflight. Gorilla mux had registered `POST` only on `/api/auth/login`, so **OPTIONS did not match any route**, returned **404**, and never attached `Access-Control-Allow-Origin`. Wrapping the **entire** router in the CORS handler (instead of only `router.Use` after registration) ensures preflight always gets a 200 and the right headers before the real `POST` runs.

Each of these is small on paper. Together they are the difference between a demo that “works on my machine if you know the incantation” and one a reviewer or hiring manager can actually run.

## Design choices (brief)

- **Kafka** for append-only streaming, replay, and clear service boundaries (versus a single monolithic queue).
- **Postgres** for ACID state, joins for analyst queries, and a straightforward audit story for transactions, scores, and alerts.
- **Logistic regression first** for explainability and latency before moving to black-box models where model risk management gets heavier.

## Roadmap (honest)

- More automated tests and an integration test that stands up Compose in CI.
- Deeper metrics and SLOs per service.
- Optional WebSocket or SSE on the gateway if the UI needs push instead of polling.
- Stronger auth (verify passwords against `users`, session revocation, audit log).

## Documentation index

- [ARCHITECTURE.md](./ARCHITECTURE.md)
- [DATABASE_SCHEMA.md](./DATABASE_SCHEMA.md)
- [frontend/dashboard/README.md](./frontend/dashboard/README.md)

## Licence

This project is released under the MIT licence. See the `LICENSE` file in the repository if present.

## Closing

I treat this project as a portfolio piece for **software engineering in regulated, high-stakes domains**: clear data paths, operational guardrails, and straight talk about what is measured versus what is claimed. If you want to discuss how this would map to your stack, partitioning strategy, or control standards, that is exactly the conversation I am looking for.

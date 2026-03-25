You are a senior staff engineer building a production-grade fintech system.

Your task is to design and implement a **Real-Time Fraud Detection & Risk Scoring Platform** using:

* Golang (backend services)
* PostgreSQL (database)
* Next.js (frontend dashboard)
* AI/ML model (fraud scoring)

This system must reflect how a real bank (e.g. JPMorgan Chase) would design internal fraud detection systems.

---

## 🎯 SYSTEM GOAL

Simulate high-volume financial transactions in real-time, process them through a distributed system, assign a fraud risk score using an AI model, and display results in a live dashboard.

---

## 🧱 ARCHITECTURE REQUIREMENTS

Design this as a **microservices-based, event-driven system**.

### Services (Golang):

1. transaction-ingest-service

   * Generates or receives transaction events
   * Publishes events to a queue (use Kafka or Redis Streams)

2. risk-engine-service

   * Consumes transaction events
   * Applies fraud detection logic + AI model scoring
   * Outputs risk score (0–100)
   * Publishes enriched event

3. alert-service

   * Consumes scored transactions
   * Triggers alerts for high-risk scores
   * Sends webhook / logs alerts

4. api-gateway

   * REST API for frontend
   * Authentication (JWT)
   * Rate limiting

---

## 🧠 AI FRAUD MODEL REQUIREMENTS

Implement a simple but realistic fraud scoring model:

* Use a trainable model (start simple: logistic regression or scoring function)

* Train on synthetic transaction data

* Features should include:

  * transaction amount
  * frequency (velocity)
  * location deviation
  * time anomalies
  * merchant category risk

* Output:

  * fraud probability (0–1)
  * convert to risk score (0–100)

* Include:

  * training script
  * saved model
  * inference inside risk-engine-service

---

## 🗄️ DATABASE (PostgreSQL)

Design proper schema:

Tables:

* transactions
* risk_scores
* alerts
* users

Requirements:

* indexed queries
* support high write throughput
* store timestamps for time-series queries

---

## 🌐 FRONTEND (Next.js)

Build a clean dashboard with:

* Live transaction feed
* Risk score visualization
* Fraud alerts panel
* Charts (risk distribution, trends)

Use:

* WebSockets or polling for real-time updates

---

## ⚡ PERFORMANCE & SCALE

* Simulate high throughput (1000+ transactions/sec)
* Use concurrency in Go (goroutines, channels)
* Ensure services are decoupled

---

## 🔐 SECURITY

* JWT authentication
* Input validation
* Rate limiting
* Audit logging

---

## 📊 OBSERVABILITY (IMPORTANT)

* Structured logging
* Metrics (request rate, processing time)
* Error tracking

---

## 🧪 TESTING

* Unit tests for services
* Integration test for pipeline
* Load simulation tool

---

## 📁 PROJECT STRUCTURE

Create a clean monorepo:

/services
/transaction-ingest
/risk-engine
/alert-service
/api-gateway
/frontend
/dashboard (Next.js)
/ml
/training
/model
/database
/migrations
/docs
architecture.md

---

## 📄 DOCUMENTATION

Generate:

* Architecture diagram (text-based explanation)
* README with:

  * system overview
  * tech stack
  * how to run locally
  * design decisions
  * trade-offs

---

## 🚀 PRIORITY

Focus on:

1. Clean architecture
2. Realistic system design
3. Code quality
4. Clear separation of concerns

Avoid:

* Overengineering UI
* Fake/mock logic without explanation

---

## OUTPUT EXPECTATION

Start by:

1. Designing full system architecture
2. Defining database schema
3. Creating service scaffolding
4. Then implement each service step-by-step

Explain decisions like a senior engineer would.

Build this as if it will be reviewed by engineers at JPMorgan Chase.

After every checkpoint, you should pause, and push the code to the github repository using the git cli.

# 🧠 Engineering Rules — Sentinel Fraud Engine

This document defines strict coding and design standards to ensure all code in this repository reflects **senior-level engineering quality** and avoids patterns typically associated with low-quality or AI-generated code.

The goal is to produce code that feels **intentional, opinionated, and production-ready**.

---

## 🚫 1. No Generic or Template-Like Code

Avoid:

* Overly generic abstractions with no clear purpose
* Boilerplate-heavy structures without justification
* Repetitive patterns that add no clarity

Rules:

* Every abstraction must solve a real problem
* If something feels “auto-generated”, rewrite it
* Prefer fewer, well-designed components over many shallow ones

---

## 🧱 2. Design Before Code

Before implementing any feature:

* Define its responsibility
* Define its inputs/outputs
* Consider failure cases
* Identify how it fits into the system

Code should reflect:

* Clear boundaries between services
* Strong separation of concerns
* Thoughtful data flow

---

## 🧼 3. Code Must Be Opinionated

Bad code is neutral. Good code has **clear decisions**.

Avoid:

* “Flexible for future use” without real need
* Over-configurability
* Premature generalisation

Prefer:

* Explicit logic over abstraction
* Hard decisions over vague extensibility

---

## ✍️ 4. Naming Must Be Precise

Names should reflect **intent, not implementation**.

Good:

* `CalculateRiskScore`
* `TransactionVelocityWindow`
* `FlagHighRiskTransaction`

Bad:

* `ProcessData`
* `HandleThing`
* `DoRisk`

Rules:

* No abbreviations unless industry standard
* No vague names
* Functions should read like sentences

---

## 🧠 5. Business Logic First, Not Framework First

Avoid writing code shaped by frameworks.

Instead:

* Model the **domain (fraud detection)** first
* Then implement using tools

Example:

* Define what “risk” means before writing handlers
* Define scoring logic before wiring queues

---

## 🔍 6. No “Magic” Logic

All non-trivial logic must be:

* Explainable
* Traceable
* Justified

Rules:

* No unexplained constants (e.g. `if score > 73`)
* Extract meaningful values:

  * `HighRiskThreshold = 75`
* Document WHY decisions exist

---

## 🧪 7. Edge Cases Are First-Class

Senior code handles failure properly.

Always consider:

* Missing data
* Duplicate events
* Out-of-order processing
* Time-based anomalies

If it can break, handle it explicitly.

---

## ⚙️ 8. Keep Functions Small but Meaningful

Avoid:

* Massive 100+ line functions
* Functions that do multiple unrelated things

Prefer:

* Focused functions with one responsibility
* Logical grouping of operations

---

## 🧵 9. Concurrency Must Be Intentional (Go)

Do not:

* Spawn goroutines randomly
* Ignore race conditions

Instead:

* Define ownership of data
* Use channels deliberately
* Control lifecycle of workers

---

## 🗄️ 10. Database Access Must Be Explicit

Avoid:

* Hidden queries
* Implicit ORM magic

Prefer:

* Clear SQL or well-defined queries
* Explicit transactions
* Indexed access patterns

---

## 🔐 11. Security Is Not Optional

Always include:

* Input validation
* Authentication checks
* Rate limiting considerations

Never assume:

* Inputs are safe
* Internal services are trusted

---

## 📊 12. Observability Is Built-In

Code must produce:

* Structured logs (not random prints)
* Useful error messages
* Context-rich debugging info

Every important operation should be traceable.

---

## 🧠 13. Comments Explain WHY, Not WHAT

Avoid:

```
// increment i
i++
```

Prefer:

```
// increment retry count to prevent infinite processing loops
i++
```

---

## 🚫 14. No Overuse of Patterns

Avoid blindly using:

* Factory patterns
* Interfaces everywhere
* Dependency injection for everything

Use patterns ONLY when:

* They reduce complexity
* They improve clarity

---

## 🧩 15. Consistency > Cleverness

Do not:

* Introduce “smart” shortcuts
* Use inconsistent styles

Prefer:

* Predictable structure
* Familiar patterns across services

---

## 🔄 16. Refactor Aggressively

If something feels:

* awkward
* unclear
* overcomplicated

Rewrite it.

Senior engineers **don’t settle for “it works”**.

---

## 📁 17. File Structure Reflects Architecture

Files should:

* Represent real system boundaries
* Group related logic

Avoid:

* Dumping everything into “utils”
* Random file placement

---

## 🧠 18. Think Like a Reviewer

Before committing, ask:

* Would this make sense to a senior engineer?
* Is the intent obvious without explanation?
* Does this look production-ready?

If not, improve it.

---

## ⚠️ 19. No Placeholder Code in Main Branch

Never commit:

* TODO-heavy code
* Fake/mock implementations without marking them clearly
* Incomplete logic

---

## 🏁 20. Final Standard

All code in this repo should feel like:

> “This could be deployed inside a system at JPMorgan Chase without embarrassment.”

If it doesn’t meet that bar, it does not get merged.

---

#!/bin/bash
set -e

echo "================================================"
echo "Sentinel Fraud Engine - Startup Script"
echo "================================================"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

echo "✓ Docker is running"

# Build Docker images
echo ""
echo "Building Docker images..."
docker-compose build

# Start infrastructure services first
echo ""
echo "Starting PostgreSQL and Kafka..."
docker-compose up -d postgres zookeeper kafka

# Wait for PostgreSQL to be ready
echo ""
echo "Waiting for PostgreSQL to be ready..."
until docker exec sentinel-postgres pg_isready -U postgres > /dev/null 2>&1; do
    echo "  Waiting for PostgreSQL..."
    sleep 2
done
echo "✓ PostgreSQL is ready"

# Run database migrations
echo ""
echo "Running database migrations..."
cd database
go run migrate.go up
cd ..
echo "✓ Migrations complete"

# Run database seeds
echo ""
echo "Seeding database with test users..."
docker exec -i sentinel-postgres psql -U postgres -d sentinel_fraud < database/seeds/001_users.sql
echo "✓ Database seeded"

# Wait for Kafka to be ready
echo ""
echo "Waiting for Kafka to be ready..."
sleep 10
echo "✓ Kafka is ready"

# Start all application services
echo ""
echo "Starting all services..."
docker-compose up -d

echo ""
echo "================================================"
echo "✓ All services started successfully!"
echo "================================================"
echo ""
echo "Service endpoints:"
echo "  API Gateway:  http://localhost:8000"
echo "  PostgreSQL:   localhost:5432"
echo "  Kafka:        localhost:9092"
echo ""
echo "Check logs:"
echo "  docker-compose logs -f [service-name]"
echo ""
echo "Stop services:"
echo "  docker-compose down"
echo ""
echo "Test API:"
echo "  curl http://localhost:8000/health"
echo ""

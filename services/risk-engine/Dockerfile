# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy shared code
COPY shared/ ./shared/

# Copy service code
COPY services/risk-engine/ ./services/risk-engine/

# Build binary
WORKDIR /build/services/risk-engine
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM python:3.11-slim

# Install Python dependencies for model inference
RUN pip install --no-cache-dir joblib==1.3.2 scikit-learn==1.4.1.post1 numpy==1.26.4

WORKDIR /app

COPY --from=builder /build/services/risk-engine/main .
COPY ml/model/ /app/ml/model/

CMD ["./main"]

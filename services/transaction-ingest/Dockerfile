# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy shared code
COPY shared/ ./shared/

# Copy service code
COPY services/transaction-ingest/ ./services/transaction-ingest/

# Build binary
WORKDIR /build/services/transaction-ingest
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/services/transaction-ingest/main .

CMD ["./main"]

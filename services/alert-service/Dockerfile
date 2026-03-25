# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy shared code
COPY shared/ ./shared/

# Copy service code
COPY services/alert-service/ ./services/alert-service/

# Build binary
WORKDIR /build/services/alert-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/services/alert-service/main .

CMD ["./main"]

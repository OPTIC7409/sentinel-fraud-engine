# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy shared code
COPY shared/ ./shared/

# Copy service code
COPY services/api-gateway/ ./services/api-gateway/

# Build binary
WORKDIR /build/services/api-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/services/api-gateway/main .

EXPOSE 8000

CMD ["./main"]

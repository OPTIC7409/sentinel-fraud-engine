package main

import (
"context"
"flag"
"fmt"
"strings"
"sync/atomic"
"time"

"github.com/rs/zerolog/log"
"github.com/sentinel-fraud-engine/monorepo/shared/kafka"
"github.com/sentinel-fraud-engine/monorepo/shared/logger"
)

func main() {
logger.InitLogger("load-test", true)

targetTPS := flag.Int("tps", 100, "Target transactions per second")
duration := flag.Int("duration", 60, "Test duration in seconds")
kafkaBrokers := flag.String("brokers", "localhost:9092", "Kafka brokers")
flag.Parse()

log.Info().
Int("target_tps", *targetTPS).
Int("duration_seconds", *duration).
Str("kafka_brokers", *kafkaBrokers).
Msg("Starting load test")

producer := kafka.NewProducer([]string{*kafkaBrokers}, kafka.TopicRawTransactions)
defer producer.Close()

var sentCount, errorCount int64
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*duration)*time.Second)
defer cancel()

interval := time.Second / time.Duration(*targetTPS)
ticker := time.NewTicker(interval)
defer ticker.Stop()

startTime := time.Now()
txnNum := 0

for {
select {
case <-ctx.Done():
elapsed := time.Since(startTime).Seconds()
actualTPS := float64(sentCount) / elapsed

fmt.Println("\n" + strings.Repeat("=", 60))
fmt.Println("LOAD TEST RESULTS")
fmt.Println(strings.Repeat("=", 60))
fmt.Printf("Transactions sent: %d\n", sentCount)
fmt.Printf("Errors: %d\n", errorCount)
fmt.Printf("Duration: %.2f seconds\n", elapsed)
fmt.Printf("Actual TPS: %.2f\n", actualTPS)
fmt.Printf("Target TPS: %d\n", *targetTPS)
if sentCount+errorCount > 0 {
fmt.Printf("Success rate: %.2f%%\n", float64(sentCount)/float64(sentCount+errorCount)*100)
}
return

case <-ticker.C:
txnNum++
event := map[string]interface{}{
"transaction_id":    fmt.Sprintf("load_test_%d", time.Now().UnixNano()),
"user_id":           1 + (time.Now().Unix() % 10000),
"amount":            50.0 + float64(time.Now().Unix()%500),
"currency":          "USD",
"merchant_id":       fmt.Sprintf("merchant_%d", time.Now().Unix()%100),
"merchant_category": "retail",
"timestamp":         time.Now().Format(time.RFC3339),
"published_at":      time.Now().Format(time.RFC3339),
}

err := producer.Publish(ctx, fmt.Sprintf("%d", event["user_id"]), event)
if err != nil {
atomic.AddInt64(&errorCount, 1)
} else {
atomic.AddInt64(&sentCount, 1)
}
}
}
}

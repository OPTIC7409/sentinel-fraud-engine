package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

const (
	// Topic names
	TopicRawTransactions    = "raw-transactions"
	TopicScoredTransactions = "scored-transactions"
	TopicAlerts             = "alerts"
)

// Config holds Kafka connection configuration
type Config struct {
	Brokers       []string
	ConsumerGroup string
}

// Producer wraps Kafka writer for publishing events
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer creates a new Kafka producer for specified topic
func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{}, // Hash by key for ordering per user
		RequiredAcks: kafka.RequireAll,
		Compression:  kafka.Snappy,
		BatchSize:    100, // Batch for performance
		BatchTimeout: 10 * time.Millisecond,
	}

	log.Info().
		Str("topic", topic).
		Strs("brokers", brokers).
		Msg("Kafka producer initialized")

	return &Producer{
		writer: writer,
		topic:  topic,
	}
}

// Publish sends a message to Kafka topic
func (p *Producer) Publish(ctx context.Context, key string, value interface{}) error {
	// Serialize value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create Kafka message
	msg := kafka.Message{
		Key:   []byte(key),
		Value: valueBytes,
		Time:  time.Now(),
	}

	// Write message
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	log.Debug().
		Str("topic", p.topic).
		Str("key", key).
		Int("size_bytes", len(valueBytes)).
		Msg("Message published to Kafka")

	return nil
}

// Close closes the Kafka writer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Consumer wraps Kafka reader for consuming events
type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// NewConsumer creates a new Kafka consumer for specified topic and consumer group
func NewConsumer(brokers []string, topic string, consumerGroup string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        consumerGroup,
		MinBytes:       1e3,  // 1KB min
		MaxBytes:       10e6, // 10MB max
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset, // Start from end for new consumers
	})

	log.Info().
		Str("topic", topic).
		Str("consumer_group", consumerGroup).
		Strs("brokers", brokers).
		Msg("Kafka consumer initialized")

	return &Consumer{
		reader: reader,
		topic:  topic,
	}
}

// Consume reads messages from Kafka in a loop
func (c *Consumer) Consume(ctx context.Context, handler func([]byte, []byte) error) error {
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("topic", c.topic).Msg("Consumer shutting down")
			return ctx.Err()
		default:
			// Read message with timeout
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return nil
				}
				log.Error().Err(err).Str("topic", c.topic).Msg("Failed to fetch message")
				time.Sleep(time.Second) // Backoff on error
				continue
			}

			// Process message
			startTime := time.Now()
			if err := handler(msg.Key, msg.Value); err != nil {
				log.Error().
					Err(err).
					Str("topic", c.topic).
					Str("key", string(msg.Key)).
					Int("offset", int(msg.Offset)).
					Msg("Failed to process message")
				// Don't commit failed messages - will retry
				continue
			}

			// Commit offset after successful processing (at-least-once delivery)
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Error().Err(err).Msg("Failed to commit message offset")
			}

			processingTime := time.Since(startTime)
			log.Debug().
				Str("topic", c.topic).
				Str("key", string(msg.Key)).
				Int("offset", int(msg.Offset)).
				Int64("processing_ms", processingTime.Milliseconds()).
				Msg("Message processed successfully")
		}
	}
}

// Close closes the Kafka reader
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// EnsureTopicsExist creates Kafka topics if they don't exist
func EnsureTopicsExist(brokers []string, topics []string) error {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial Kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to dial controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := make([]kafka.TopicConfig, len(topics))
	for i, topic := range topics {
		topicConfigs[i] = kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     8, // Allow parallel consumption
			ReplicationFactor: 1, // Single node for local dev
		}
	}

	if err := controllerConn.CreateTopics(topicConfigs...); err != nil {
		// Ignore "topic already exists" errors
		log.Warn().Err(err).Msg("Topic creation warning (may already exist)")
	}

	log.Info().Strs("topics", topics).Msg("Kafka topics ensured")
	return nil
}

package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger configures structured JSON logging for a service.
// All services use consistent logging format for observability.
func InitLogger(serviceName string, isDevelopment bool) {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	if isDevelopment {
		// Pretty print for local development
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	} else {
		// JSON output for production (parseable by log aggregators)
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Str("service", serviceName).Logger()
	}

	// Set global log level
	if isDevelopment {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// WithRequestID adds request_id to logger context for tracing.
func WithRequestID(requestID string) zerolog.Logger {
	return log.With().Str("request_id", requestID).Logger()
}

// WithTransactionID adds transaction_id to logger context.
func WithTransactionID(txnID string) zerolog.Logger {
	return log.With().Str("transaction_id", txnID).Logger()
}

// WithError logs error with full context and stack trace.
func LogError(err error, msg string, fields map[string]interface{}) {
	event := log.Error().Err(err)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

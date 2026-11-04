package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr string
	LogLevel string

	DatabaseURL string

	CustomerGRPCAddr string

	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	var errs []error

	cfg := &Config{
		GRPCAddr:         getEnv("ACCOUNT_GRPC_ADDR", ":50056"),
		LogLevel:         getEnv("ACCOUNT_LOG_LEVEL", "info"),
		DatabaseURL:      os.Getenv("ACCOUNT_DATABASE_URL"),
		CustomerGRPCAddr: getEnv("ACCOUNT_CUSTOMER_GRPC_ADDR", "localhost:50052"),
		KafkaBrokers:     splitCSV(getEnv("ACCOUNT_KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:       getEnv("ACCOUNT_KAFKA_TOPIC", "kyc.events"),
		KafkaGroupID:     getEnv("ACCOUNT_KAFKA_GROUP_ID", "account-service"),
	}

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("ACCOUNT_DATABASE_URL is required"))
	}
	if len(cfg.KafkaBrokers) == 0 {
		errs = append(errs, errors.New("ACCOUNT_KAFKA_BROKERS must not be empty"))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid account-service config: %w", errors.Join(errs...))
	}

	return cfg, nil
}

func getEnv(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func splitCSV(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
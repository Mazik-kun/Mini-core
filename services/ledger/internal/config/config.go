package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr    string
	AccountGRPCAddr string
	LogLevel    string
	DatabaseURL string

	KafkaBrokers []string
	KafkaTopic string
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	var errs []error

	cfg := &Config{
		GRPCAddr:    getEnv("LEDGER_GRPC_ADDR", ":50057"), // Следующий свободный порт
		AccountGRPCAddr: getEnv("LEDGER_ACCOUNT_GRPC_ADDR", ":50056"),
		LogLevel:    getEnv("LEDGER_LOG_LEVEL", "info"),
		DatabaseURL: os.Getenv("LEDGER_DATABASE_URL"),
		KafkaBrokers: splitCSV(getEnv("LEDGER_KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic: getEnv("LEDGER_KAFKA_TOPIC", "ledfer.events"),
	}

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("LEDGER_DATABASE_URL is required"))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid ledger config: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func getEnv(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func splitCSV(raw string) []string{
	var out []string

	for _, p := range strings.Split(raw, ","){
		if p = strings.TrimSpace(p); p != ""{
			out = append(out, p)
		}
	}
	return out
}
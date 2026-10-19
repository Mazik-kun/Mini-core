package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr    string
	DatabaseURL string

	KafkaBrokers []string
	KafkaGroupID string

	KafkaAuthEventsTopic     string
	KafkaCustomerEventsTopic string

	DefaultPageSize int
	MaxPageSize     int

	RedisAddr string

	JWTPublicKeyPath string
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	cfg := &Config{
		GRPCAddr:                 getEnv("CUSTOMER_GRPC_ADDR", ":50052"),
		DatabaseURL:              os.Getenv("CUSTOMER_DATABASE_URL"),
		KafkaGroupID:             getEnv("CUSTOMER_KAFKA_GROUP_ID", "customer-service"),
		KafkaAuthEventsTopic:     getEnv("KAFKA_AUTH_EVENTS_TOPIC", "auth.events"),
		KafkaCustomerEventsTopic: getEnv("KAFKA_CUSTOMER_EVENTS_TOPIC", "customer.events"),
		RedisAddr:                os.Getenv("CUSTOMER_REDIS_ADDR"),
		JWTPublicKeyPath:         os.Getenv("JWT_PUBLIC_KEY_PATH"),
	}

	if rawBrokers := os.Getenv("KAFKA_BROKERS"); rawBrokers != "" {
		cfg.KafkaBrokers = splitAndTrim(rawBrokers)
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("CUSTOMER_DATABASE_URL is not set")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS is not set")
	}
	if cfg.RedisAddr == "" {
		return nil, errors.New("CUSTOMER_REDIS_ADDR is not set")
	}
	if cfg.JWTPublicKeyPath == "" {
		return nil, errors.New("JWT_PUBLIC_KEY_PATH is not set")
	}
	if cfg.KafkaAuthEventsTopic == "" {
		return nil, errors.New("KAFKA_AUTH_EVENTS_TOPIC is empty")
	}
	if cfg.KafkaCustomerEventsTopic == "" {
		return nil, errors.New("KAFKA_CUSTOMER_EVENTS_TOPIC is empty")
	}

	var err error
	cfg.DefaultPageSize, err = envInt("CUSTOMER_DEFAULT_PAGE_SIZE", 50)
	if err != nil {
		return nil, err
	}
	cfg.MaxPageSize, err = envInt("CUSTOMER_MAX_PAGE_SIZE", 100)
	if err != nil {
		return nil, err
	}
	if cfg.DefaultPageSize <= 0 {
		return nil, errors.New("CUSTOMER_DEFAULT_PAGE_SIZE must be > 0")
	}
	if cfg.MaxPageSize < cfg.DefaultPageSize {
		return nil, errors.New("CUSTOMER_MAX_PAGE_SIZE must be >= CUSTOMER_DEFAULT_PAGE_SIZE")
	}

	return cfg, nil
}

func getEnv(name, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return defaultValue
}

func envInt(name string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

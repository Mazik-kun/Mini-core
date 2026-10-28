package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr    string
	DatabaseURL string

	JWTPublicKeyPath string

	CustomerGRPCAddr string
	DocumentGRPCAddr string

	KafkaBrokers      []string
	KafkaKycTopic     string
	KafkaConsumerGroup string

	WorkerInterval time.Duration
	WorkerBatch    int
	MaxAttempts    int
	MinAge         int

	DefaultPageSize int
	MaxPageSize     int
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	cfg := &Config{
		GRPCAddr:         getEnv("KYC_GRPC_ADDR", ":50055"),
		DatabaseURL:      os.Getenv("KYC_DATABASE_URL"),
		JWTPublicKeyPath: os.Getenv("JWT_PUBLIC_KEY_PATH"),
		
		CustomerGRPCAddr: getEnv("GATEWAY_CUSTOMER_GRPC_ADDR", "localhost:50052"),
		DocumentGRPCAddr: getEnv("DOCUMENT_GRPC_ADDR", "localhost:50054"),

		KafkaBrokers:       strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaKycTopic:      getEnv("KAFKA_KYC_TOPIC", "kyc.events"),
		KafkaConsumerGroup: getEnv("KAFKA_KYC_GROUP_ID", "kyc-service"),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("KYC_DATABASE_URL is not set")
	}

	var err error
	if cfg.WorkerInterval, err = envDuration("KYC_WORKER_INTERVAL", 5*time.Second); err != nil {
		return nil, err
	}
	if cfg.WorkerBatch, err = envInt("KYC_WORKER_BATCH", 10); err != nil {
		return nil, err
	}
	if cfg.MaxAttempts, err = envInt("KYC_MAX_ATTEMPTS", 3); err != nil {
		return nil, err
	}
	if cfg.MinAge, err = envInt("KYC_MIN_AGE", 18); err != nil {
		return nil, err
	}
	if cfg.DefaultPageSize, err = envInt("KYC_DEFAULT_PAGE_SIZE", 50); err != nil {
		return nil, err
	}
	if cfg.MaxPageSize, err = envInt("KYC_MAX_PAGE_SIZE", 100); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func envInt(name string, def int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" { return def, nil }
	return strconv.Atoi(raw)
}

func envDuration(name string, def time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" { return def, nil }
	v, err := time.ParseDuration(raw)
	if err != nil { return 0, fmt.Errorf("%s: %w", name, err) }
	return v, nil
}
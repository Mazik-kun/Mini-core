package config

import (
	"errors"
	"os"
	"strings"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr    string
	DatabaseURL string

	KafkaBrokers []string
	KafkaGroupID string

	RedisAddr string

	JWTPublicKeyPath string
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	cfg := &Config{
		GRPCAddr:         os.Getenv("CUSTOMER_GRPC_ADDR"),
		DatabaseURL:      os.Getenv("CUSTOMER_DATABASE_URL"),
		KafkaGroupID:     os.Getenv("CUSTOMER_KAFKA_GROUP_ID"),
		RedisAddr:        os.Getenv("CUSTOMER_REDIS_ADDR"),
		JWTPublicKeyPath: os.Getenv("JWT_PUBLIC_KEY_PATH"),
	}

	if rawBrokers := os.Getenv("KAFKA_BROKERS"); rawBrokers != "" {
		cfg.KafkaBrokers = strings.Split(rawBrokers, ",")
	}

	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":50052"
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("CUSTOMER_DATABASE_URL is not set")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS is not set")
	}
	if cfg.KafkaGroupID == "" {
		cfg.KafkaGroupID = "customer-service"
	}
	if cfg.RedisAddr == "" {
		return nil, errors.New("CUSTOMER_REDIS_ADDR is not set")
	}
	if cfg.JWTPublicKeyPath == "" {
		return nil, errors.New("JWT_PUBLIC_KEY_PATH is not set")
	}

	return cfg, nil
}
package config

import (
	"errors"
	"os"
	"strings"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr          string
	HTTPAddr          string
	DatabaseURL       string
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	JWTKeyID          string
	KafkaBrokers      []string
	KafkaGroupID string
	RedisAddr    string

	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	cfg := &Config{
		GRPCAddr:          os.Getenv("AUTH_GRPC_ADDR"),
		HTTPAddr:          os.Getenv("AUTH_HTTP_ADDR"),
		DatabaseURL:       os.Getenv("AUTH_DATABASE_URL"),
		JWTPrivateKeyPath: os.Getenv("JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:  os.Getenv("JWT_PUBLIC_KEY_PATH"),
		JWTKeyID:          os.Getenv("JWT_KEY_ID"),
		KafkaBrokers:      parseKafkaBrokers(os.Getenv("KAFKA_BROKERS")),
		KafkaGroupID: os.Getenv("KAFKA_GROUP_ID"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),

		BootstrapAdminEmail:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
	}

	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":50051"
	}
	if cfg.KafkaGroupID == "" {
		cfg.KafkaGroupID = "auth-service"
	}
	if cfg.RedisAddr == "" {
		return nil, errors.New("REDIS_ADDR is not set")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if cfg.JWTKeyID == "" {
		cfg.JWTKeyID = "auth-1"
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	if cfg.JWTPrivateKeyPath == "" {
		return nil, errors.New("JWT_PRIVATE_KEY_PATH is not set")
	}
	if cfg.JWTPublicKeyPath == "" {
		return nil, errors.New("JWT_PUBLIC_KEY_PATH is not set")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS is not set")
	}

	return cfg, nil
}

// parseKafkaBrokers разбирает "a:9092,b:9092" в срез адресов.
func parseKafkaBrokers(rawBrokers string) []string {
	if rawBrokers == "" {
		return nil
	}

	parts := strings.Split(rawBrokers, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}
package config

import (
	"errors"
	"os"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr    string
	DatabaseURL string
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	cfg := &Config{
		GRPCAddr:    os.Getenv("AUTH_GRPC_ADDR"),
		DatabaseURL: os.Getenv("AUTH_DATABASE_URL"),
	}

	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":50051"
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	return cfg, nil
}
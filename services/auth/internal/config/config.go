package config

import (
	"errors"
	"os"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr          string
	DatabaseURL       string
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	cfg := &Config{
		GRPCAddr:          os.Getenv("AUTH_GRPC_ADDR"),
		DatabaseURL:       os.Getenv("AUTH_DATABASE_URL"),
		JWTPrivateKeyPath: os.Getenv("JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:  os.Getenv("JWT_PUBLIC_KEY_PATH"),
	}

	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":50051"
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

	return cfg, nil
}
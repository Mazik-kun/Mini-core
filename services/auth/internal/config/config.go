package config

import (
	"errors"
	"os"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
)

type Config struct {
	GRPCAddr          string
	HTTPAddr          string
	DatabaseURL       string
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	JWTKeyID          string

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

		BootstrapAdminEmail:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
	}

	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = ":50051"
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

	return cfg, nil
}
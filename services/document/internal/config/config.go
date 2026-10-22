package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	envloader "github.com/Mazik-kun/mini-core/pkg/config"
	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

type Config struct {
	GRPCAddr    string
	DatabaseURL string

	JWTPublicKeyPath string

	MinioEndpoint  string
	// MinioPublicEndpoint — хост в presigned URL (виден клиенту, не docker-сети). Пусто = MinioEndpoint.
	MinioPublicEndpoint string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool
	MinioBucket    string

	UploadURLTTL   time.Duration
	DownloadURLTTL time.Duration
	MaxSize        int64

	// RequiredDocTypes — KYC_REQUIRED_DOC_TYPES; дефолт для HasRequiredDocuments при пустом required_types.
	RequiredDocTypes []domain.Type

	DefaultPageSize int
	MaxPageSize     int
}

func Load() (*Config, error) {
	envloader.LoadEnv()

	cfg := &Config{
		GRPCAddr:         getEnv("DOCUMENT_GRPC_ADDR", ":50054"),
		DatabaseURL:      os.Getenv("DOCUMENT_DATABASE_URL"),
		JWTPublicKeyPath: os.Getenv("JWT_PUBLIC_KEY_PATH"),
		MinioEndpoint:    os.Getenv("MINIO_ENDPOINT"),
		MinioPublicEndpoint: strings.TrimSpace(os.Getenv("MINIO_PUBLIC_ENDPOINT")),
		MinioAccessKey:   os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:   os.Getenv("MINIO_SECRET_KEY"),
		MinioBucket:      getEnv("MINIO_BUCKET", "documents"),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DOCUMENT_DATABASE_URL is not set")
	}
	if cfg.JWTPublicKeyPath == "" {
		return nil, errors.New("JWT_PUBLIC_KEY_PATH is not set")
	}
	if cfg.MinioEndpoint == "" || cfg.MinioAccessKey == "" || cfg.MinioSecretKey == "" {
		return nil, errors.New("MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY must be set")
	}

	var err error
	if cfg.MinioUseSSL, err = envBool("MINIO_USE_SSL", false); err != nil {
		return nil, err
	}
	if cfg.UploadURLTTL, err = envDuration("DOC_UPLOAD_URL_TTL", 15*time.Minute); err != nil {
		return nil, err
	}
	if cfg.DownloadURLTTL, err = envDuration("DOC_DOWNLOAD_URL_TTL", 5*time.Minute); err != nil {
		return nil, err
	}
	if cfg.MaxSize, err = envInt64("DOC_MAX_SIZE", 10485760); err != nil {
		return nil, err
	}
	if cfg.MaxSize <= 0 {
		return nil, errors.New("DOC_MAX_SIZE must be > 0")
	}
	if cfg.RequiredDocTypes, err = requiredDocTypes(); err != nil {
		return nil, err
	}
	if cfg.DefaultPageSize, err = envInt("DOCUMENT_DEFAULT_PAGE_SIZE", 50); err != nil {
		return nil, err
	}
	if cfg.MaxPageSize, err = envInt("DOCUMENT_MAX_PAGE_SIZE", 100); err != nil {
		return nil, err
	}
	if cfg.DefaultPageSize <= 0 || cfg.MaxPageSize < cfg.DefaultPageSize {
		return nil, errors.New("invalid DOCUMENT page size settings")
	}
	return cfg, nil
}

// requiredDocTypes читает KYC_REQUIRED_DOC_TYPES (используется document-service
// для валидации входящих типов и kyc-service для вызова HasRequiredDocuments).
func requiredDocTypes() ([]domain.Type, error) {
	raw := getEnv("KYC_REQUIRED_DOC_TYPES", "PASSPORT,SELFIE")
	var out []domain.Type
	for _, p := range strings.Split(raw, ",") {
		if strings.TrimSpace(p) == "" {
			continue
		}
		t, err := domain.ParseType(p)
		if err != nil {
			return nil, fmt.Errorf("KYC_REQUIRED_DOC_TYPES: %w", err)
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil, errors.New("KYC_REQUIRED_DOC_TYPES is empty")
	}
	return out, nil
}

func getEnv(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func envInt(name string, def int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func envInt64(name string, def int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func envBool(name string, def bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func envDuration(name string, def time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	if v <= 0 {
		return 0, fmt.Errorf("%s must be > 0", name)
	}
	return v, nil
}

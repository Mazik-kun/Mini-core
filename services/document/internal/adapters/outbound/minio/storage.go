// Package minio — адаптер ObjectStorage поверх minio-go v7.
package minio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

// presignRegion фиксирован, чтобы подпись не требовала сетевого запроса за регионом
// (клиент для presign может смотреть на публичный адрес, недоступный из контейнера).
const presignRegion = "us-east-1"

type Options struct {
	Endpoint       string // внутренний адрес (из docker-сети)
	PublicEndpoint string // адрес, который увидит клиент в presigned URL; пусто = Endpoint
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	Bucket         string
}

type Storage struct {
	client  *miniogo.Client // операции Stat/Get/Remove
	presign *miniogo.Client // только подпись URL
	bucket  string
}

func New(o Options) (*Storage, error) {
	creds := credentials.NewStaticV4(o.AccessKey, o.SecretKey, "")

	client, err := miniogo.New(o.Endpoint, &miniogo.Options{Creds: creds, Secure: o.UseSSL})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	publicEndpoint := o.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = o.Endpoint
	}
	presign, err := miniogo.New(publicEndpoint, &miniogo.Options{Creds: creds, Secure: o.UseSSL, Region: presignRegion})
	if err != nil {
		return nil, fmt.Errorf("minio presign client: %w", err)
	}

	return &Storage{client: client, presign: presign, bucket: o.Bucket}, nil
}

// EnsureBucket создаёт бакет, если его нет. Любая ошибка — fail-fast на старте.
func (s *Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", s.bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, miniogo.MakeBucketOptions{Region: presignRegion}); err != nil {
		// гонка с другим инстансом
		if resp := miniogo.ToErrorResponse(err); resp.Code == "BucketAlreadyOwnedByYou" {
			return nil
		}
		return fmt.Errorf("create bucket %q: %w", s.bucket, err)
	}
	return nil
}

func (s *Storage) PresignUpload(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.presign.PresignedPutObject(ctx, s.bucket, key, ttl)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *Storage) PresignDownload(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.presign.PresignedGetObject(ctx, s.bucket, key, ttl, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *Storage) Stat(ctx context.Context, key string) (domain.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, miniogo.StatObjectOptions{})
	if err != nil {
		return domain.ObjectInfo{}, mapErr(err)
	}
	return domain.ObjectInfo{SizeBytes: info.Size}, nil
}

// ReadHead читает первые n байт объекта (Range: bytes=0-(n-1)).
func (s *Storage) ReadHead(ctx context.Context, key string, n int) ([]byte, error) {
	opts := miniogo.GetObjectOptions{}
	if err := opts.SetRange(0, int64(n)-1); err != nil {
		return nil, err
	}
	obj, err := s.client.GetObject(ctx, s.bucket, key, opts)
	if err != nil {
		return nil, mapErr(err)
	}
	defer obj.Close()

	head, err := io.ReadAll(io.LimitReader(obj, int64(n)))
	if err != nil {
		return nil, mapErr(err)
	}
	return head, nil
}

// SHA256 считает хеш потоком, не загружая объект в память.
func (s *Storage) SHA256(ctx context.Context, key string) (string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, miniogo.GetObjectOptions{})
	if err != nil {
		return "", mapErr(err)
	}
	defer obj.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, obj); err != nil {
		return "", mapErr(err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *Storage) Remove(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, miniogo.RemoveObjectOptions{})
}

func mapErr(err error) error {
	if resp := miniogo.ToErrorResponse(err); resp.Code == "NoSuchKey" || resp.StatusCode == 404 {
		return domain.ErrObjectNotFound
	}
	return err
}

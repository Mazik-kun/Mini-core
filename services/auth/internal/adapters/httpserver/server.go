package httpserver

import (
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second

	healthPath = "/healthz"
	jwksPath   = "/.well-known/jwks.json"
)

type Server struct {
	jwksJSON []byte
	logger   *slog.Logger
	httpSrv  *http.Server
}

// NewServer готовит HTTP-сервер с /healthz и /.well-known/jwks.json.
func NewServer(addr string, publicKey ed25519.PublicKey, keyID string, logger *slog.Logger) (*Server, error) {
	jwksJSON, err := BuildJWKS(publicKey, keyID)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	srv := &Server{
		jwksJSON: jwksJSON,
		logger:   logger,
	}
	mux.HandleFunc(healthPath, srv.handleHealthz)
	mux.HandleFunc(jwksPath, srv.handleJWKS)

	srv.httpSrv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return srv, nil
}

// Run запускает сервер и завершает его при отмене ctx.
func (s *Server) Run(ctx context.Context) error {
	serveErr := make(chan error, 1)
	go s.serve(serveErr)

	select {
	case <-ctx.Done():
		return s.shutdown()
	case err := <-serveErr:
		return err
	}
}

func (s *Server) serve(serveErr chan<- error) {
	s.logger.Info("http server started", "addr", s.httpSrv.Addr)
	err := s.httpSrv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		serveErr <- err
	}
}

func (s *Server) shutdown() error {
	s.logger.Info("http server stopping")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return s.httpSrv.Shutdown(shutdownCtx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(s.jwksJSON)
}
// Package httpserver — HTTP-сервер gateway и сборка роутера.
package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type Server struct {
	srv             *http.Server
	shutdownTimeout time.Duration
}

func New(addr string, h http.Handler, shutdownTimeout time.Duration) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           h,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		shutdownTimeout: shutdownTimeout,
	}
}

// Run блокируется до ошибки сервера или отмены ctx, затем делает graceful shutdown.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	sctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.srv.Shutdown(sctx)
}

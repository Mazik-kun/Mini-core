package middleware

import (
	"net"
	"net/http"
	"strings"
)

// statusRecorder запоминает статус и размер ответа для logging/metrics.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func wrapWriter(w http.ResponseWriter) *statusRecorder {
	if sr, ok := w.(*statusRecorder); ok {
		return sr
	}
	return &statusRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status, s.wrote = code, true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.wrote = true
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// Unwrap позволяет http.ResponseController достучаться до оригинала (Flush и т.п.).
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// ClientIP возвращает IP клиента. X-Forwarded-For учитывается только если gateway
// стоит за доверенным прокси (trustXFF) — иначе его можно подделать и обойти rate limit.
func ClientIP(r *http.Request, trustXFF bool) string {
	if trustXFF {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first, _, _ := strings.Cut(xff, ",")
			if ip := strings.TrimSpace(first); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

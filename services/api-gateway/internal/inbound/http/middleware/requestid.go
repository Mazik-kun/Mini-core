package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

const HeaderRequestID = "X-Request-ID"

// RequestID принимает валидный входящий X-Request-ID, иначе генерирует UUIDv7.
// Кладёт *reqctx.Info в ctx и дублирует id в заголовок ответа.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(HeaderRequestID)
			if !validRequestID(id) {
				if u, err := uuid.NewV7(); err == nil {
					id = u.String()
				} else {
					id = uuid.NewString()
				}
			}
			w.Header().Set(HeaderRequestID, id)
			ctx := reqctx.WithInfo(r.Context(), &reqctx.Info{RequestID: id})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func validRequestID(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
		default:
			return false
		}
	}
	return true
}

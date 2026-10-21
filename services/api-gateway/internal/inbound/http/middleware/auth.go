package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"

	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

// publicRoutes — единственные эндпоинты без JWT. Всё остальное (включая неизвестные
// пути и /v1/auth/officers) требует валидного токена.
var publicRoutes = map[string]struct{}{
	"POST /v1/auth/register": {},
	"POST /v1/auth/login":    {},
	"POST /v1/auth/refresh":  {},
	"POST /v1/auth/logout":   {},
}

func IsPublic(method, path string) bool {
	_, ok := publicRoutes[method+" "+path]
	return ok
}

// Auth валидирует JWT локально (Ed25519, ключ по kid из JWKS-кэша), кладёт claims и
// сырой токен в ctx. issuer == "" отключает проверку iss.
//
// Для публичных маршрутов входящий Authorization удаляется, чтобы мусор не уехал downstream.
func Auth(keys port.JWKSProvider, issuer string, log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsPublic(r.Method, r.URL.Path) {
				r.Header.Del("Authorization")
				next.ServeHTTP(w, r)
				return
			}

			raw, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				unauthenticated(w, r, "missing or malformed bearer token")
				return
			}
			kid, err := authjwt.KeyID(raw)
			if err != nil {
				unauthenticated(w, r, "invalid token")
				return
			}

			pub, err := keys.KeyByID(r.Context(), kid)
			switch {
			case errors.Is(err, port.ErrJWKSUnavailable):
				log.ErrorContext(r.Context(), "jwks unavailable", "error", err)
				w.Header().Set("Retry-After", "1")
				apierr.WriteStatus(w, r, codes.Unavailable, "authentication keys unavailable")
				return
			case err != nil: // ErrKeyNotFound и всё прочее
				unauthenticated(w, r, "invalid token")
				return
			}

			claims, err := authjwt.VerifyWithKey(raw, pub)
			if err != nil {
				if errors.Is(err, authjwt.ErrTokenExpired) {
					unauthenticated(w, r, "token expired")
				} else {
					unauthenticated(w, r, "invalid token")
				}
				return
			}
			if issuer != "" && claims.Issuer != issuer {
				unauthenticated(w, r, "invalid token")
				return
			}

			ctx := reqctx.WithClaims(r.Context(), claims)
			ctx = reqctx.WithToken(ctx, raw)
			if info := reqctx.InfoFrom(ctx); info != nil {
				info.UserID = claims.Subject.String()
			}
			// Нормализуем заголовок: grpc-gateway форвардит его в metadata как есть.
			r.Header.Set("Authorization", "Bearer "+raw)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthenticated(w http.ResponseWriter, r *http.Request, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="gateway"`)
	apierr.WriteStatus(w, r, codes.Unauthenticated, msg)
}

func bearerToken(h string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(h), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

package middleware

import (
	"net/http"
	"strings"
)

// CORS: пустой список origins = CORS выключен. Preflight отвечает 204 до auth.
// Credentials (cookies) не разрешаем — API работает с Bearer-токеном.
func CORS(origins []string) Middleware {
	allowAll := false
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		if o == "*" {
			allowAll = true
		}
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		if len(origins) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Add("Vary", "Origin")
			_, ok := allowed[origin]
			ok = ok || allowAll

			if ok {
				h := w.Header()
				if allowAll {
					h.Set("Access-Control-Allow-Origin", "*")
				} else {
					h.Set("Access-Control-Allow-Origin", origin)
				}
				h.Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After")
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				if ok {
					h := w.Header()
					h.Add("Vary", "Access-Control-Request-Method")
					h.Add("Vary", "Access-Control-Request-Headers")
					h.Set("Access-Control-Allow-Methods", strings.Join([]string{
						http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodOptions}, ", "))
					h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
					h.Set("Access-Control-Max-Age", "600")
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

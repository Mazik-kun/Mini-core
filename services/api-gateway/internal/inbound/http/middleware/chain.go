// Package middleware — HTTP-middleware gateway.
package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

// Chain собирает middleware так, что первый в списке — самый внешний.
func Chain(mws ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}

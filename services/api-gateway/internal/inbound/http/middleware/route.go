package middleware

import (
	"net/http"
	"strings"
)

const customersPrefix = "/v1/customers/"

// RouteLabel возвращает шаблон маршрута для метрик/трейсов (низкая кардинальность):
// /v1/customers/<uuid> → /v1/customers/{customer_id}, всё неизвестное → "unmatched".
func RouteLabel(r *http.Request) string {
	p := r.URL.Path
	switch p {
	case "/v1/auth/register", "/v1/auth/login", "/v1/auth/refresh", "/v1/auth/logout", "/v1/auth/officers",
		"/v1/customers", "/v1/customers/me", "/v1/customers/me/status":
		return p
	}
	if rest, ok := strings.CutPrefix(p, customersPrefix); ok && rest != "" && !strings.Contains(rest, "/") {
		return "/v1/customers/{customer_id}"
	}
	return "unmatched"
}

func methodLabel(m string) string {
	switch m {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete,
		http.MethodOptions, http.MethodHead:
		return m
	}
	return "OTHER"
}

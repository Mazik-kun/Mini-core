package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Check struct {
	Name string
	Fn   func(ctx context.Context) error
}

type Health struct {
	checks  []Check
	timeout time.Duration
}

func NewHealth(checks ...Check) *Health { return &Health{checks: checks, timeout: 2 * time.Second} }

// Live — liveness: процесс жив.
func (h *Health) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready — readiness: auth и customer отвечают grpc.health, Redis отвечает на PING.
func (h *Health) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	results := make(map[string]string, len(h.checks))
	ok := true
	for _, c := range h.checks {
		if err := c.Fn(ctx); err != nil {
			results[c.Name] = err.Error()
			ok = false
		} else {
			results[c.Name] = "ok"
		}
	}
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "fail", "checks": results})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "checks": results})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

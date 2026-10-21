package handlers

import (
	"net/http"

	authv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/auth/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

type Auth struct {
	client port.AuthClient
}

func NewAuth(c port.AuthClient) *Auth { return &Auth{client: c} }

func (h *Auth) Register(w http.ResponseWriter, r *http.Request) {
	var req authv1.RegisterRequest
	if !parseProtoJSON(w, r, &req) { return }
	resp, err := h.client.Register(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req authv1.LoginRequest
	if !parseProtoJSON(w, r, &req) { return }
	resp, err := h.client.Login(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

func (h *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	var req authv1.RefreshRequest
	if !parseProtoJSON(w, r, &req) { return }
	resp, err := h.client.Refresh(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	var req authv1.LogoutRequest
	if !parseProtoJSON(w, r, &req) { return }
	resp, err := h.client.Logout(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

func (h *Auth) CreateOfficer(w http.ResponseWriter, r *http.Request) {
	var req authv1.CreateOfficerRequest
	if !parseProtoJSON(w, r, &req) { return }
	resp, err := h.client.CreateOfficer(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}
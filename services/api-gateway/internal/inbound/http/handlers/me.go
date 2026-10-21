package handlers

import (
	"log/slog"
	"net/http"

	"google.golang.org/grpc/codes"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

type Me struct {
	customers port.CustomerClient
	log       *slog.Logger
}

func NewMe(c port.CustomerClient, log *slog.Logger) *Me { return &Me{customers: c, log: log} }

func (h *Me) GetMe(w http.ResponseWriter, r *http.Request) {
	id, ok := h.customerID(w, r)
	if !ok {
		return
	}
	resp, err := h.customers.GetCustomer(r.Context(), &customerv1.GetCustomerRequest{CustomerId: id})
	respondProtoJSON(w, r, resp, err)
}

func (h *Me) UpdateMe(w http.ResponseWriter, r *http.Request) {
	id, ok := h.customerID(w, r)
	if !ok {
		return
	}
	var req customerv1.UpdateProfileRequest
	if !parseProtoJSON(w, r, &req) {
		return
	}
	req.CustomerId = id // перетираем любое значение из тела: чужой профиль через /me не изменить

	resp, err := h.customers.UpdateProfile(r.Context(), &req)
	respondProtoJSON(w, r, resp, err)
}

func (h *Me) GetStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.customerID(w, r)
	if !ok {
		return
	}
	resp, err := h.customers.GetCustomerStatus(r.Context(), &customerv1.GetCustomerStatusRequest{CustomerId: id})
	respondProtoJSON(w, r, resp, err)
}

func (h *Me) customerID(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := reqctx.Claims(r.Context())
	if !ok {
		apierr.WriteStatus(w, r, codes.Unauthenticated, "authentication required")
		return "", false
	}
	if claims.CustomerID == nil {
		apierr.WriteStatus(w, r, codes.FailedPrecondition, "customer profile not yet linked")
		return "", false
	}
	return claims.CustomerID.String(), true
}
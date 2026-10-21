package handlers

import (
	"net/http"
	"strconv"

	"google.golang.org/grpc/codes"

	commonv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/common/v1"
	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/port"
)

type Customer struct {
	client port.CustomerClient
}

func NewCustomer(c port.CustomerClient) *Customer { return &Customer{client: c} }

func (h *Customer) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("customer_id") // Go 1.22+
	if id == "" {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "missing customer_id")
		return
	}
	resp, err := h.client.GetCustomer(r.Context(), &customerv1.GetCustomerRequest{CustomerId: id})
	respondProtoJSON(w, r, resp, err)
}

func (h *Customer) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := &customerv1.ListCustomersRequest{}
	
	// Парсим пагинацию
	if ps := q.Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil {
			if req.Page == nil {
				req.Page = &commonv1.PageRequest{}
			}
			req.Page.PageSize = int32(v)
		}
	}
	if pt := q.Get("page_token"); pt != "" {
		if req.Page == nil {
			req.Page = &commonv1.PageRequest{}
		}
		req.Page.PageToken = pt
	}
	
	// Парсим фильтр по статусу
	if sf := q.Get("status_filter"); sf != "" {
		if val, ok := customerv1.CustomerStatus_value[sf]; ok {
			req.StatusFilter = customerv1.CustomerStatus(val)
		}
	}

	resp, err := h.client.ListCustomers(r.Context(), req)
	respondProtoJSON(w, r, resp, err)
}
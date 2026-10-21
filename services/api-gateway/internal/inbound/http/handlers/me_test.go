package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	customerv1 "github.com/Mazik-kun/mini-core/contracts/gen/bank/customer/v1"
	"github.com/Mazik-kun/mini-core/pkg/authjwt"
	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

type fakeCustomers struct {
	gotGet    *customerv1.GetCustomerRequest
	gotUpdate *customerv1.UpdateProfileRequest
	gotStatus *customerv1.GetCustomerStatusRequest
	err       error
}

func (f *fakeCustomers) GetCustomer(_ context.Context, in *customerv1.GetCustomerRequest, _ ...grpc.CallOption) (*customerv1.Customer, error) {
	f.gotGet = in
	return &customerv1.Customer{CustomerId: in.CustomerId, Status: customerv1.CustomerStatus_CUSTOMER_STATUS_NEW}, f.err
}

func (f *fakeCustomers) UpdateProfile(_ context.Context, in *customerv1.UpdateProfileRequest, _ ...grpc.CallOption) (*customerv1.Customer, error) {
	f.gotUpdate = in
	return &customerv1.Customer{CustomerId: in.CustomerId, FullName: in.FullName,
		Status: customerv1.CustomerStatus_CUSTOMER_STATUS_PROFILE_FILLED}, f.err
}

func (f *fakeCustomers) GetCustomerStatus(_ context.Context, in *customerv1.GetCustomerStatusRequest, _ ...grpc.CallOption) (*customerv1.GetCustomerStatusResponse, error) {
	f.gotStatus = in
	return &customerv1.GetCustomerStatusResponse{Status: customerv1.CustomerStatus_CUSTOMER_STATUS_NEW}, f.err
}

func newReq(method, body string, customerID *uuid.UUID) *http.Request {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	r := httptest.NewRequest(method, "/v1/customers/me", rd)
	claims := authjwt.Claims{Subject: uuid.New(), Role: "CLIENT", CustomerID: customerID}
	return r.WithContext(reqctx.WithClaims(r.Context(), claims))
}

func newMe(f *fakeCustomers) *Me {
	return NewMe(f, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
func (f *fakeCustomers) ListCustomers(_ context.Context, in *customerv1.ListCustomersRequest, _ ...grpc.CallOption) (*customerv1.ListCustomersResponse, error) {
	return &customerv1.ListCustomersResponse{}, f.err
}

func TestMe_GetMe_UsesCustomerIDFromClaims(t *testing.T) {
	f := &fakeCustomers{}
	cid := uuid.New()
	rec := httptest.NewRecorder()
	newMe(f).GetMe(rec, newReq(http.MethodGet, "", &cid))

	if rec.Code != 200 || f.gotGet.CustomerId != cid.String() {
		t.Fatalf("status=%d got=%v", rec.Code, f.gotGet)
	}
	var m map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	if m["customerId"] != cid.String() || m["status"] != "CUSTOMER_STATUS_NEW" {
		t.Fatalf("unexpected body: %s", rec.Body)
	}
}

func TestMe_EmptyCustomerID_Returns412(t *testing.T) {
	f := &fakeCustomers{}
	for name, call := range map[string]func(*Me, http.ResponseWriter, *http.Request){
		"GetMe": (*Me).GetMe, "UpdateMe": (*Me).UpdateMe, "GetStatus": (*Me).GetStatus,
	} {
		rec := httptest.NewRecorder()
		call(newMe(f), rec, newReq(http.MethodPatch, `{"fullName":"x"}`, nil))
		if rec.Code != 412 || !strings.Contains(rec.Body.String(), "customer profile not yet linked") {
			t.Errorf("%s: status=%d body=%s", name, rec.Code, rec.Body)
		}
	}
	if f.gotGet != nil || f.gotUpdate != nil || f.gotStatus != nil {
		t.Fatal("downstream must not be called when customer_id is empty")
	}
}

func TestMe_UpdateMe_OverridesCustomerIDFromBody(t *testing.T) {
	f := &fakeCustomers{}
	mine, victim := uuid.New(), uuid.New()
	rec := httptest.NewRecorder()
	body := `{"customerId":"` + victim.String() + `","fullName":"Nino Beridze","phone":"+995555"}`
	newMe(f).UpdateMe(rec, newReq(http.MethodPatch, body, &mine))

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
	if f.gotUpdate.CustomerId != mine.String() {
		t.Fatalf("SECURITY: customer_id from body leaked through: %s", f.gotUpdate.CustomerId)
	}
	if f.gotUpdate.FullName != "Nino Beridze" || f.gotUpdate.Phone != "+995555" {
		t.Fatalf("fields not forwarded: %v", f.gotUpdate)
	}
}

func TestMe_UpdateMe_InvalidBody_Returns400(t *testing.T) {
	cid := uuid.New()
	for _, body := range []string{"", "{not json", `{"unknownField":1}`} {
		rec := httptest.NewRecorder()
		newMe(&fakeCustomers{}).UpdateMe(rec, newReq(http.MethodPatch, body, &cid))
		if rec.Code != 400 {
			t.Errorf("body %q: status=%d", body, rec.Code)
		}
	}
}

func TestMe_DownstreamError_MappedToEnvelope(t *testing.T) {
	cid := uuid.New()
	f := &fakeCustomers{err: status.Error(codes.NotFound, "customer not found")}
	rec := httptest.NewRecorder()
	newMe(f).GetStatus(rec, newReq(http.MethodGet, "", &cid))
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), `"code":"NOT_FOUND"`) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
}

package apierr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

func TestHTTPStatus_AllCodes(t *testing.T) {
	cases := map[codes.Code]int{
		codes.OK:                 200,
		codes.InvalidArgument:    400,
		codes.Unauthenticated:    401,
		codes.PermissionDenied:   403,
		codes.NotFound:           404,
		codes.AlreadyExists:      409,
		codes.Aborted:            409,
		codes.FailedPrecondition: 412,
		codes.ResourceExhausted:  429,
		codes.Unimplemented:      501,
		codes.Unavailable:        503,
		codes.DeadlineExceeded:   504,
		codes.Internal:           500,
		codes.Unknown:            500,
	}
	for c, want := range cases {
		t.Run(CodeName(c), func(t *testing.T) {
			if got := HTTPStatus(c); got != want {
				t.Fatalf("HTTPStatus(%v) = %d, want %d", c, got, want)
			}
		})
	}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("bad json %q: %v", rec.Body.String(), err)
	}
	return m
}

func TestWrite_DetailsAlwaysEmptyArray(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(reqctx.WithInfo(req.Context(), &reqctx.Info{RequestID: "rid-1"}))

	Write(rec, req, status.Error(codes.NotFound, "customer not found"))

	if rec.Code != 404 {
		t.Fatalf("status = %d", rec.Code)
	}
	m := decode(t, rec)
	if m["code"] != "NOT_FOUND" || m["message"] != "customer not found" || m["request_id"] != "rid-1" {
		t.Fatalf("unexpected envelope: %v", m)
	}
	d, ok := m["details"].([]any)
	if !ok || len(d) != 0 {
		t.Fatalf("details must be [], got %#v", m["details"])
	}
}

func TestWrite_InternalMessageIsMasked(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, httptest.NewRequest(http.MethodGet, "/", nil), status.Error(codes.Internal, "pq: password authentication failed"))
	m := decode(t, rec)
	if m["message"] != "internal error" || rec.Code != 500 {
		t.Fatalf("internal details leaked: %v", m)
	}
}

func TestWrite_PlainErrorBecomes500(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, httptest.NewRequest(http.MethodGet, "/", nil), http.ErrAbortHandler)
	if rec.Code != 500 {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWriteStatus_RetryInfoDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteStatus(rec, httptest.NewRequest(http.MethodGet, "/", nil),
		codes.ResourceExhausted, "rate limit exceeded",
		&errdetails.RetryInfo{RetryDelay: durationpb.New(1500e6)})

	m := decode(t, rec)
	d := m["details"].([]any)
	if len(d) != 1 {
		t.Fatalf("details = %v", d)
	}
	first := d[0].(map[string]any)
	if first["@type"] != "type.googleapis.com/google.rpc.RetryInfo" || first["retryDelay"] != "1.500s" {
		t.Fatalf("unexpected detail: %v", first)
	}
}

func TestWrite_RequestIDFallsBackToResponseHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-ID", "from-header")
	Write(rec, httptest.NewRequest(http.MethodGet, "/", nil), status.Error(codes.Internal, "x"))
	if decode(t, rec)["request_id"] != "from-header" {
		t.Fatal("request_id fallback failed")
	}
}

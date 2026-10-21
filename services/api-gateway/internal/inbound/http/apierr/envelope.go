// Package apierr — единый формат ошибок наружу: {code, message, details, request_id}.
package apierr

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	_ "google.golang.org/genproto/googleapis/rpc/errdetails" // регистрирует типы для protojson(Any)
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/reqctx"
)

type Envelope struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   []json.RawMessage `json:"details"` // всегда массив, никогда null
	RequestID string            `json:"request_id"`
}

// Write преобразует любую ошибку (gRPC status, ctx-ошибки, прочее) в envelope.
func Write(w http.ResponseWriter, r *http.Request, err error) {
	st := statusFromError(err)
	msg := st.Message()
	switch st.Code() {
	case codes.Internal, codes.Unknown, codes.DataLoss:
		// Внутренние детали downstream наружу не отдаём.
		slog.ErrorContext(r.Context(), "request failed",
			"grpc_code", CodeName(st.Code()), "error", err, "request_id", requestID(w, r))
		msg = "internal error"
	}
	write(w, r, HTTPStatus(st.Code()), st.Code(), msg, detailsFromStatus(st))
}

// WriteStatus пишет ошибку с явным кодом и опциональными типизированными details
// (например, *errdetails.RetryInfo).
func WriteStatus(w http.ResponseWriter, r *http.Request, code codes.Code, msg string, details ...proto.Message) {
	raw := make([]json.RawMessage, 0, len(details))
	for _, d := range details {
		a, err := anypb.New(d)
		if err != nil {
			continue
		}
		if b, err := protojson.Marshal(a); err == nil {
			raw = append(raw, b)
		}
	}
	write(w, r, HTTPStatus(code), code, msg, raw)
}

// WriteHTTP — для ошибок роутинга, где HTTP-статус задан не gRPC-кодом (напр. 405).
func WriteHTTP(w http.ResponseWriter, r *http.Request, httpStatus int, code codes.Code, msg string) {
	write(w, r, httpStatus, code, msg, nil)
}

func write(w http.ResponseWriter, r *http.Request, httpStatus int, code codes.Code, msg string, details []json.RawMessage) {
	if details == nil {
		details = []json.RawMessage{}
	}
	name := CodeName(code)
	if info := reqctx.InfoFrom(r.Context()); info != nil {
		info.GRPCCode = name
	}
	body, err := json.Marshal(Envelope{
		Code:      name,
		Message:   msg,
		Details:   details,
		RequestID: requestID(w, r),
	})
	if err != nil { // не должно случиться: все поля — строки/RawMessage
		body = []byte(`{"code":"INTERNAL","message":"internal error","details":[],"request_id":""}`)
		httpStatus = http.StatusInternalServerError
	}
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(httpStatus)
	_, _ = w.Write(body)
}

// requestID берёт id из ctx, а если запрос уже «потерял» ctx (recovery стоит снаружи
// requestid-middleware) — из заголовка ответа.
func requestID(w http.ResponseWriter, r *http.Request) string {
	if id := reqctx.RequestID(r.Context()); id != "" {
		return id
	}
	return w.Header().Get("X-Request-ID")
}

func statusFromError(err error) *status.Status {
	if err == nil {
		return status.New(codes.Internal, "internal error")
	}
	if st, ok := status.FromError(err); ok {
		return st
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return status.New(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.New(codes.Canceled, "request canceled")
	}
	return status.New(codes.Internal, "internal error")
}

func detailsFromStatus(st *status.Status) []json.RawMessage {
	pb := st.Proto()
	out := make([]json.RawMessage, 0, len(pb.GetDetails()))
	for _, d := range pb.GetDetails() {
		b, err := protojson.Marshal(d)
		if err != nil { // неизвестный тип — отдаём хотя бы @type
			b, _ = json.Marshal(map[string]string{"@type": d.GetTypeUrl()})
		}
		out = append(out, b)
	}
	return out
}

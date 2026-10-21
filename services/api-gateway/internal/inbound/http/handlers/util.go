package handlers

import (
	"io"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/Mazik-kun/mini-core/services/api-gateway/internal/inbound/http/apierr"
)

var (
	marshalOpts   = protojson.MarshalOptions{EmitUnpopulated: true}
	unmarshalOpts = protojson.UnmarshalOptions{DiscardUnknown: false}
)

// parseProtoJSON читает тело и парсит его в proto-сообщение.
func parseProtoJSON(w http.ResponseWriter, r *http.Request, msg proto.Message) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "request body is required and must be valid JSON")
		return false
	}
	if err := unmarshalOpts.Unmarshal(body, msg); err != nil {
		apierr.WriteStatus(w, r, codes.InvalidArgument, "invalid request body")
		return false
	}
	return true
}

// respondProtoJSON отправляет proto-сообщение в ответ.
func respondProtoJSON(w http.ResponseWriter, r *http.Request, msg proto.Message, err error) {
	if err != nil {
		apierr.Write(w, r, err)
		return
	}
	b, err := marshalOpts.Marshal(msg)
	if err != nil {
		apierr.WriteStatus(w, r, codes.Internal, "internal error")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
package kafka

import (
	"testing"
	"time"
)

type testPayload struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

func TestEnvelopeRoundTrip(t *testing.T) {
	originalPayload := testPayload{OrderID: "order-1", Amount: 1500}

	envelope, err := NewEnvelope("order.created", originalPayload)
	if err != nil {
		t.Fatalf("NewEnvelope вернул ошибку: %v", err)
	}

	if envelope.EventID == "" {
		t.Fatal("EventID пустой")
	}
	if envelope.EventType != "order.created" {
		t.Fatalf("EventType = %q, ожидалось %q", envelope.EventType, "order.created")
	}
	if envelope.OccurredAt.IsZero() {
		t.Fatal("OccurredAt пустой")
	}
	if time.Since(envelope.OccurredAt) > time.Minute {
		t.Fatalf("OccurredAt слишком старый: %v", envelope.OccurredAt)
	}

	var decodedPayload testPayload
	if err := envelope.UnmarshalPayload(&decodedPayload); err != nil {
		t.Fatalf("UnmarshalPayload вернул ошибку: %v", err)
	}
	if decodedPayload != originalPayload {
		t.Fatalf("payload не совпадает: got %+v, want %+v", decodedPayload, originalPayload)
	}
}

func TestEnvelopeEventIDsAreUnique(t *testing.T) {
	first, err := NewEnvelope("order.created", testPayload{OrderID: "a"})
	if err != nil {
		t.Fatalf("NewEnvelope #1: %v", err)
	}
	second, err := NewEnvelope("order.created", testPayload{OrderID: "b"})
	if err != nil {
		t.Fatalf("NewEnvelope #2: %v", err)
	}
	if first.EventID == second.EventID {
		t.Fatalf("EventID совпали: %s", first.EventID)
	}
}

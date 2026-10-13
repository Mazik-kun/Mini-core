package kafka

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Envelope — обёртка события для Kafka.
type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// NewEnvelope создаёт конверт с UUID v7 и сериализованным payload.
func NewEnvelope(eventType string, payload any) (Envelope, error) {
	eventID, err := uuid.NewV7()
	if err != nil {
		return Envelope{}, err
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		EventID:    eventID.String(),
		EventType:  eventType,
		OccurredAt: time.Now().UTC(),
		Payload:    payloadJSON,
	}, nil
}

// UnmarshalPayload разбирает Payload в target.
func (envelope Envelope) UnmarshalPayload(target any) error {
	return json.Unmarshal(envelope.Payload, target)
}
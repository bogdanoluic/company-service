package outbox

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   uuid.UUID
	Type          string
	Payload       json.RawMessage
	OccurredAt    time.Time
}

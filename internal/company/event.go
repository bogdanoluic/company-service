package company

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bogdanoluic/company-service/internal/outbox"
	"github.com/google/uuid"
)

const (
	EventTypeCreated = "company.created"
	EventTypeUpdated = "company.updated"
	EventTypeDeleted = "company.deleted"

	EventAggregateType = "company"
)

type companyEventPayload struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description"`
	AmountOfEmployees int       `json:"amount_of_employees"`
	Registered        bool      `json:"registered"`
	Type              Type      `json:"type"`
	Version           int64     `json:"version"`
}

type companyDeletedEventPayload struct {
	ID uuid.UUID `json:"id"`
}

func newCreatedEvent(
	c Company,
	eventID uuid.UUID,
	occurredAt time.Time,
) (outbox.Event, error) {
	return newEvent(
		eventID,
		c.ID,
		EventTypeCreated,
		companyEventPayloadFrom(c),
		occurredAt,
	)
}

func newUpdatedEvent(
	c Company,
	eventID uuid.UUID,
	occurredAt time.Time,
) (outbox.Event, error) {
	// The repository will increment the stored version.
	updated := c
	updated.Version++

	return newEvent(
		eventID,
		c.ID,
		EventTypeUpdated,
		companyEventPayloadFrom(updated),
		occurredAt,
	)
}

func newDeletedEvent(
	companyID uuid.UUID,
	eventID uuid.UUID,
	occurredAt time.Time,
) (outbox.Event, error) {
	return newEvent(
		eventID,
		companyID,
		EventTypeDeleted,
		companyDeletedEventPayload{
			ID: companyID,
		},
		occurredAt,
	)
}

func newEvent(
	eventID uuid.UUID,
	companyID uuid.UUID,
	eventType string,
	payload any,
	occurredAt time.Time,
) (outbox.Event, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return outbox.Event{}, fmt.Errorf(
			"marshal %s payload: %w",
			eventType,
			err,
		)
	}

	return outbox.Event{
		ID:            eventID,
		AggregateType: EventAggregateType,
		AggregateID:   companyID,
		Type:          eventType,
		Payload:       encodedPayload,
		OccurredAt:    occurredAt.UTC(),
	}, nil
}

func companyEventPayloadFrom(c Company) companyEventPayload {
	return companyEventPayload{
		ID:                c.ID,
		Name:              c.Name,
		Description:       c.Description,
		AmountOfEmployees: c.AmountOfEmployees,
		Registered:        c.Registered,
		Type:              c.Type,
		Version:           c.Version,
	}
}

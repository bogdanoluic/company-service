package postgres

import (
	"context"
	"fmt"

	"github.com/bogdanoluic/company-service/internal/outbox"
	"github.com/jackc/pgx/v5"
)

func insertOutboxEvent(
	ctx context.Context,
	tx pgx.Tx,
	event outbox.Event,
) error {
	const query = `
		INSERT INTO outbox_events (
			id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			occurred_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5::jsonb,
			$6
		)
	`

	_, err := tx.Exec(
		ctx,
		query,
		event.ID,
		event.AggregateType,
		event.AggregateID,
		event.Type,
		string(event.Payload),
		event.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}

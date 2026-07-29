CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,

    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,

    payload JSONB NOT NULL,

    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,

    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,

    CONSTRAINT outbox_events_aggregate_type_non_empty
        CHECK (aggregate_type <> ''),

    CONSTRAINT outbox_events_event_type_non_empty
        CHECK (event_type <> ''),

    CONSTRAINT outbox_events_attempts_non_negative
        CHECK (attempts >= 0)
);

CREATE INDEX outbox_events_pending_idx
    ON outbox_events (available_at, occurred_at)
    WHERE published_at IS NULL;

---- create above / drop below ----

DROP TABLE outbox_events;

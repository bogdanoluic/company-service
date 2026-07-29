//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/bogdanoluic/company-service/internal/company"
	"github.com/bogdanoluic/company-service/internal/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestCompanyRepository(t *testing.T) {
	repository := setupCompanyRepository(t)
	ctx := context.Background()

	t.Run("create and get company", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		description := "Software company"

		want := company.Company{
			ID:                uuid.New(),
			Name:              "XM",
			Description:       &description,
			AmountOfEmployees: 25,
			Registered:        true,
			Type:              company.TypeCorporations,
			Version:           1,
		}

		event := testOutboxEvent(
			company.EventTypeCreated,
			want.ID,
		)

		if err := repository.Create(
			ctx,
			want,
			event,
		); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repository.GetByID(ctx, want.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		assertCompanyEqual(t, got, want)
		assertOutboxEventStored(t, ctx, repository, event)
		assertOutboxEventCount(t, ctx, repository, 1)
	})

	t.Run("nullable description", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		want := company.Company{
			ID:                uuid.New(),
			Name:              "NoDesc",
			Description:       nil,
			AmountOfEmployees: 3,
			Registered:        false,
			Type:              company.TypeNonProfit,
			Version:           1,
		}

		event := testOutboxEvent(
			company.EventTypeCreated,
			want.ID,
		)

		if err := repository.Create(
			ctx,
			want,
			event,
		); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repository.GetByID(ctx, want.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		assertCompanyEqual(t, got, want)
		assertOutboxEventStored(t, ctx, repository, event)
		assertOutboxEventCount(t, ctx, repository, 1)
	})

	t.Run("duplicate name", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		first := testCompany("XM")
		second := testCompany("XM")

		firstEvent := testOutboxEvent(
			company.EventTypeCreated,
			first.ID,
		)

		if err := repository.Create(
			ctx,
			first,
			firstEvent,
		); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		secondEvent := testOutboxEvent(
			company.EventTypeCreated,
			second.ID,
		)

		err := repository.Create(
			ctx,
			second,
			secondEvent,
		)
		if !errors.Is(err, company.ErrNameAlreadyExists) {
			t.Fatalf(
				"second Create() error = %v, want %v",
				err,
				company.ErrNameAlreadyExists,
			)
		}

		assertOutboxEventStored(
			t,
			ctx,
			repository,
			firstEvent,
		)

		assertOutboxEventMissing(
			t,
			ctx,
			repository,
			secondEvent.ID,
		)

		assertOutboxEventCount(t, ctx, repository, 1)
	})

	t.Run("get unknown company", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		_, err := repository.GetByID(ctx, uuid.New())
		if !errors.Is(err, company.ErrNotFound) {
			t.Fatalf(
				"GetByID() error = %v, want %v",
				err,
				company.ErrNotFound,
			)
		}

		assertOutboxEventCount(t, ctx, repository, 0)
	})

	t.Run("update company", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		original := testCompany("XM")

		createEvent := testOutboxEvent(
			company.EventTypeCreated,
			original.ID,
		)

		if err := repository.Create(
			ctx,
			original,
			createEvent,
		); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		description := "Updated description"

		updated := original
		updated.Name = "Trading.com"
		updated.Description = &description
		updated.AmountOfEmployees = 100
		updated.Registered = true
		updated.Type = company.TypeCooperative

		updateEvent := testOutboxEvent(
			company.EventTypeUpdated,
			updated.ID,
		)

		if err := repository.Update(
			ctx,
			updated,
			updateEvent,
		); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		// PostgreSQL increments the version during Update.
		updated.Version++

		got, err := repository.GetByID(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		assertCompanyEqual(t, got, updated)

		assertOutboxEventStored(
			t,
			ctx,
			repository,
			createEvent,
		)

		assertOutboxEventStored(
			t,
			ctx,
			repository,
			updateEvent,
		)

		assertOutboxEventCount(t, ctx, repository, 2)
	})

	t.Run("update unknown company", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		unknown := testCompany("Unknown")

		event := testOutboxEvent(
			company.EventTypeUpdated,
			unknown.ID,
		)

		err := repository.Update(
			ctx,
			unknown,
			event,
		)
		if !errors.Is(err, company.ErrNotFound) {
			t.Fatalf(
				"Update() error = %v, want %v",
				err,
				company.ErrNotFound,
			)
		}

		assertOutboxEventMissing(
			t,
			ctx,
			repository,
			event.ID,
		)

		assertOutboxEventCount(t, ctx, repository, 0)
	})

	t.Run("delete company", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		existing := testCompany("XM")

		createEvent := testOutboxEvent(
			company.EventTypeCreated,
			existing.ID,
		)

		if err := repository.Create(
			ctx,
			existing,
			createEvent,
		); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		deleteEvent := testOutboxEvent(
			company.EventTypeDeleted,
			existing.ID,
		)

		if err := repository.Delete(
			ctx,
			existing.ID,
			deleteEvent,
		); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err := repository.GetByID(ctx, existing.ID)
		if !errors.Is(err, company.ErrNotFound) {
			t.Fatalf(
				"GetByID() after Delete() error = %v, want %v",
				err,
				company.ErrNotFound,
			)
		}

		assertOutboxEventStored(
			t,
			ctx,
			repository,
			createEvent,
		)

		assertOutboxEventStored(
			t,
			ctx,
			repository,
			deleteEvent,
		)

		assertOutboxEventCount(t, ctx, repository, 2)
	})

	t.Run("delete unknown company", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		id := uuid.New()

		event := testOutboxEvent(
			company.EventTypeDeleted,
			id,
		)

		err := repository.Delete(
			ctx,
			id,
			event,
		)
		if !errors.Is(err, company.ErrNotFound) {
			t.Fatalf(
				"Delete() error = %v, want %v",
				err,
				company.ErrNotFound,
			)
		}

		assertOutboxEventMissing(
			t,
			ctx,
			repository,
			event.ID,
		)

		assertOutboxEventCount(t, ctx, repository, 0)
	})

	t.Run("rejects stale update", func(t *testing.T) {
		clearDatabase(t, ctx, repository)

		c := testCompany("XM")

		createEvent := testOutboxEvent(
			company.EventTypeCreated,
			c.ID,
		)

		if err := repository.Create(
			ctx,
			c,
			createEvent,
		); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		firstCopy, err := repository.GetByID(ctx, c.ID)
		if err != nil {
			t.Fatalf("first GetByID() error = %v", err)
		}

		staleCopy, err := repository.GetByID(ctx, c.ID)
		if err != nil {
			t.Fatalf("second GetByID() error = %v", err)
		}

		firstCopy.Name = "First Update"

		firstUpdateEvent := testOutboxEvent(
			company.EventTypeUpdated,
			firstCopy.ID,
		)

		if err := repository.Update(
			ctx,
			firstCopy,
			firstUpdateEvent,
		); err != nil {
			t.Fatalf("first Update() error = %v", err)
		}

		afterFirstUpdate, err := repository.GetByID(
			ctx,
			c.ID,
		)
		if err != nil {
			t.Fatalf(
				"GetByID() after first update error = %v",
				err,
			)
		}

		if afterFirstUpdate.Version != 2 {
			t.Fatalf(
				"version after first update = %d, want 2",
				afterFirstUpdate.Version,
			)
		}

		staleCopy.Name = "Stale Update"

		staleUpdateEvent := testOutboxEvent(
			company.EventTypeUpdated,
			staleCopy.ID,
		)

		err = repository.Update(
			ctx,
			staleCopy,
			staleUpdateEvent,
		)
		if !errors.Is(err, company.ErrConflict) {
			t.Fatalf(
				"second Update() error = %v, want %v",
				err,
				company.ErrConflict,
			)
		}

		assertOutboxEventMissing(
			t,
			ctx,
			repository,
			staleUpdateEvent.ID,
		)

		stored, err := repository.GetByID(ctx, c.ID)
		if err != nil {
			t.Fatalf("final GetByID() error = %v", err)
		}

		if stored.Name != "First Update" {
			t.Errorf(
				"stored name = %q, want %q",
				stored.Name,
				"First Update",
			)
		}

		if stored.Version != 2 {
			t.Errorf(
				"stored version = %d, want 2",
				stored.Version,
			)
		}

		assertOutboxEventStored(
			t,
			ctx,
			repository,
			createEvent,
		)

		assertOutboxEventStored(
			t,
			ctx,
			repository,
			firstUpdateEvent,
		)

		assertOutboxEventCount(t, ctx, repository, 2)
	})
}

func testCompany(name string) company.Company {
	return company.Company{
		ID:                uuid.New(),
		Name:              name,
		Description:       nil,
		AmountOfEmployees: 10,
		Registered:        false,
		Type:              company.TypeCorporations,
		Version:           1,
	}
}

func testOutboxEvent(
	eventType string,
	companyID uuid.UUID,
) outbox.Event {
	return outbox.Event{
		ID:            uuid.New(),
		AggregateType: company.EventAggregateType,
		AggregateID:   companyID,
		Type:          eventType,
		Payload:       json.RawMessage(`{}`),
		OccurredAt:    time.Now().UTC(),
	}
}

func clearDatabase(
	t *testing.T,
	ctx context.Context,
	repository *CompanyRepository,
) {
	t.Helper()

	const query = `
		TRUNCATE TABLE
			outbox_events,
			companies
	`

	if _, err := repository.pool.Exec(ctx, query); err != nil {
		t.Fatalf("clear test database: %v", err)
	}
}

func assertCompanyEqual(
	t *testing.T,
	got company.Company,
	want company.Company,
) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID = %s, want %s", got.ID, want.ID)
	}

	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}

	switch {
	case got.Description == nil && want.Description == nil:
	case got.Description == nil:
		t.Errorf(
			"Description = nil, want %q",
			*want.Description,
		)
	case want.Description == nil:
		t.Errorf(
			"Description = %q, want nil",
			*got.Description,
		)
	case *got.Description != *want.Description:
		t.Errorf(
			"Description = %q, want %q",
			*got.Description,
			*want.Description,
		)
	}

	if got.AmountOfEmployees != want.AmountOfEmployees {
		t.Errorf(
			"AmountOfEmployees = %d, want %d",
			got.AmountOfEmployees,
			want.AmountOfEmployees,
		)
	}

	if got.Registered != want.Registered {
		t.Errorf(
			"Registered = %t, want %t",
			got.Registered,
			want.Registered,
		)
	}

	if got.Type != want.Type {
		t.Errorf(
			"Type = %q, want %q",
			got.Type,
			want.Type,
		)
	}

	if got.Version != want.Version {
		t.Errorf(
			"Version = %d, want %d",
			got.Version,
			want.Version,
		)
	}
}

func assertOutboxEventStored(
	t *testing.T,
	ctx context.Context,
	repository *CompanyRepository,
	want outbox.Event,
) {
	t.Helper()

	const query = `
		SELECT
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			occurred_at,
			published_at,
			attempts,
			last_error
		FROM outbox_events
		WHERE id = $1
	`

	var (
		gotAggregateType string
		gotAggregateID   uuid.UUID
		gotEventType     string
		gotPayload       []byte
		gotOccurredAt    time.Time
		publishedAt      *time.Time
		attempts         int
		lastError        *string
	)

	err := repository.pool.QueryRow(
		ctx,
		query,
		want.ID,
	).Scan(
		&gotAggregateType,
		&gotAggregateID,
		&gotEventType,
		&gotPayload,
		&gotOccurredAt,
		&publishedAt,
		&attempts,
		&lastError,
	)
	if err != nil {
		t.Fatalf("query outbox event: %v", err)
	}

	if gotAggregateType != want.AggregateType {
		t.Errorf(
			"AggregateType = %q, want %q",
			gotAggregateType,
			want.AggregateType,
		)
	}

	if gotAggregateID != want.AggregateID {
		t.Errorf(
			"AggregateID = %s, want %s",
			gotAggregateID,
			want.AggregateID,
		)
	}

	if gotEventType != want.Type {
		t.Errorf(
			"EventType = %q, want %q",
			gotEventType,
			want.Type,
		)
	}

	assertJSONEqual(t, gotPayload, want.Payload)

	if difference := gotOccurredAt.Sub(
		want.OccurredAt,
	).Abs(); difference > time.Millisecond {
		t.Errorf(
			"OccurredAt = %s, want %s; difference = %s",
			gotOccurredAt,
			want.OccurredAt,
			difference,
		)
	}

	if publishedAt != nil {
		t.Errorf(
			"PublishedAt = %s, want nil",
			publishedAt.String(),
		)
	}

	if attempts != 0 {
		t.Errorf("Attempts = %d, want 0", attempts)
	}

	if lastError != nil {
		t.Errorf(
			"LastError = %q, want nil",
			*lastError,
		)
	}
}

func assertOutboxEventMissing(
	t *testing.T,
	ctx context.Context,
	repository *CompanyRepository,
	eventID uuid.UUID,
) {
	t.Helper()

	var count int

	err := repository.pool.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM outbox_events
			WHERE id = $1
		`,
		eventID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query outbox event count: %v", err)
	}

	if count != 0 {
		t.Errorf(
			"outbox event %s exists, want missing",
			eventID,
		)
	}
}

func assertOutboxEventCount(
	t *testing.T,
	ctx context.Context,
	repository *CompanyRepository,
	want int,
) {
	t.Helper()

	var got int

	err := repository.pool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM outbox_events",
	).Scan(&got)
	if err != nil {
		t.Fatalf("count outbox events: %v", err)
	}

	if got != want {
		t.Errorf(
			"outbox event count = %d, want %d",
			got,
			want,
		)
	}
}

func assertJSONEqual(
	t *testing.T,
	got []byte,
	want []byte,
) {
	t.Helper()

	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decode stored payload: %v", err)
	}

	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("decode expected payload: %v", err)
	}

	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf(
			"Payload = %s, want %s",
			got,
			want,
		)
	}
}

func setupCompanyRepository(
	t *testing.T,
) *CompanyRepository {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase(
			"company_service_test",
		),
		tcpostgres.WithUsername(
			"company_service_test",
		),
		tcpostgres.WithPassword(
			"test-password",
		),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf(
			"start PostgreSQL container: %v",
			err,
		)
	}

	testcontainers.CleanupContainer(t, container)

	connectionString, err := container.ConnectionString(
		ctx,
		"sslmode=disable",
	)
	if err != nil {
		t.Fatalf(
			"get PostgreSQL connection string: %v",
			err,
		)
	}

	runMigrations(t, ctx, connectionString)

	pool, err := pgxpool.New(
		ctx,
		connectionString,
	)
	if err != nil {
		t.Fatalf(
			"create PostgreSQL pool: %v",
			err,
		)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		t.Fatalf(
			"ping PostgreSQL: %v",
			err,
		)
	}

	t.Cleanup(pool.Close)

	return NewCompanyRepository(pool)
}

func runMigrations(
	t *testing.T,
	ctx context.Context,
	connectionString string,
) {
	t.Helper()

	conn, err := pgx.Connect(
		ctx,
		connectionString,
	)
	if err != nil {
		t.Fatalf(
			"connect migration client: %v",
			err,
		)
	}

	t.Cleanup(func() {
		if err := conn.Close(
			context.Background(),
		); err != nil {
			t.Errorf(
				"close migration connection: %v",
				err,
			)
		}
	})

	migrator, err := migrate.NewMigrator(
		ctx,
		conn,
		"public.schema_version",
	)
	if err != nil {
		t.Fatalf(
			"create migrator: %v",
			err,
		)
	}

	migrationsPath := filepath.Join(
		"..",
		"..",
		"..",
		"migrations",
	)

	if err := migrator.LoadMigrations(
		os.DirFS(migrationsPath),
	); err != nil {
		t.Fatalf(
			"load migrations: %v",
			err,
		)
	}

	if err := migrator.Migrate(ctx); err != nil {
		t.Fatalf(
			"run migrations: %v",
			err,
		)
	}
}

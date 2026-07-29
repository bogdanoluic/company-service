//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bogdanoluic/company-service/internal/company"
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
		clearCompanies(t, ctx, repository)

		description := "Software company"
		want := company.Company{
			ID:                uuid.New(),
			Name:              "XM",
			Description:       &description,
			AmountOfEmployees: 25,
			Registered:        true,
			Type:              company.TypeCorporations,
		}

		if err := repository.Create(ctx, want); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repository.GetByID(ctx, want.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		assertCompanyEqual(t, got, want)
	})

	t.Run("nullable description", func(t *testing.T) {
		clearCompanies(t, ctx, repository)

		want := company.Company{
			ID:                uuid.New(),
			Name:              "NoDesc",
			Description:       nil,
			AmountOfEmployees: 3,
			Registered:        false,
			Type:              company.TypeNonProfit,
		}

		if err := repository.Create(ctx, want); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repository.GetByID(ctx, want.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		assertCompanyEqual(t, got, want)
	})

	t.Run("duplicate name", func(t *testing.T) {
		clearCompanies(t, ctx, repository)

		first := testCompany("XM")
		second := testCompany("XM")

		if err := repository.Create(ctx, first); err != nil {
			t.Fatalf("first Create() error = %v", err)
		}

		err := repository.Create(ctx, second)
		if !errors.Is(err, company.ErrNameAlreadyExists) {
			t.Fatalf(
				"second Create() error = %v, want %v",
				err,
				company.ErrNameAlreadyExists,
			)
		}
	})

	t.Run("get unknown company", func(t *testing.T) {
		clearCompanies(t, ctx, repository)

		_, err := repository.GetByID(ctx, uuid.New())
		if !errors.Is(err, company.ErrNotFound) {
			t.Fatalf(
				"GetByID() error = %v, want %v",
				err,
				company.ErrNotFound,
			)
		}
	})

	t.Run("update company", func(t *testing.T) {
		clearCompanies(t, ctx, repository)

		original := testCompany("XM")

		if err := repository.Create(ctx, original); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		description := "Updated description"
		updated := company.Company{
			ID:                original.ID,
			Name:              "Trading.com",
			Description:       &description,
			AmountOfEmployees: 100,
			Registered:        true,
			Type:              company.TypeCooperative,
		}

		if err := repository.Update(ctx, updated); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		got, err := repository.GetByID(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}

		assertCompanyEqual(t, got, updated)
	})

	t.Run("update unknown company", func(t *testing.T) {
		clearCompanies(t, ctx, repository)

		unknown := testCompany("Unknown")

		err := repository.Update(ctx, unknown)
		if !errors.Is(err, company.ErrNotFound) {
			t.Fatalf(
				"Update() error = %v, want %v",
				err,
				company.ErrNotFound,
			)
		}
	})

	t.Run("delete company", func(t *testing.T) {
		clearCompanies(t, ctx, repository)

		existing := testCompany("XM")

		if err := repository.Create(ctx, existing); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := repository.Delete(ctx, existing.ID); err != nil {
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
	})

	t.Run("delete unknown company", func(t *testing.T) {
		clearCompanies(t, ctx, repository)

		err := repository.Delete(ctx, uuid.New())
		if !errors.Is(err, company.ErrNotFound) {
			t.Fatalf(
				"Delete() error = %v, want %v",
				err,
				company.ErrNotFound,
			)
		}
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
	}
}

func clearCompanies(
	t *testing.T,
	ctx context.Context,
	repository *CompanyRepository,
) {
	t.Helper()

	if _, err := repository.pool.Exec(ctx, "DELETE FROM companies"); err != nil {
		t.Fatalf("clear companies table: %v", err)
	}
}

func setupCompanyRepository(t *testing.T) *CompanyRepository {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("company_service_test"),
		tcpostgres.WithUsername("company_service_test"),
		tcpostgres.WithPassword("test-password"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL container: %v", err)
	}

	testcontainers.CleanupContainer(t, container)

	connectionString, err := container.ConnectionString(
		ctx,
		"sslmode=disable",
	)
	if err != nil {
		t.Fatalf("get PostgreSQL connection string: %v", err)
	}

	runMigrations(t, ctx, connectionString)

	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		t.Fatalf("create PostgreSQL pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping PostgreSQL: %v", err)
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

	conn, err := pgx.Connect(ctx, connectionString)
	if err != nil {
		t.Fatalf("connect migration client: %v", err)
	}

	t.Cleanup(func() {
		if err := conn.Close(context.Background()); err != nil {
			t.Errorf("close migration connection: %v", err)
		}
	})

	migrator, err := migrate.NewMigrator(
		ctx,
		conn,
		"public.schema_version",
	)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}

	migrationsPath := filepath.Join(
		"..",
		"..",
		"..",
		"migrations",
	)

	if err := migrator.LoadMigrations(os.DirFS(migrationsPath)); err != nil {
		t.Fatalf("load migrations: %v", err)
	}

	if err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("run migrations: %v", err)
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
		t.Errorf("Description = nil, want %q", *want.Description)
	case want.Description == nil:
		t.Errorf("Description = %q, want nil", *got.Description)
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
		t.Errorf("Type = %q, want %q", got.Type, want.Type)
	}
}

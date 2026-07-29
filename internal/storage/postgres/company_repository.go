package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/bogdanoluic/company-service/internal/company"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompanyRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{
		pool: pool,
	}
}

func (r *CompanyRepository) Create(
	ctx context.Context,
	c company.Company,
) error {
	const query = `
		INSERT INTO companies (
			id,
			name,
			description,
			amount_of_employees,
			registered,
			type
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		c.ID,
		c.Name,
		c.Description,
		c.AmountOfEmployees,
		c.Registered,
		c.Type,
	)
	if err != nil {
		if isCompanyNameUniqueViolation(err) {
			return company.ErrNameAlreadyExists
		}

		return fmt.Errorf("insert company: %w", err)
	}

	return nil
}

func (r *CompanyRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (company.Company, error) {
	const query = `
		SELECT
			id,
			name,
			description,
			amount_of_employees,
			registered,
			type
		FROM companies
		WHERE id = $1
	`

	var c company.Company

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.Description,
		&c.AmountOfEmployees,
		&c.Registered,
		&c.Type,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return company.Company{}, company.ErrNotFound
	}

	if err != nil {
		return company.Company{}, fmt.Errorf("select company by ID: %w", err)
	}

	return c, nil
}

func (r *CompanyRepository) Update(
	ctx context.Context,
	c company.Company,
) error {
	const query = `
		UPDATE companies
		SET
			name = $2,
			description = $3,
			amount_of_employees = $4,
			registered = $5,
			type = $6
		WHERE id = $1
	`

	commandTag, err := r.pool.Exec(
		ctx,
		query,
		c.ID,
		c.Name,
		c.Description,
		c.AmountOfEmployees,
		c.Registered,
		c.Type,
	)
	if err != nil {
		if isCompanyNameUniqueViolation(err) {
			return company.ErrNameAlreadyExists
		}

		return fmt.Errorf("update company: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return company.ErrNotFound
	}

	return nil
}

func (r *CompanyRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	const query = `
		DELETE FROM companies
		WHERE id = $1
	`

	commandTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete company: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return company.ErrNotFound
	}

	return nil
}

func isCompanyNameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "companies_name_unique"
}

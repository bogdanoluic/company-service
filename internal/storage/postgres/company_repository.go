package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/bogdanoluic/company-service/internal/company"
	"github.com/bogdanoluic/company-service/internal/outbox"
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

var _ company.Repository = (*CompanyRepository)(nil)

func (r *CompanyRepository) Create(
	ctx context.Context,
	c company.Company,
	event outbox.Event,
) error {
	return withTransaction(
		ctx,
		r.pool,
		func(tx pgx.Tx) error {
			const query = `
				INSERT INTO companies (
					id,
					name,
					description,
					amount_of_employees,
					registered,
					type,
					version
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`

			_, err := tx.Exec(
				ctx,
				query,
				c.ID,
				c.Name,
				c.Description,
				c.AmountOfEmployees,
				c.Registered,
				c.Type,
				c.Version,
			)
			if err != nil {
				if isCompanyNameUniqueViolation(err) {
					return company.ErrNameAlreadyExists
				}

				return fmt.Errorf(
					"insert company: %w",
					err,
				)
			}

			if err := insertOutboxEvent(
				ctx,
				tx,
				event,
			); err != nil {
				return err
			}

			return nil
		},
	)
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
			type,
			version
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
		&c.Version,
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
	event outbox.Event,
) error {
	return withTransaction(
		ctx,
		r.pool,
		func(tx pgx.Tx) error {
			const query = `
		UPDATE companies
		SET
			name = $2,
			description = $3,
			amount_of_employees = $4,
			registered = $5,
			type = $6,
			version = version + 1
		WHERE id = $1
			AND version = $7
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
				c.Version,
			)
			if err != nil {
				if isCompanyNameUniqueViolation(err) {
					return company.ErrNameAlreadyExists
				}

				return fmt.Errorf("update company: %w", err)
			}

			if commandTag.RowsAffected() == 0 {

				exists, err := r.companyExists(ctx, c.ID)
				if err != nil {
					return fmt.Errorf(
						"check company existence after failed update: %w",
						err,
					)
				}

				if !exists {
					return company.ErrNotFound
				}

				return company.ErrConflict
			}

			if err := insertOutboxEvent(
				ctx,
				tx,
				event,
			); err != nil {
				return err
			}

			return nil
		})
}

func (r *CompanyRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
	event outbox.Event,
) error {

	return withTransaction(
		ctx,
		r.pool,
		func(tx pgx.Tx) error {
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

			if err := insertOutboxEvent(
				ctx,
				tx,
				event,
			); err != nil {
				return err
			}

			return nil
		})
}

func isCompanyNameUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "companies_name_unique"
}

func (r *CompanyRepository) companyExists(
	ctx context.Context,
	id uuid.UUID,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM companies
			WHERE id = $1
		)
	`

	var exists bool

	if err := r.pool.QueryRow(
		ctx,
		query,
		id,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf(
			"query company existence: %w",
			err,
		)
	}

	return exists, nil
}

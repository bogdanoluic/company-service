package company

import (
	"context"
	"errors"

	"github.com/bogdanoluic/company-service/internal/outbox"
	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("company not found")
	ErrNameAlreadyExists = errors.New("company name already exists")
	ErrConflict          = errors.New("company was modified concurrently")
)

type Repository interface {
	Create(
		ctx context.Context,
		company Company,
		event outbox.Event,
	) error

	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (Company, error)

	Update(
		ctx context.Context,
		company Company,
		event outbox.Event,
	) error

	Delete(
		ctx context.Context,
		id uuid.UUID,
		event outbox.Event,
	) error
}

package company

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("company not found")
	ErrNameAlreadyExists = errors.New("company name already exists")
)

type Repository interface {
	Create(ctx context.Context, company Company) error
	GetByID(ctx context.Context, id uuid.UUID) (Company, error)
	Update(ctx context.Context, company Company) error
	Delete(ctx context.Context, id uuid.UUID) error
}

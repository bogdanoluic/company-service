package company

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type CreateInput struct {
	Name              string
	Description       *string
	AmountOfEmployees int
	Registered        bool
	Type              Type
}

// PatchField distinguishes between:
//
//   - Set == false: field was omitted
//   - Set == true: field was provided
//
// For Description, Value may also be nil, meaning that the client
// explicitly wants to clear the description.
type PatchField[T any] struct {
	Set   bool
	Value T
}

type PatchInput struct {
	Name              PatchField[string]
	Description       PatchField[*string]
	AmountOfEmployees PatchField[int]
	Registered        PatchField[bool]
	Type              PatchField[Type]
}

func (p PatchInput) Empty() bool {
	return !p.Name.Set &&
		!p.Description.Set &&
		!p.AmountOfEmployees.Set &&
		!p.Registered.Set &&
		!p.Type.Set
}

type Service struct {
	repository Repository
	newID      func() uuid.UUID
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		newID:      uuid.New,
	}
}

func (s *Service) Create(
	ctx context.Context,
	input CreateInput,
) (Company, error) {
	c := Company{
		ID:                s.newID(),
		Name:              input.Name,
		Description:       input.Description,
		AmountOfEmployees: input.AmountOfEmployees,
		Registered:        input.Registered,
		Type:              input.Type,
	}

	if err := validate(c); err != nil {
		return Company{}, err
	}

	if err := s.repository.Create(ctx, c); err != nil {
		return Company{}, err
	}

	return c, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (Company, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Patch(
	ctx context.Context,
	id uuid.UUID,
	input PatchInput,
) (Company, error) {
	if input.Empty() {
		return Company{}, &ValidationError{
			Field:   "body",
			Message: "at least one field must be provided",
		}
	}

	c, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Company{}, err
	}

	if input.Name.Set {
		c.Name = input.Name.Value
	}

	if input.Description.Set {
		c.Description = input.Description.Value
	}

	if input.AmountOfEmployees.Set {
		c.AmountOfEmployees = input.AmountOfEmployees.Value
	}

	if input.Registered.Set {
		c.Registered = input.Registered.Value
	}

	if input.Type.Set {
		c.Type = input.Type.Value
	}

	if err := validate(c); err != nil {
		return Company{}, err
	}

	if err := s.repository.Update(ctx, c); err != nil {
		return Company{}, err
	}

	return c, nil
}

func (s *Service) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	return s.repository.Delete(ctx, id)
}

func validate(c Company) error {
	if strings.TrimSpace(c.Name) == "" {
		return &ValidationError{
			Field:   "name",
			Message: "is required",
		}
	}

	if utf8.RuneCountInString(c.Name) > 15 {
		return &ValidationError{
			Field:   "name",
			Message: "must not exceed 15 characters",
		}
	}

	if c.Description != nil &&
		utf8.RuneCountInString(*c.Description) > 3000 {
		return &ValidationError{
			Field:   "description",
			Message: "must not exceed 3000 characters",
		}
	}

	if c.AmountOfEmployees < 0 {
		return &ValidationError{
			Field:   "amount_of_employees",
			Message: "must not be negative",
		}
	}

	if !c.Type.Valid() {
		return &ValidationError{
			Field:   "type",
			Message: "is invalid",
		}
	}

	return nil
}

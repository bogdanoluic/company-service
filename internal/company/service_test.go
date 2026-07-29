package company

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/bogdanoluic/company-service/internal/outbox"
	"github.com/google/uuid"
)

type fakeRepository struct {
	createFn func(
		context.Context,
		Company,
		outbox.Event,
	) error

	getByIDFn func(
		context.Context,
		uuid.UUID,
	) (Company, error)

	updateFn func(
		context.Context,
		Company,
		outbox.Event,
	) error

	deleteFn func(
		context.Context,
		uuid.UUID,
		outbox.Event,
	) error
}

var _ Repository = (*fakeRepository)(nil)

func (r *fakeRepository) Create(
	ctx context.Context,
	c Company,
	event outbox.Event,
) error {
	if r.createFn == nil {
		panic("unexpected call to Create")
	}

	return r.createFn(ctx, c, event)
}

func (r *fakeRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (Company, error) {
	if r.getByIDFn == nil {
		panic("unexpected call to GetByID")
	}

	return r.getByIDFn(ctx, id)
}

func (r *fakeRepository) Update(
	ctx context.Context,
	c Company,
	event outbox.Event,
) error {
	if r.updateFn == nil {
		panic("unexpected call to Update")
	}

	return r.updateFn(ctx, c, event)
}

func (r *fakeRepository) Delete(
	ctx context.Context,
	id uuid.UUID,
	event outbox.Event,
) error {
	if r.deleteFn == nil {
		panic("unexpected call to Delete")
	}

	return r.deleteFn(ctx, id, event)
}

func TestServiceCreate(t *testing.T) {
	t.Run("creates valid company", func(t *testing.T) {
		fixedID := uuid.MustParse(
			"5ec4a25a-793f-43a1-8912-f9d1ea163117",
		)

		description := "Software company"

		want := Company{
			ID:                fixedID,
			Name:              "XM",
			Description:       &description,
			AmountOfEmployees: 25,
			Registered:        true,
			Type:              TypeCorporations,
			Version:           1,
		}

		var repositoryInput Company

		repository := &fakeRepository{
			createFn: func(
				_ context.Context,
				c Company,
				_ outbox.Event,
			) error {
				repositoryInput = c
				return nil
			},
		}

		service := NewService(repository)
		service.newID = func() uuid.UUID {
			return fixedID
		}

		got, err := service.Create(
			context.Background(),
			CreateInput{
				Name:              "XM",
				Description:       &description,
				AmountOfEmployees: 25,
				Registered:        true,
				Type:              TypeCorporations,
			},
		)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("Create() = %+v, want %+v", got, want)
		}

		if !reflect.DeepEqual(repositoryInput, want) {
			t.Errorf(
				"repository Create() input = %+v, want %+v",
				repositoryInput,
				want,
			)
		}
	})

	t.Run("returns repository error", func(t *testing.T) {
		repositoryError := errors.New("repository failure")

		repository := &fakeRepository{
			createFn: func(
				context.Context,
				Company,
				outbox.Event,
			) error {
				return repositoryError
			},
		}

		service := NewService(repository)

		_, err := service.Create(
			context.Background(),
			CreateInput{
				Name:              "XM",
				AmountOfEmployees: 10,
				Type:              TypeCorporations,
			},
		)

		if !errors.Is(err, repositoryError) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				repositoryError,
			)
		}
	})
}

func TestServiceCreateRejectsInvalidInput(t *testing.T) {
	longDescription := strings.Repeat("a", 3001)

	tests := []struct {
		name      string
		input     CreateInput
		wantField string
	}{
		{
			name: "empty name",
			input: CreateInput{
				Name:              "   ",
				AmountOfEmployees: 10,
				Type:              TypeCorporations,
			},
			wantField: "name",
		},
		{
			name: "name too long",
			input: CreateInput{
				Name:              "1234567890123456",
				AmountOfEmployees: 10,
				Type:              TypeCorporations,
			},
			wantField: "name",
		},
		{
			name: "description too long",
			input: CreateInput{
				Name:              "XM",
				Description:       &longDescription,
				AmountOfEmployees: 10,
				Type:              TypeCorporations,
			},
			wantField: "description",
		},
		{
			name: "negative employee amount",
			input: CreateInput{
				Name:              "XM",
				AmountOfEmployees: -1,
				Type:              TypeCorporations,
			},
			wantField: "amount_of_employees",
		},
		{
			name: "invalid company type",
			input: CreateInput{
				Name:              "XM",
				AmountOfEmployees: 10,
				Type:              Type("Unknown"),
			},
			wantField: "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryCalled := false

			repository := &fakeRepository{
				createFn: func(
					context.Context,
					Company,
					outbox.Event,
				) error {
					repositoryCalled = true
					return nil
				},
			}

			service := NewService(repository)

			_, err := service.Create(
				context.Background(),
				tt.input,
			)
			if err == nil {
				t.Fatal("Create() error = nil, want validation error")
			}

			var validationError *ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf(
					"Create() error = %T, want *ValidationError",
					err,
				)
			}

			if validationError.Field != tt.wantField {
				t.Errorf(
					"validation field = %q, want %q",
					validationError.Field,
					tt.wantField,
				)
			}

			if repositoryCalled {
				t.Error(
					"repository Create() called for invalid input",
				)
			}
		})
	}
}

func TestServiceGetByID(t *testing.T) {
	id := uuid.New()
	want := testServiceCompany(id, "XM")

	repository := &fakeRepository{
		getByIDFn: func(
			_ context.Context,
			gotID uuid.UUID,
		) (Company, error) {
			if gotID != id {
				t.Errorf("GetByID() ID = %s, want %s", gotID, id)
			}

			return want, nil
		},
	}

	service := NewService(repository)

	got, err := service.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetByID() = %+v, want %+v", got, want)
	}
}

func TestServicePatch(t *testing.T) {
	t.Run("updates only provided fields", func(t *testing.T) {
		id := uuid.New()

		description := "Original description"
		existing := testServiceCompany(id, "XM")
		existing.Description = &description
		existing.AmountOfEmployees = 25
		existing.Registered = true
		existing.Version = 1

		wantRepositoryUpdate := existing
		wantRepositoryUpdate.Description = nil
		wantRepositoryUpdate.AmountOfEmployees = 0
		wantRepositoryUpdate.Registered = false
		wantRepositoryUpdate.Type = TypeNonProfit

		wantResult := wantRepositoryUpdate
		wantResult.Version = 2

		var repositoryUpdate Company

		repository := &fakeRepository{
			getByIDFn: func(
				_ context.Context,
				gotID uuid.UUID,
			) (Company, error) {
				if gotID != id {
					t.Errorf(
						"GetByID() ID = %s, want %s",
						gotID,
						id,
					)
				}

				return existing, nil
			},
			updateFn: func(
				_ context.Context,
				c Company,
				_ outbox.Event,
			) error {
				repositoryUpdate = c
				return nil
			},
		}

		service := NewService(repository)

		got, err := service.Patch(
			context.Background(),
			id,
			PatchInput{
				Description: PatchField[*string]{
					Set:   true,
					Value: nil,
				},
				AmountOfEmployees: PatchField[int]{
					Set:   true,
					Value: 0,
				},
				Registered: PatchField[bool]{
					Set:   true,
					Value: false,
				},
				Type: PatchField[Type]{
					Set:   true,
					Value: TypeNonProfit,
				},
			},
		)
		if err != nil {
			t.Fatalf("Patch() error = %v", err)
		}

		if !reflect.DeepEqual(repositoryUpdate, wantRepositoryUpdate) {
			t.Errorf(
				"repository Update() input = %+v, want %+v",
				repositoryUpdate,
				wantRepositoryUpdate,
			)
		}

		if !reflect.DeepEqual(got, wantResult) {
			t.Errorf(
				"Patch() = %+v, want %+v",
				got,
				wantResult,
			)
		}
	})

	t.Run("rejects empty patch", func(t *testing.T) {
		service := NewService(&fakeRepository{})

		_, err := service.Patch(
			context.Background(),
			uuid.New(),
			PatchInput{},
		)
		if err == nil {
			t.Fatal("Patch() error = nil, want validation error")
		}

		var validationError *ValidationError
		if !errors.As(err, &validationError) {
			t.Fatalf(
				"Patch() error = %T, want *ValidationError",
				err,
			)
		}

		if validationError.Field != "body" {
			t.Errorf(
				"validation field = %q, want %q",
				validationError.Field,
				"body",
			)
		}
	})

	t.Run("does not update invalid merged company", func(t *testing.T) {
		id := uuid.New()
		existing := testServiceCompany(id, "XM")

		updateCalled := false

		repository := &fakeRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (Company, error) {
				return existing, nil
			},
			updateFn: func(
				context.Context,
				Company,
				outbox.Event,
			) error {
				updateCalled = true
				return nil
			},
		}

		service := NewService(repository)

		_, err := service.Patch(
			context.Background(),
			id,
			PatchInput{
				Name: PatchField[string]{
					Set:   true,
					Value: "",
				},
			},
		)
		if err == nil {
			t.Fatal("Patch() error = nil, want validation error")
		}

		var validationError *ValidationError
		if !errors.As(err, &validationError) {
			t.Fatalf(
				"Patch() error = %T, want *ValidationError",
				err,
			)
		}

		if validationError.Field != "name" {
			t.Errorf(
				"validation field = %q, want %q",
				validationError.Field,
				"name",
			)
		}

		if updateCalled {
			t.Error("repository Update() called for invalid company")
		}
	})

	t.Run("returns get error", func(t *testing.T) {
		repositoryError := errors.New("get failure")

		repository := &fakeRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (Company, error) {
				return Company{}, repositoryError
			},
		}

		service := NewService(repository)

		_, err := service.Patch(
			context.Background(),
			uuid.New(),
			PatchInput{
				Name: PatchField[string]{
					Set:   true,
					Value: "Trading.com",
				},
			},
		)

		if !errors.Is(err, repositoryError) {
			t.Fatalf(
				"Patch() error = %v, want %v",
				err,
				repositoryError,
			)
		}
	})

	t.Run("returns update error", func(t *testing.T) {
		id := uuid.New()
		existing := testServiceCompany(id, "XM")
		repositoryError := errors.New("update failure")

		repository := &fakeRepository{
			getByIDFn: func(
				context.Context,
				uuid.UUID,
			) (Company, error) {
				return existing, nil
			},
			updateFn: func(
				context.Context,
				Company,
				outbox.Event,
			) error {
				return repositoryError
			},
		}

		service := NewService(repository)

		_, err := service.Patch(
			context.Background(),
			id,
			PatchInput{
				Name: PatchField[string]{
					Set:   true,
					Value: "Trading.com",
				},
			},
		)

		if !errors.Is(err, repositoryError) {
			t.Fatalf(
				"Patch() error = %v, want %v",
				err,
				repositoryError,
			)
		}
	})
}

func TestServiceDelete(t *testing.T) {
	id := uuid.New()
	repositoryError := errors.New("delete failure")

	repository := &fakeRepository{
		deleteFn: func(
			_ context.Context,
			gotID uuid.UUID,
			_ outbox.Event,
		) error {
			if gotID != id {
				t.Errorf("Delete() ID = %s, want %s", gotID, id)
			}

			return repositoryError
		},
	}

	service := NewService(repository)

	err := service.Delete(context.Background(), id)
	if !errors.Is(err, repositoryError) {
		t.Fatalf(
			"Delete() error = %v, want %v",
			err,
			repositoryError,
		)
	}
}

func TestValidateCountsUnicodeCharacters(t *testing.T) {
	repository := &fakeRepository{
		createFn: func(context.Context, Company, outbox.Event) error {
			return nil
		},
	}

	service := NewService(repository)

	_, err := service.Create(
		context.Background(),
		CreateInput{
			Name:              strings.Repeat("Ž", 15),
			AmountOfEmployees: 10,
			Type:              TypeCorporations,
		},
	)
	if err != nil {
		t.Fatalf(
			"Create() error = %v for a 15-character Unicode name",
			err,
		)
	}
}

func testServiceCompany(id uuid.UUID, name string) Company {
	return Company{
		ID:                id,
		Name:              name,
		Description:       nil,
		AmountOfEmployees: 10,
		Registered:        false,
		Type:              TypeCorporations,
		Version:           1,
	}
}

// Package usecase contiene los casos de uso de testdesign.
package usecase

import (
	"context"
	"errors"
	"slices"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionManage es el permiso para definir UC y TC.
const PermissionManage = "testdesign.manage"

// Errores de los casos de uso.
var (
	ErrFunctionalCaseNotFound = axis.New(axis.NotFoundResource, "El UC no existe.")
	ErrTestCaseNotFound       = axis.New(axis.NotFoundResource, "El TC no existe.")
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository   port.Repository
	Requirements port.Requirements
	Components   port.Components
	Permissions  port.Permissions
	AuditTrail   port.AuditTrail
	UnitOfWork   uow.Factory
	Clock        clock.Clock
	IdGenerator  idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateFunctionalCase     *CreateFunctionalCase
	GetFunctionalCase        *GetFunctionalCase
	ListFunctionalCases      *ListFunctionalCases
	UpdateFunctionalCase     *UpdateFunctionalCase
	CreateTestCase           *CreateTestCase
	GetTestCase              *GetTestCase
	ListTestCases            *ListTestCases
	UpdateTestCase           *UpdateTestCase
	ListRequirementTestCases *ListRequirementTestCases
	ListTestCaseComponents   *ListTestCaseComponents
	SetTestCaseComponents    *SetTestCaseComponents
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Requirements == nil || d.Components == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("testdesign: faltan dependencias")
	}
	return &UseCases{
		CreateFunctionalCase:     &CreateFunctionalCase{d: d},
		GetFunctionalCase:        &GetFunctionalCase{d: d},
		ListFunctionalCases:      &ListFunctionalCases{d: d},
		UpdateFunctionalCase:     &UpdateFunctionalCase{d: d},
		CreateTestCase:           &CreateTestCase{d: d},
		GetTestCase:              &GetTestCase{d: d},
		ListTestCases:            &ListTestCases{d: d},
		UpdateTestCase:           &UpdateTestCase{d: d},
		ListRequirementTestCases: &ListRequirementTestCases{d: d},
		ListTestCaseComponents:   &ListTestCaseComponents{d: d},
		SetTestCaseComponents:    &SetTestCaseComponents{d: d},
	}, nil
}

func findFunctionalCase(ctx context.Context, repo port.Repository, raw string) (domain.FunctionalCase, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.FunctionalCase{}, ErrFunctionalCaseNotFound
	}
	functionalCase, err := repo.FindFunctionalCase(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.FunctionalCase{}, ErrFunctionalCaseNotFound
	}
	return functionalCase, err
}

func findTestCase(ctx context.Context, repo port.Repository, raw string) (domain.TestCase, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.TestCase{}, ErrTestCaseNotFound
	}
	testCase, err := repo.FindTestCase(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.TestCase{}, ErrTestCaseNotFound
	}
	return testCase, err
}

// resolveFunctionalCases busca los UC por código; si alguno no existe, lo dice en el campo.
func resolveFunctionalCases(ctx context.Context, repo port.Repository, codes []string) ([]domain.FunctionalCaseRef, error) {
	refs, err := repo.FunctionalCaseRefs(ctx, codes)
	if err != nil {
		return nil, err
	}
	for _, code := range codes {
		if !slices.ContainsFunc(refs, func(r domain.FunctionalCaseRef) bool { return r.Code == code }) {
			return nil, axis.Invalid(axis.InvalidField("functionalCases", "El UC "+code+" no existe."))
		}
	}
	return refs, nil
}

func recordChanges(ctx context.Context, d Deps, actor shared.UserId, entityType, id string, code shared.Code, changes []domain.FieldChange) error {
	for _, c := range changes {
		if err := d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: actor, EntityType: entityType, EntityId: id, Code: code, Action: port.AuditUpdate,
			Field: c.Field, OldValue: c.Old, NewValue: c.New,
		}); err != nil {
			return err
		}
	}
	return nil
}

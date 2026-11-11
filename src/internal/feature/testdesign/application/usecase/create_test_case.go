package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateTestCase define un TC que valida uno o más UC.
type CreateTestCase struct {
	d Deps
}

// CreateTestCaseInput es la entrada de CreateTestCase.
type CreateTestCaseInput struct {
	Actor           shared.Actor
	Title           string
	SystemKey       string
	Preconditions   string
	Steps           string
	ExpectedResult  string
	IsCritical      bool
	FunctionalCases []string
}

// Execute exige el permiso testdesign.manage.
func (uc *CreateTestCase) Execute(ctx context.Context, in CreateTestCaseInput) (domain.TestCaseSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return domain.TestCaseSnapshot{}, err
	}
	systemName, err := uc.d.Repository.FindSystem(ctx, in.SystemKey)
	if axis.IsCode(err, axis.NotFoundResource) {
		field := axis.InvalidField("system", "El sistema no existe.")
		if in.SystemKey == "" {
			field = axis.Required("system", "Elige un sistema.")
		}
		return domain.TestCaseSnapshot{}, axis.Invalid(field)
	}
	if err != nil {
		return domain.TestCaseSnapshot{}, err
	}
	refs, err := resolveFunctionalCases(ctx, uc.d.Repository, in.FunctionalCases)
	if err != nil {
		return domain.TestCaseSnapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.TestCaseSnapshot{}, err
	}
	now := uc.d.Clock.Now()
	var out domain.TestCaseSnapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixTestCase, now.Year())
		if err != nil {
			return err
		}
		testCase, err := domain.NewTestCase(domain.NewTestCaseParams{
			Id: id, Code: code, SystemKey: in.SystemKey, SystemName: systemName, Title: in.Title,
			Preconditions: in.Preconditions, Steps: in.Steps, ExpectedResult: in.ExpectedResult,
			IsCritical: in.IsCritical, FunctionalCases: refs, CreatedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertTestCase(ctx, testCase); err != nil {
			return err
		}
		out = testCase.Snapshot()
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityTestCase, EntityId: id, Code: code,
			Action: port.AuditCreate, NewValue: out.Title,
		})
	})
	if err != nil {
		return domain.TestCaseSnapshot{}, err
	}
	return out, nil
}

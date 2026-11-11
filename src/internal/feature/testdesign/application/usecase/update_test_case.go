package usecase

import (
	"context"

	"kaoru/internal/feature/testdesign/application/port"
	"kaoru/internal/feature/testdesign/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateTestCase edita o retira un TC y audita cada campo que cambió.
type UpdateTestCase struct {
	d Deps
}

// UpdateTestCaseInput es la entrada de UpdateTestCase; nil deja el campo como está.
type UpdateTestCaseInput struct {
	Actor          shared.Actor
	Code           string
	Title          *string
	Preconditions  *string
	Steps          *string
	ExpectedResult *string
	IsCritical     *bool
	// FunctionalCases reemplaza los UC que valida el TC.
	FunctionalCases *[]string
	Retired         *bool
}

// Execute exige el permiso testdesign.manage.
func (uc *UpdateTestCase) Execute(ctx context.Context, in UpdateTestCaseInput) (domain.TestCaseSnapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return domain.TestCaseSnapshot{}, err
	}
	edit := domain.TestCaseEdit{
		Title: in.Title, Preconditions: in.Preconditions, Steps: in.Steps, ExpectedResult: in.ExpectedResult,
		IsCritical: in.IsCritical, Retired: in.Retired,
	}
	if in.FunctionalCases != nil {
		refs, err := resolveFunctionalCases(ctx, uc.d.Repository, *in.FunctionalCases)
		if err != nil {
			return domain.TestCaseSnapshot{}, err
		}
		edit.FunctionalCases = &refs
	}
	var out domain.TestCaseSnapshot
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		testCase, err := findTestCase(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		changes, err := testCase.Apply(edit, uc.d.Clock.Now())
		out = testCase.Snapshot()
		if err != nil || len(changes) == 0 {
			return err
		}
		if err := uc.d.Repository.UpdateTestCase(ctx, testCase); err != nil {
			return err
		}
		return recordChanges(ctx, uc.d, in.Actor.Id, port.EntityTestCase, out.Id, out.Code, changes)
	})
	if err != nil {
		return domain.TestCaseSnapshot{}, err
	}
	return out, nil
}

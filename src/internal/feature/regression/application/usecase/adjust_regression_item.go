package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/regression/application/port"
	"kaoru/internal/feature/regression/domain"
	shared "kaoru/internal/shared/domain"
)

// AdjustRegressionItem incluye o excluye un TC de la regresión de un PKG, con el criterio de QA.
type AdjustRegressionItem struct {
	d Deps
}

// AdjustRegressionItemInput es la entrada de AdjustRegressionItem.
type AdjustRegressionItemInput struct {
	Actor        shared.Actor
	PackageCode  string
	TestCaseCode string
	// Included es nil si la petición no dijo si el TC queda incluido.
	Included *bool
	Reason   string
}

// Execute exige el permiso regression.manage. Devuelve NOT_FOUND_RESOURCE si el PKG o el TC no existen
// o si la regresión todavía no se propuso, INVALID_ACTION si ya está confirmada o si se quiere incluir
// un TC retirado o de otro sistema, e INVALID_REQUEST si falta el criterio que la decisión exige.
func (uc *AdjustRegressionItem) Execute(ctx context.Context, in AdjustRegressionItemInput) (RegressionOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return RegressionOutput{}, err
	}
	if in.Included == nil {
		return RegressionOutput{}, axis.Invalid(axis.Required("included", "Indica si el TC queda incluido."))
	}
	var out RegressionOutput
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		pkg, err := lockPackage(ctx, uc.d.Repository, in.PackageCode)
		if err != nil {
			return err
		}
		view, err := findSelection(ctx, uc.d.Repository, pkg.Id)
		if err != nil {
			return err
		}
		if view.Selection.Confirmed() {
			return domain.ErrConfirmed
		}
		testCase, err := findTestCase(ctx, uc.d.Repository, in.TestCaseCode)
		if err != nil {
			return err
		}
		var current *domain.Item
		for i := range view.Items {
			if view.Items[i].Item.TestCase.Id == testCase.Ref.Id {
				current = &view.Items[i].Item
			}
		}
		adjustment, err := domain.Adjust(current, testCase, pkg.SystemKey, domain.Decision{
			Included: *in.Included, Reason: in.Reason, By: in.Actor.Id, At: uc.d.Clock.Now(),
		})
		if err != nil {
			return err
		}
		if !adjustment.Changed {
			out = toOutput(pkg, view)
			return nil
		}
		record := port.AuditRecord{
			Actor: in.Actor.Id, PackageId: pkg.Id, Code: pkg.Code, Action: port.AuditDecision, Field: fieldItem,
			OldValue: describe(testCase.Ref, current), Reason: adjustment.Item.Reason,
		}
		if adjustment.Removed {
			record.NewValue = describe(testCase.Ref, nil)
			err = uc.d.Repository.DeleteItem(ctx, pkg.Id, testCase.Ref.Id)
		} else {
			record.NewValue = describe(testCase.Ref, &adjustment.Item)
			err = uc.d.Repository.SaveItem(ctx, pkg.Id, adjustment.Item)
		}
		if err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, record); err != nil {
			return err
		}
		out, err = readOutput(ctx, uc.d.Repository, pkg)
		return err
	})
	if err != nil {
		return RegressionOutput{}, err
	}
	return out, nil
}

// findTestCase valida el código y lee el TC; un código mal formado tampoco existe.
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

// describe dice, para la auditoría, cómo está un TC en la selección; item es nil si no está.
func describe(testCase domain.TestCaseRef, item *domain.Item) string {
	switch {
	case item == nil:
		return testCase.Code + ": absent"
	case item.Included:
		return testCase.Code + ": included"
	default:
		return testCase.Code + ": excluded"
	}
}

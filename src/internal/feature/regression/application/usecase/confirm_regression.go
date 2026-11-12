package usecase

import (
	"context"

	"kaoru/internal/feature/regression/application/port"
	shared "kaoru/internal/shared/domain"
)

// ConfirmRegression fija la selección de regresión de un PKG: desde entonces no cambia.
type ConfirmRegression struct {
	d Deps
}

// ConfirmRegressionInput es la entrada de ConfirmRegression.
type ConfirmRegressionInput struct {
	Actor       shared.Actor
	PackageCode string
}

// Execute exige el permiso regression.manage. Devuelve NOT_FOUND_RESOURCE si la regresión todavía
// no se propuso e INVALID_ACTION si ya estaba confirmada.
func (uc *ConfirmRegression) Execute(ctx context.Context, in ConfirmRegressionInput) (RegressionOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return RegressionOutput{}, err
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
		selection := view.Selection
		if err := selection.Confirm(in.Actor.Id, uc.d.Clock.Now()); err != nil {
			return err
		}
		if err := uc.d.Repository.SaveSelection(ctx, selection); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, PackageId: pkg.Id, Code: pkg.Code, Action: port.AuditDecision,
			Field: fieldConfirmed, OldValue: "false", NewValue: "true",
		}); err != nil {
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

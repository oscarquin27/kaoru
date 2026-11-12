package usecase

import (
	"context"
	"strconv"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/regression/application/port"
	"kaoru/internal/feature/regression/domain"
	shared "kaoru/internal/shared/domain"
)

// ProposeRegression propone la regresión de un PKG por impacto: los TC activos que ejercitan algún
// componente tocado por sus REQ, más los críticos del sistema. Volver a proponer conserva los ajustes de QA.
type ProposeRegression struct {
	d Deps
}

// ProposeRegressionInput es la entrada de ProposeRegression.
type ProposeRegressionInput struct {
	Actor       shared.Actor
	PackageCode string
}

// Execute exige el permiso regression.manage. Devuelve INVALID_ACTION si la selección ya está confirmada.
func (uc *ProposeRegression) Execute(ctx context.Context, in ProposeRegressionInput) (RegressionOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return RegressionOutput{}, err
	}
	var out RegressionOutput
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		pkg, err := lockPackage(ctx, uc.d.Repository, in.PackageCode)
		if err != nil {
			return err
		}
		now := uc.d.Clock.Now()
		record := port.AuditRecord{Actor: in.Actor.Id, PackageId: pkg.Id, Code: pkg.Code, Action: port.AuditUpdate, Field: fieldProposed}
		view, err := uc.d.Repository.FindSelection(ctx, pkg.Id)
		selection := view.Selection
		switch {
		case axis.IsCode(err, axis.NotFoundResource):
			selection, record.Action = domain.NewSelection(pkg.Id, in.Actor.Id, now), port.AuditCreate
		case err != nil:
			return err
		default:
			if err := selection.Propose(in.Actor.Id, now); err != nil {
				return err
			}
			record.OldValue = strconv.Itoa(countProposed(items(view)))
		}
		candidates, err := uc.d.Repository.ListCandidates(ctx, pkg.Id)
		if err != nil {
			return err
		}
		next := domain.Propose(items(view), candidates)
		if err := uc.d.Repository.SaveSelection(ctx, selection); err != nil {
			return err
		}
		if err := uc.d.Repository.ReplaceItems(ctx, pkg.Id, next); err != nil {
			return err
		}
		record.NewValue = strconv.Itoa(countProposed(next))
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

func countProposed(items []domain.Item) int {
	n := 0
	for _, item := range items {
		if item.Proposed {
			n++
		}
	}
	return n
}

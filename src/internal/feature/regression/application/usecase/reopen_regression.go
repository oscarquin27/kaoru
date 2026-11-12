package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/regression/application/port"
	shared "kaoru/internal/shared/domain"
)

// ReopenRegression quita la confirmación de la selección de un PKG. No tiene ruta: lo llama el
// paquete dentro de su transacción cuando cambia su contenido o vuelve a armado, porque la selección
// confirmada ya no corresponde. El permiso lo comprobó el caso de uso del paquete.
type ReopenRegression struct {
	d Deps
}

// ReopenRegressionInput es la entrada de ReopenRegression.
type ReopenRegressionInput struct {
	ActorId     shared.UserId
	PackageCode string
	// Reason es por qué la selección deja de valer; queda en la auditoría.
	Reason string
}

// Execute no hace nada si el PKG no tiene selección o no está confirmada.
func (uc *ReopenRegression) Execute(ctx context.Context, in ReopenRegressionInput) error {
	return uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		pkg, err := lockPackage(ctx, uc.d.Repository, in.PackageCode)
		if err != nil {
			return err
		}
		view, err := uc.d.Repository.FindSelection(ctx, pkg.Id)
		if axis.IsCode(err, axis.NotFoundResource) {
			return nil
		}
		if err != nil {
			return err
		}
		selection := view.Selection
		if !selection.Reopen() {
			return nil
		}
		if err := uc.d.Repository.SaveSelection(ctx, selection); err != nil {
			return err
		}
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.ActorId, PackageId: pkg.Id, Code: pkg.Code, Action: port.AuditUpdate,
			Field: fieldConfirmed, OldValue: "true", NewValue: "false", Reason: in.Reason,
		})
	})
}

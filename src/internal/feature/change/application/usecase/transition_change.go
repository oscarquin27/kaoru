package usecase

import (
	"context"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/change/application/port"
	shared "kaoru/internal/shared/domain"
)

// TransitionChange cambia el estado de un CHG según su flujo configurado.
type TransitionChange struct {
	d Deps
}

// TransitionChangeInput es la entrada de TransitionChange.
type TransitionChangeInput struct {
	Actor  shared.Actor
	Code   string
	To     string
	Reason string
}

// Execute devuelve INVALID_ACTION si el flujo no permite el paso, PERMISSION_DENIED si el
// rol no tiene el permiso de la transición e INVALID_REQUEST si falta el motivo que exige.
func (uc *TransitionChange) Execute(ctx context.Context, in TransitionChangeInput) (ChangeOutput, error) {
	if in.To == "" {
		return ChangeOutput{}, axis.Invalid(axis.Required("to", "Indica el estado destino."))
	}
	reason := strings.TrimSpace(in.Reason)
	var view port.ChangeView
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		var err error
		if view, err = lockChange(ctx, uc.d.Repository, in.Code); err != nil {
			return err
		}
		change := view.Change
		transition, err := uc.d.Workflow.ResolveTransition(ctx, change.Snapshot().State.Key, in.To, in.Actor.Roles)
		if err != nil {
			return err
		}
		if transition.RequiresReason && reason == "" {
			return axis.Invalid(axis.Required("reason", "Escribe el motivo."))
		}
		moved := change.MoveTo(transition.To, uc.d.Clock.Now())
		if err := uc.d.Repository.Update(ctx, change); err != nil {
			return err
		}
		snapshot := change.Snapshot()
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, ChangeId: snapshot.Id, Code: snapshot.Code, Action: port.AuditTransition,
			Field: moved.Field, OldValue: moved.Old, NewValue: moved.New, Reason: reason,
		}); err != nil {
			return err
		}
		view.Change = change
		return nil
	})
	return toOutput(view), err
}

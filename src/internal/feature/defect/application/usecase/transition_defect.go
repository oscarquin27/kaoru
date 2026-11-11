package usecase

import (
	"context"
	"fmt"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/defect/application/port"
	"kaoru/internal/feature/defect/domain"
	shared "kaoru/internal/shared/domain"
)

// TransitionDefect cambia el estado de un DEF si el flujo, el rol y las reglas lo permiten.
// Quién aceptó diferirlo y por qué queda en la auditoría de la transición.
type TransitionDefect struct {
	d Deps
}

// TransitionDefectInput es la entrada de TransitionDefect.
type TransitionDefectInput struct {
	Actor  shared.Actor
	Code   string
	To     string
	Reason string
}

// Execute devuelve INVALID_ACTION si el flujo o una regla impiden el paso, PERMISSION_DENIED
// si el rol no tiene el permiso de la transición e INVALID_REQUEST si falta el motivo que exige.
func (uc *TransitionDefect) Execute(ctx context.Context, in TransitionDefectInput) (DefectOutput, error) {
	if in.To == "" {
		return DefectOutput{}, axis.Invalid(axis.Required("to", "Indica el estado destino."))
	}
	reason := strings.TrimSpace(in.Reason)
	var defect domain.Defect
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		var err error
		if defect, err = findDefect(ctx, uc.d.Repository, in.Code); err != nil {
			return err
		}
		transition, err := uc.d.Workflow.ResolveTransition(ctx, defect.Snapshot().State.Key, in.To, in.Actor.Roles)
		if err != nil {
			return err
		}
		if transition.RequiresReason && reason == "" {
			return axis.Invalid(axis.Required("reason", "Escribe el motivo."))
		}
		if blockers := defect.Blockers(transition.Rules); len(blockers) > 0 {
			return axis.New(axis.InvalidAction,
				fmt.Sprintf("No se puede pasar a «%s». %s", transition.To.Name, strings.Join(blockers, " ")))
		}
		moved := defect.MoveTo(transition.To, uc.d.Clock.Now())
		if err := uc.d.Repository.Update(ctx, defect); err != nil {
			return err
		}
		s := defect.Snapshot()
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, DefectId: s.Id, Code: s.Code, Action: port.AuditTransition,
			Field: moved.Field, OldValue: moved.Old, NewValue: moved.New, Reason: reason,
		})
	})
	if err != nil {
		return DefectOutput{}, err
	}
	// Acaba de cambiar de estado: lleva cero días en él.
	return DefectOutput{Defect: defect.Snapshot()}, nil
}

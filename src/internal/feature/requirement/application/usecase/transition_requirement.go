package usecase

import (
	"context"
	"fmt"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// TransitionRequirement cambia el estado de un REQ si el flujo, el rol y las reglas
// de bloqueo lo permiten.
type TransitionRequirement struct {
	d Deps
}

// TransitionRequirementInput es la entrada de TransitionRequirement.
type TransitionRequirementInput struct {
	Actor  shared.Actor
	Code   string
	To     string
	Reason string
}

// Execute devuelve INVALID_ACTION si el flujo no permite el paso o si una regla de bloqueo
// no se cumple (con lo que falta en el mensaje), PERMISSION_DENIED si el rol no tiene el
// permiso de la transición e INVALID_REQUEST si falta el motivo que exige.
func (uc *TransitionRequirement) Execute(ctx context.Context, in TransitionRequirementInput) (domain.Snapshot, error) {
	if in.To == "" {
		return domain.Snapshot{}, axis.Invalid(axis.Required("to", "Indica el estado destino."))
	}
	reason := strings.TrimSpace(in.Reason)
	var out domain.Snapshot
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		requirement, err := lockRequirement(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		transition, err := uc.d.Workflow.ResolveTransition(ctx, requirement.Snapshot().State.Key, in.To, in.Actor.Roles)
		if err != nil {
			return err
		}
		if transition.RequiresReason && reason == "" {
			return axis.Invalid(axis.Required("reason", "Escribe el motivo."))
		}
		facts, err := gatherFacts(ctx, uc.d, requirement)
		if err != nil {
			return err
		}
		if blocking := domain.Blocking(domain.Unmet(transition.Rules, facts)); len(blocking) > 0 {
			return blockedError(transition.To, blocking)
		}
		moved := requirement.MoveTo(transition.To, uc.d.Clock.Now())
		if err := uc.d.Repository.Update(ctx, requirement); err != nil {
			return err
		}
		out = requirement.Snapshot()
		if transition.Permission == PermissionApproveWithDefects {
			// Sus DEF abiertos quedan aceptados, con el mismo motivo: siguen como traza sin bloquear.
			if err := uc.d.Facts.AcceptOpenDefects(ctx, out.Code, in.Actor, reason); err != nil {
				return err
			}
		}
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, RequirementId: out.Id, Code: out.Code, Action: port.AuditTransition,
			Field: moved.Field, OldValue: moved.Old, NewValue: moved.New, Reason: reason,
		})
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}

// blockedError dice qué falta para avanzar, con un mensaje que el usuario puede seguir.
func blockedError(to domain.State, blocking []domain.Condition) error {
	messages := make([]string, len(blocking))
	for i, c := range blocking {
		messages[i] = c.Message
	}
	return axis.New(axis.InvalidAction, fmt.Sprintf("No se puede pasar a «%s». %s", to.Name, strings.Join(messages, " ")))
}

package usecase

import (
	"context"
	"fmt"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// TransitionPackage cambia el estado de un PKG si el flujo, el rol y las reglas de bloqueo lo permiten.
type TransitionPackage struct {
	d Deps
}

// TransitionPackageInput es la entrada de TransitionPackage.
type TransitionPackageInput struct {
	Actor  shared.Actor
	Code   string
	To     string
	Reason string
}

// Execute devuelve INVALID_ACTION si el flujo no permite el paso o si una regla de bloqueo
// no se cumple (con lo que falta en el mensaje), PERMISSION_DENIED si el rol no tiene el
// permiso de la transición e INVALID_REQUEST si falta el motivo que exige.
// Volver a armado quita la confirmación de la regresión: el contenido puede cambiar.
func (uc *TransitionPackage) Execute(ctx context.Context, in TransitionPackageInput) (PackageOutput, error) {
	if in.To == "" {
		return PackageOutput{}, axis.Invalid(axis.Required("to", "Indica el estado destino."))
	}
	reason := strings.TrimSpace(in.Reason)
	var out PackageOutput
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		pkg, err := lockPackage(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		transition, err := uc.d.Workflow.ResolveTransition(ctx, pkg.Snapshot().State.Key, in.To, in.Actor.Roles)
		if err != nil {
			return err
		}
		if transition.RequiresReason && reason == "" {
			return axis.Invalid(axis.Required("reason", "Escribe el motivo."))
		}
		facts, err := gatherFacts(ctx, uc.d, pkg)
		if err != nil {
			return err
		}
		if blocking := domain.Blocking(domain.Unmet(transition.Rules, facts)); len(blocking) > 0 {
			return blockedError(transition.To, blocking)
		}
		moved := pkg.MoveTo(transition.To, uc.d.Clock.Now())
		if err := uc.d.Repository.Update(ctx, pkg); err != nil {
			return err
		}
		s := pkg.Snapshot()
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, PackageId: s.Id, Code: s.Code, Action: port.AuditTransition,
			Field: moved.Field, OldValue: moved.Old, NewValue: moved.New, Reason: reason,
		}); err != nil {
			return err
		}
		if transition.To.IsInitial {
			if err := uc.d.Regression.Reopen(ctx, in.Actor.Id, s.Code, reasonBackToAssembly); err != nil {
				return err
			}
		}
		out, err = readPackage(ctx, uc.d, in.Code)
		return err
	})
	if err != nil {
		return PackageOutput{}, err
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

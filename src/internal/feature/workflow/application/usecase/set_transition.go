package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// SetTransition agrega una transición entre dos estados, o cambia el permiso que exige
// y si pide motivo. Las reglas de bloqueo de una transición existente no se tocan.
type SetTransition struct {
	d Deps
}

// SetTransitionInput es la entrada de SetTransition.
type SetTransitionInput struct {
	Actor          shared.Actor
	EntityType     string
	From           string
	To             string
	Permission     string
	RequiresReason bool
}

// Execute exige el permiso admin.configure.
func (uc *SetTransition) Execute(ctx context.Context, in SetTransitionInput) (domain.Workflow, error) {
	return changeWorkflow(ctx, uc.d, in.Actor, in.EntityType, func(ctx context.Context, wf domain.Workflow) (port.AuditRecord, error) {
		known, err := uc.d.Repository.Permissions(ctx)
		if err != nil {
			return port.AuditRecord{}, err
		}
		transition, err := wf.NewTransition(in.From, in.To, in.Permission, in.RequiresReason, known)
		if err != nil {
			return port.AuditRecord{}, err
		}
		before, after := "", describeTransition(transition)
		if current, exists := wf.FindTransition(in.From, in.To); exists {
			before = describeTransition(current)
		}
		if before == after {
			return port.AuditRecord{}, nil
		}
		if err := uc.d.Repository.SaveTransition(ctx, wf.EntityType, transition, uc.d.Clock.Now()); err != nil {
			return port.AuditRecord{}, err
		}
		return port.AuditRecord{
			Entity: auditTransition, Key: transitionAuditKey(wf, in.From, in.To), Field: "transition",
			OldValue: before, NewValue: after,
		}, nil
	})
}

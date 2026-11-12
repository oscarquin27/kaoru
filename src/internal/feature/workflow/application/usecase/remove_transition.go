package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// RemoveTransition quita una transición y, con ella, sus reglas de bloqueo.
type RemoveTransition struct {
	d Deps
}

// RemoveTransitionInput es la entrada de RemoveTransition.
type RemoveTransitionInput struct {
	Actor      shared.Actor
	EntityType string
	From       string
	To         string
}

// Execute exige el permiso admin.configure.
func (uc *RemoveTransition) Execute(ctx context.Context, in RemoveTransitionInput) (domain.Workflow, error) {
	return changeWorkflow(ctx, uc.d, in.Actor, in.EntityType, func(ctx context.Context, wf domain.Workflow) (port.AuditRecord, error) {
		current, exists := wf.FindTransition(in.From, in.To)
		if !exists {
			return port.AuditRecord{}, domain.ErrTransitionMissing
		}
		if err := uc.d.Repository.DeleteTransition(ctx, wf.EntityType, in.From, in.To); err != nil {
			return port.AuditRecord{}, err
		}
		return port.AuditRecord{
			Entity: auditTransition, Key: transitionAuditKey(wf, in.From, in.To), Field: "transition",
			OldValue: describeTransition(current),
		}, nil
	})
}

package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// RemoveState quita un estado que nadie usa: sin transiciones y sin registros en él.
type RemoveState struct {
	d Deps
}

// RemoveStateInput es la entrada de RemoveState.
type RemoveStateInput struct {
	Actor      shared.Actor
	EntityType string
	Key        string
}

// Execute exige el permiso admin.configure.
func (uc *RemoveState) Execute(ctx context.Context, in RemoveStateInput) (domain.Workflow, error) {
	return changeWorkflow(ctx, uc.d, in.Actor, in.EntityType, func(ctx context.Context, wf domain.Workflow) (port.AuditRecord, error) {
		state, err := wf.RemovableState(in.Key)
		if err != nil {
			return port.AuditRecord{}, err
		}
		if err := uc.d.Repository.DeleteState(ctx, wf.EntityType, state.Key); err != nil {
			return port.AuditRecord{}, err
		}
		return port.AuditRecord{
			Entity: auditState, Key: stateAuditKey(wf, state.Key), Field: "state", OldValue: describeState(state),
		}, nil
	})
}

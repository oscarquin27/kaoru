package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateState cambia el nombre de un estado o su lugar en la línea de vida.
type UpdateState struct {
	d Deps
}

// UpdateStateInput es la entrada de UpdateState.
type UpdateStateInput struct {
	Actor      shared.Actor
	EntityType string
	Key        string
	Edit       domain.StateEdit
}

// Execute exige el permiso admin.configure.
func (uc *UpdateState) Execute(ctx context.Context, in UpdateStateInput) (domain.Workflow, error) {
	return changeWorkflow(ctx, uc.d, in.Actor, in.EntityType, func(ctx context.Context, wf domain.Workflow) (port.AuditRecord, error) {
		state, err := wf.EditState(in.Key, in.Edit)
		if err != nil {
			return port.AuditRecord{}, err
		}
		current, _ := wf.State(in.Key)
		before, after := describeState(current), describeState(state)
		if before == after {
			return port.AuditRecord{}, nil
		}
		if err := uc.d.Repository.UpdateState(ctx, wf.EntityType, state, uc.d.Clock.Now()); err != nil {
			return port.AuditRecord{}, err
		}
		return port.AuditRecord{
			Entity: auditState, Key: stateAuditKey(wf, state.Key), Field: "state", OldValue: before, NewValue: after,
		}, nil
	})
}

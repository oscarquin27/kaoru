package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// AddState agrega un estado a un flujo. Queda sin transiciones hasta que se le agreguen.
type AddState struct {
	d Deps
}

// AddStateInput es la entrada de AddState; Position en nil lo deja como estado lateral.
type AddStateInput struct {
	Actor      shared.Actor
	EntityType string
	Key        string
	Name       string
	Position   *int
	IsFinal    bool
}

// Execute exige el permiso admin.configure.
func (uc *AddState) Execute(ctx context.Context, in AddStateInput) (domain.Workflow, error) {
	return changeWorkflow(ctx, uc.d, in.Actor, in.EntityType, func(ctx context.Context, wf domain.Workflow) (port.AuditRecord, error) {
		state, err := wf.NewState(in.Key, in.Name, in.Position, in.IsFinal)
		if err != nil {
			return port.AuditRecord{}, err
		}
		if err := uc.d.Repository.InsertState(ctx, wf.EntityType, state, uc.d.Clock.Now()); err != nil {
			return port.AuditRecord{}, err
		}
		return port.AuditRecord{
			Entity: auditState, Key: stateAuditKey(wf, state.Key), Field: "state", NewValue: describeState(state),
		}, nil
	})
}

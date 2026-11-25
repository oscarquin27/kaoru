package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
)

// ResolveTransition responde si una transición vale para los roles de un actor
// y qué exige. El feature dueño de la entidad aplica la respuesta.
type ResolveTransition struct {
	repo port.Repository
}

// ResolveTransitionInput es la entrada de ResolveTransition.
type ResolveTransitionInput struct {
	EntityType string
	From       string
	To         string
	Roles      []string
}

// ResolveTransitionOutput es la transición resuelta.
type ResolveTransitionOutput struct {
	To             domain.State
	RequiresReason bool
	// Permission es el permiso de la transición: algunos pasos hacen algo más según cuál sea.
	Permission string
	// Rules son las reglas activas de la transición.
	Rules []domain.Rule
}

// Execute devuelve INVALID_ACTION si el flujo no permite el paso
// y PERMISSION_DENIED si ningún rol tiene el permiso de la transición.
func (uc *ResolveTransition) Execute(ctx context.Context, in ResolveTransitionInput) (ResolveTransitionOutput, error) {
	wf, err := loadWorkflow(ctx, uc.repo, in.EntityType)
	if err != nil {
		return ResolveTransitionOutput{}, err
	}
	transition, err := wf.Transition(in.From, in.To)
	if err != nil {
		return ResolveTransitionOutput{}, err
	}
	if err := authorize(ctx, uc.repo, in.Roles, transition.Permission); err != nil {
		return ResolveTransitionOutput{}, err
	}
	to, err := wf.State(in.To)
	if err != nil {
		return ResolveTransitionOutput{}, err
	}
	out := ResolveTransitionOutput{To: to, RequiresReason: transition.RequiresReason, Permission: transition.Permission}
	for _, r := range transition.Rules {
		if r.Enabled {
			out.Rules = append(out.Rules, r)
		}
	}
	return out, nil
}

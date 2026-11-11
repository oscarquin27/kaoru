package usecase

import (
	"context"

	"kaoru/internal/feature/requirement/domain"
)

// GetReadiness calcula «qué falta para avanzar»: las condiciones pendientes de cada
// transición que sale del estado actual del REQ.
type GetReadiness struct {
	d Deps
}

// GetReadinessInput es la entrada de GetReadiness.
type GetReadinessInput struct {
	Code string
}

// TransitionReadiness es una transición posible con lo que le falta.
type TransitionReadiness struct {
	To             domain.State
	RequiresReason bool
	// Conditions incluye las que bloquean y las que solo avisan.
	Conditions []domain.Condition
}

// Execute evalúa las reglas sin cambiar nada. Leer no exige permiso.
func (uc *GetReadiness) Execute(ctx context.Context, in GetReadinessInput) ([]TransitionReadiness, error) {
	requirement, err := findRequirement(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return nil, err
	}
	transitions, err := uc.d.Workflow.Outgoing(ctx, requirement.Snapshot().State.Key)
	if err != nil {
		return nil, err
	}
	facts, err := gatherFacts(ctx, uc.d, requirement)
	if err != nil {
		return nil, err
	}
	out := make([]TransitionReadiness, len(transitions))
	for i, t := range transitions {
		out[i] = TransitionReadiness{To: t.To, RequiresReason: t.RequiresReason, Conditions: domain.Unmet(t.Rules, facts)}
	}
	return out, nil
}

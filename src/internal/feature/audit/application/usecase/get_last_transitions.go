package usecase

import (
	"context"
	"time"
)

// GetLastTransitions dice cuándo entró cada entidad a su estado actual. Lo usan los
// features con ciclo de vida para calcular el tiempo en estado al leer; no se expone por HTTP.
type GetLastTransitions struct {
	d Deps
}

// GetLastTransitionsInput es la entrada de GetLastTransitions.
type GetLastTransitionsInput struct {
	EntityIds []string
}

// Execute devuelve la fecha de la última transición por id de entidad.
// Una entidad que nunca cambió de estado no aparece.
func (uc *GetLastTransitions) Execute(ctx context.Context, in GetLastTransitionsInput) (map[string]time.Time, error) {
	if len(in.EntityIds) == 0 {
		return map[string]time.Time{}, nil
	}
	return uc.d.Repository.LastTransitions(ctx, in.EntityIds)
}

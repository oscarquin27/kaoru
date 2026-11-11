// Package requirement implementa el puerto Readiness con los casos de uso de requirement.
package requirement

import (
	"context"

	requirementuc "kaoru/internal/feature/requirement/application/usecase"
	"kaoru/internal/feature/traceability/application/port"
	"kaoru/internal/feature/traceability/domain"
)

// Readiness pregunta a requirement qué falta para que un REQ avance.
type Readiness struct {
	get *requirementuc.GetReadiness
}

var _ port.Readiness = (*Readiness)(nil)

// New crea el adaptador.
func New(get *requirementuc.GetReadiness) *Readiness { return &Readiness{get: get} }

// Of conserva el código de error de requirement.
func (r *Readiness) Of(ctx context.Context, requirementCode string) ([]domain.Readiness, error) {
	transitions, err := r.get.Execute(ctx, requirementuc.GetReadinessInput{Code: requirementCode})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Readiness, len(transitions))
	for i, t := range transitions {
		out[i] = domain.Readiness{To: domain.StateRef{Key: t.To.Key, Name: t.To.Name}, Conditions: make([]domain.Condition, len(t.Conditions))}
		for j, c := range t.Conditions {
			out[i].Conditions[j] = domain.Condition{RuleKey: c.RuleKey, Message: c.Message, Blocks: c.Blocks, Related: c.Related}
		}
	}
	return out, nil
}

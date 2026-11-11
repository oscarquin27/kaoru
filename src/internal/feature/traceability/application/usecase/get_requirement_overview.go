package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/traceability/domain"
)

// GetRequirementOverview arma la ficha consolidada de un REQ: encabezado, línea de vida con
// el tiempo en cada etapa, «qué falta para avanzar», cobertura y lo relacionado.
type GetRequirementOverview struct {
	d Deps
}

// GetRequirementOverviewInput es la entrada de GetRequirementOverview.
type GetRequirementOverviewInput struct {
	Code string
}

// Execute devuelve NOT_FOUND_RESOURCE si el REQ no existe. Leer no exige permiso.
func (uc *GetRequirementOverview) Execute(ctx context.Context, in GetRequirementOverviewInput) (domain.Overview, error) {
	repo := uc.d.Repository
	header, err := repo.RequirementHeader(ctx, in.Code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Overview{}, ErrRequirementNotFound
	}
	if err != nil {
		return domain.Overview{}, err
	}
	out := domain.Overview{Requirement: header}
	if out.Missing, err = uc.d.Readiness.Of(ctx, header.Code); err != nil {
		return domain.Overview{}, err
	}
	if out.Coverage, err = uc.d.Coverage.Of(ctx, header.Code); err != nil {
		return domain.Overview{}, err
	}
	if out.FunctionalCases, err = repo.FunctionalCases(ctx, header.Id); err != nil {
		return domain.Overview{}, err
	}
	if out.TestCases, err = repo.TestCases(ctx, header.Id); err != nil {
		return domain.Overview{}, err
	}
	if out.Defects, err = repo.Defects(ctx, header.Id); err != nil {
		return domain.Overview{}, err
	}
	states, err := repo.LifecycleStates(ctx)
	if err != nil {
		return domain.Overview{}, err
	}
	transitions, err := repo.Transitions(ctx, header.Id)
	if err != nil {
		return domain.Overview{}, err
	}
	// La entidad nació en el estado del que salió su primera transición; si nunca cambió, en el actual.
	initial := header.State.Key
	if len(transitions) > 0 {
		initial = transitions[0].From
	}
	out.Lifecycle = domain.Lifecycle(states, initial, header.CreatedAt, transitions, header.State.Key,
		nextIsBlocked(states, header.State.Key, out.Missing), uc.d.Clock.Now())
	return out, nil
}

// nextIsBlocked dice si la etapa que sigue a la actual tiene alguna condición que bloquea.
func nextIsBlocked(states []domain.StateRef, current string, missing []domain.Readiness) bool {
	for i, s := range states {
		if s.Key != current || i+1 >= len(states) {
			continue
		}
		for _, readiness := range missing {
			if readiness.To.Key != states[i+1].Key {
				continue
			}
			for _, c := range readiness.Conditions {
				if c.Blocks {
					return true
				}
			}
		}
	}
	return false
}

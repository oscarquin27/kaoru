package usecase

import (
	"context"

	"kaoru/internal/feature/execution/domain"
)

// ListExplorations devuelve una página de EXP, del más reciente al más antiguo. Leer no exige permiso.
type ListExplorations struct {
	d Deps
}

// ListExplorationsInput es la entrada de ListExplorations. RequirementCode vacío no filtra.
type ListExplorationsInput struct {
	RequirementCode string
	Page            int
	Size            int
}

// ListExplorationsOutput es una página de EXP.
type ListExplorationsOutput struct {
	Explorations []domain.ExplorationSnapshot
	Total        int
}

// Execute lista los EXP.
func (uc *ListExplorations) Execute(ctx context.Context, in ListExplorationsInput) (ListExplorationsOutput, error) {
	explorations, total, err := uc.d.Repository.ListExplorations(ctx, in.RequirementCode, in.Page, in.Size)
	if err != nil {
		return ListExplorationsOutput{}, err
	}
	out := ListExplorationsOutput{Explorations: make([]domain.ExplorationSnapshot, len(explorations)), Total: total}
	for i, e := range explorations {
		out.Explorations[i] = e.Snapshot()
	}
	return out, nil
}

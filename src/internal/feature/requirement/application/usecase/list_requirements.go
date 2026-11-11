package usecase

import (
	"context"

	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
)

// ListRequirements devuelve una página de REQ, del más reciente al más antiguo.
type ListRequirements struct {
	d Deps
}

// ListRequirementsInput es la entrada de ListRequirements. Un filtro vacío no filtra.
type ListRequirementsInput struct {
	SystemKey  string
	StateKey   string
	ChangeCode string
	OwnerId    string
	Query      string
	Page       int
	Size       int
}

// ListRequirementsOutput es una página de REQ.
type ListRequirementsOutput struct {
	Requirements []domain.Snapshot
	Total        int
}

// Execute lista los REQ. Leer no exige permiso.
func (uc *ListRequirements) Execute(ctx context.Context, in ListRequirementsInput) (ListRequirementsOutput, error) {
	requirements, total, err := uc.d.Repository.List(ctx, port.Filter{
		SystemKey: in.SystemKey, StateKey: in.StateKey, ChangeCode: in.ChangeCode, OwnerId: in.OwnerId,
		Query: in.Query, Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListRequirementsOutput{}, err
	}
	out := ListRequirementsOutput{Requirements: make([]domain.Snapshot, len(requirements)), Total: total}
	for i, r := range requirements {
		out.Requirements[i] = r.Snapshot()
	}
	return out, nil
}

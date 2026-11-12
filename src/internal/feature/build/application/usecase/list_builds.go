package usecase

import (
	"context"

	"kaoru/internal/feature/build/application/port"
	"kaoru/internal/feature/build/domain"
)

// ListBuilds devuelve una página de BUILD, del más reciente al más antiguo.
type ListBuilds struct {
	d Deps
}

// ListBuildsInput es la entrada de ListBuilds. Un filtro vacío no filtra.
type ListBuildsInput struct {
	SystemKey       string
	RequirementCode string
	Query           string
	Page            int
	Size            int
}

// ListBuildsOutput es una página de BUILD.
type ListBuildsOutput struct {
	Builds []domain.Snapshot
	Total  int
}

// Execute lista los BUILD. Leer no exige permiso.
func (uc *ListBuilds) Execute(ctx context.Context, in ListBuildsInput) (ListBuildsOutput, error) {
	builds, total, err := uc.d.Repository.List(ctx, port.Filter{
		SystemKey: in.SystemKey, RequirementCode: in.RequirementCode, Query: in.Query, Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListBuildsOutput{}, err
	}
	out := ListBuildsOutput{Builds: make([]domain.Snapshot, len(builds)), Total: total}
	for i, b := range builds {
		out.Builds[i] = b.Snapshot()
	}
	return out, nil
}

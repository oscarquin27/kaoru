package usecase

import (
	"context"

	"kaoru/internal/feature/environment/application/port"
)

// ListEnvironments devuelve una página de ambientes con lo que cada uno tiene instalado.
type ListEnvironments struct {
	d Deps
}

// ListEnvironmentsInput es la entrada de ListEnvironments. Un filtro vacío no filtra;
// los retirados solo salen con IncludeRetired.
type ListEnvironmentsInput struct {
	SystemKey      string
	Kind           string
	IncludeRetired bool
	Page           int
	Size           int
}

// ListEnvironmentsOutput es una página de ENV.
type ListEnvironmentsOutput struct {
	Environments []EnvironmentOutput
	Total        int
}

// Execute lista los ENV. Leer no exige permiso.
func (uc *ListEnvironments) Execute(ctx context.Context, in ListEnvironmentsInput) (ListEnvironmentsOutput, error) {
	views, total, err := uc.d.Repository.ListEnvironments(ctx, port.EnvironmentFilter{
		SystemKey: in.SystemKey, Kind: in.Kind, IncludeRetired: in.IncludeRetired, Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListEnvironmentsOutput{}, err
	}
	out := ListEnvironmentsOutput{Environments: make([]EnvironmentOutput, len(views)), Total: total}
	for i, v := range views {
		out.Environments[i] = toOutput(v)
	}
	return out, nil
}

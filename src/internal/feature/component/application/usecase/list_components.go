package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/component/application/port"
	"kaoru/internal/feature/component/domain"
)

// ListComponents devuelve una página del catálogo, por nombre.
type ListComponents struct {
	d Deps
}

// ListComponentsInput es la entrada de ListComponents. Un filtro vacío no filtra;
// sin IncludeRetired, los retirados no salen.
type ListComponentsInput struct {
	SystemKey      string
	Kind           string
	Query          string
	IncludeRetired bool
	Page           int
	Size           int
}

// ListComponentsOutput es una página de COMP.
type ListComponentsOutput struct {
	Components []ComponentOutput
	Total      int
}

// Execute lista los COMP. Leer no exige permiso.
func (uc *ListComponents) Execute(ctx context.Context, in ListComponentsInput) (ListComponentsOutput, error) {
	if _, ok := domain.ParseKind(in.Kind); in.Kind != "" && !ok {
		return ListComponentsOutput{}, axis.Invalid(axis.InvalidField("kind", "La clase de componente no es válida."))
	}
	views, total, err := uc.d.Repository.List(ctx, port.Filter{
		SystemKey: in.SystemKey, Kind: in.Kind, Query: in.Query, IncludeRetired: in.IncludeRetired,
		Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListComponentsOutput{}, err
	}
	out := ListComponentsOutput{Components: make([]ComponentOutput, len(views)), Total: total}
	for i, v := range views {
		out.Components[i] = toOutput(v)
	}
	return out, nil
}

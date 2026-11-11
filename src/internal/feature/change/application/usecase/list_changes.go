package usecase

import (
	"context"

	"kaoru/internal/feature/change/application/port"
)

// ListChanges devuelve una página de CHG, del más reciente al más antiguo.
type ListChanges struct {
	d Deps
}

// ListChangesInput es la entrada de ListChanges. Un filtro vacío no filtra.
type ListChangesInput struct {
	SystemKey string
	StateKey  string
	Query     string
	Page      int
	Size      int
}

// ListChangesOutput es una página de CHG.
type ListChangesOutput struct {
	Changes []ChangeOutput
	Total   int
}

// Execute lista los CHG. Leer no exige permiso.
func (uc *ListChanges) Execute(ctx context.Context, in ListChangesInput) (ListChangesOutput, error) {
	views, total, err := uc.d.Repository.List(ctx, port.Filter{
		SystemKey: in.SystemKey, StateKey: in.StateKey, Query: in.Query, Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListChangesOutput{}, err
	}
	out := ListChangesOutput{Changes: make([]ChangeOutput, len(views)), Total: total}
	for i, v := range views {
		out.Changes[i] = toOutput(v)
	}
	return out, nil
}

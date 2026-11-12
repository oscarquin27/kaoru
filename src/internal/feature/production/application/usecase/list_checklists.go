package usecase

import (
	"context"

	"kaoru/internal/feature/production/application/port"
)

// ListChecklists devuelve una página de checklists, del más reciente al más antiguo.
type ListChecklists struct {
	d Deps
}

// ListInput es la entrada de las listas del feature. PackageCode vacío no filtra.
type ListInput struct {
	PackageCode string
	Page        int
	Size        int
}

func (in ListInput) filter() port.Filter {
	return port.Filter{PackageCode: in.PackageCode, Page: in.Page, Size: in.Size}
}

// ListChecklistsOutput es una página de CHK.
type ListChecklistsOutput struct {
	Checklists []ChecklistOutput
	Total      int
}

// Execute lista los CHK. Leer no exige permiso.
func (uc *ListChecklists) Execute(ctx context.Context, in ListInput) (ListChecklistsOutput, error) {
	checklists, total, err := uc.d.Repository.ListChecklists(ctx, in.filter())
	if err != nil {
		return ListChecklistsOutput{}, err
	}
	out := ListChecklistsOutput{Checklists: make([]ChecklistOutput, len(checklists)), Total: total}
	for i, c := range checklists {
		out.Checklists[i] = toChecklistOutput(c)
	}
	return out, nil
}

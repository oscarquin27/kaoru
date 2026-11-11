package usecase

import (
	"context"

	"kaoru/internal/feature/audit/application/port"
	shared "kaoru/internal/shared/domain"
)

// ListEntries consulta la auditoría, del registro más reciente al más antiguo.
type ListEntries struct {
	d Deps
}

// ListEntriesInput es la entrada de ListEntries.
type ListEntriesInput struct {
	Actor  shared.Actor
	Filter port.Filter
}

// ListEntriesOutput es una página de registros.
type ListEntriesOutput struct {
	Entries []port.EntryView
	Total   int
}

// Execute exige el permiso audit.read.
func (uc *ListEntries) Execute(ctx context.Context, in ListEntriesInput) (ListEntriesOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionRead); err != nil {
		return ListEntriesOutput{}, err
	}
	entries, total, err := uc.d.Repository.List(ctx, in.Filter)
	if err != nil {
		return ListEntriesOutput{}, err
	}
	return ListEntriesOutput{Entries: entries, Total: total}, nil
}

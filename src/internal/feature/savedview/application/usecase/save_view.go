package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/savedview/domain"
	shared "kaoru/internal/shared/domain"
)

// SaveView guarda los filtros de una lista con un nombre. Cualquier usuario con sesión
// guarda las suyas: no exige permiso.
type SaveView struct {
	d Deps
}

// SaveViewInput es la entrada de SaveView.
type SaveViewInput struct {
	Owner  shared.UserId
	Screen string
	Name   string
	Query  string
}

// Execute valida y guarda la vista.
func (uc *SaveView) Execute(ctx context.Context, in SaveViewInput) (domain.View, error) {
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.View{}, err
	}
	view, err := domain.New(id, in.Owner, in.Screen, in.Name, in.Query, uc.d.Clock.Now())
	if err != nil {
		return domain.View{}, err
	}
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		count, err := uc.d.Repository.Count(ctx, view.OwnerId, view.Screen)
		if err != nil {
			return err
		}
		if count >= domain.MaxPerScreen {
			return domain.ErrTooMany
		}
		err = uc.d.Repository.Insert(ctx, view)
		if axis.IsCode(err, axis.DuplicateResource) {
			return axis.Invalid(axis.InvalidField("name", "Ya tienes una vista con ese nombre en esta lista."))
		}
		return err
	})
	if err != nil {
		return domain.View{}, err
	}
	return view, nil
}

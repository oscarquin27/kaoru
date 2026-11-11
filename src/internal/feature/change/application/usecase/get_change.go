package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/change/application/port"
	shared "kaoru/internal/shared/domain"
)

// ErrChangeNotFound es el error de un CHG que no existe.
var ErrChangeNotFound = axis.New(axis.NotFoundResource, "El CHG no existe.")

// GetChange devuelve un CHG por su ID legible. Leer no exige permiso.
type GetChange struct {
	d Deps
}

// GetChangeInput es la entrada de GetChange.
type GetChangeInput struct {
	Code string
}

// Execute busca el CHG.
func (uc *GetChange) Execute(ctx context.Context, in GetChangeInput) (ChangeOutput, error) {
	view, err := findChange(ctx, uc.d.Repository, in.Code)
	return toOutput(view), err
}

// lockChange bloquea la entidad y la lee: se usa dentro de la transacción de todo caso de uso que
// escribe, para que dos cambios simultáneos sobre la misma entidad no se pisen.
func lockChange(ctx context.Context, repo port.Repository, raw string) (port.ChangeView, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return port.ChangeView{}, ErrChangeNotFound
	}
	if err := repo.Lock(ctx, code); err != nil {
		if axis.IsCode(err, axis.NotFoundResource) {
			return port.ChangeView{}, ErrChangeNotFound
		}
		return port.ChangeView{}, err
	}
	return findChange(ctx, repo, raw)
}

// findChange valida el código y lee el CHG; un código mal formado tampoco existe.
func findChange(ctx context.Context, repo port.Repository, raw string) (port.ChangeView, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return port.ChangeView{}, ErrChangeNotFound
	}
	view, err := repo.FindByCode(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return port.ChangeView{}, ErrChangeNotFound
	}
	return view, err
}

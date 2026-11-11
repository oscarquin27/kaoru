package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// GetRun devuelve un RUN por su ID legible. Leer no exige permiso.
type GetRun struct {
	d Deps
}

// GetRunInput es la entrada de GetRun.
type GetRunInput struct {
	Code string
}

// Execute busca el RUN; un código mal formado tampoco existe.
func (uc *GetRun) Execute(ctx context.Context, in GetRunInput) (domain.RunSnapshot, error) {
	code, err := shared.ParseCode(in.Code)
	if err != nil {
		return domain.RunSnapshot{}, ErrRunNotFound
	}
	run, err := uc.d.Repository.FindRun(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.RunSnapshot{}, ErrRunNotFound
	}
	return run.Snapshot(), err
}

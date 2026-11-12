package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// GetSmokeTest devuelve un SMOKE por su ID legible. Leer no exige permiso.
type GetSmokeTest struct {
	d Deps
}

// GetSmokeTestInput es la entrada de GetSmokeTest.
type GetSmokeTestInput struct {
	Code string
}

// Execute busca el SMOKE; un código mal formado tampoco existe.
func (uc *GetSmokeTest) Execute(ctx context.Context, in GetSmokeTestInput) (domain.SmokeTestSnapshot, error) {
	code, err := shared.ParseCode(in.Code)
	if err != nil {
		return domain.SmokeTestSnapshot{}, ErrSmokeTestNotFound
	}
	smokeTest, err := uc.d.Repository.FindSmokeTest(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.SmokeTestSnapshot{}, ErrSmokeTestNotFound
	}
	return smokeTest.Snapshot(), err
}

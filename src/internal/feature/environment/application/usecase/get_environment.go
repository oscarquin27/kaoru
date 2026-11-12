package usecase

import (
	"context"
)

// GetEnvironment devuelve la ficha de un ambiente con lo que tiene instalado. Leer no exige permiso.
type GetEnvironment struct {
	d Deps
}

// GetEnvironmentInput es la entrada de GetEnvironment.
type GetEnvironmentInput struct {
	Code string
}

// Execute busca el ENV.
func (uc *GetEnvironment) Execute(ctx context.Context, in GetEnvironmentInput) (EnvironmentOutput, error) {
	view, err := findEnvironment(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return EnvironmentOutput{}, err
	}
	return toOutput(view), nil
}

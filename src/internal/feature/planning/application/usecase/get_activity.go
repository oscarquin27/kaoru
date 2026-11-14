package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
)

// GetActivity devuelve la ficha de una actividad con sus solapes.
type GetActivity struct {
	d Deps
}

// Execute lee la actividad. Leer no exige permiso.
func (uc *GetActivity) Execute(ctx context.Context, code string) (ActivityOutput, error) {
	activity, err := uc.d.Repository.FindActivity(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return ActivityOutput{}, ErrActivityNotFound
	}
	if err != nil {
		return ActivityOutput{}, err
	}
	return withOverlaps(ctx, uc.d.Repository, activity, uc.d.Clock.Now())
}

package usecase

import (
	"context"
	"slices"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/savedview/domain"
	shared "kaoru/internal/shared/domain"
)

// ListViews devuelve las vistas que una persona guardó en una lista.
type ListViews struct {
	d Deps
}

// Execute exige una lista conocida.
func (uc *ListViews) Execute(ctx context.Context, owner shared.UserId, screen string) ([]domain.View, error) {
	if !slices.Contains(domain.Screens, screen) {
		return nil, axis.Invalid(axis.InvalidField("screen", "Esa lista no admite vistas guardadas."))
	}
	return uc.d.Repository.List(ctx, owner, screen)
}

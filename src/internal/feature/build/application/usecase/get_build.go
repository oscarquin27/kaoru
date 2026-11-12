package usecase

import (
	"context"

	"kaoru/internal/feature/build/domain"
)

// GetBuild devuelve un BUILD con su manifiesto. Leer no exige permiso.
type GetBuild struct {
	d Deps
}

// GetBuildInput es la entrada de GetBuild.
type GetBuildInput struct {
	Code string
}

// Execute busca el BUILD.
func (uc *GetBuild) Execute(ctx context.Context, in GetBuildInput) (domain.Snapshot, error) {
	build, err := findBuild(ctx, uc.d.Repository, in.Code)
	return build.Snapshot(), err
}

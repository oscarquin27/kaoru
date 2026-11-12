// Package build implementa el puerto Builds con los casos de uso de build.
package build

import (
	"context"

	builduc "kaoru/internal/feature/build/application/usecase"
	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
)

// Builds busca BUILD en el feature build.
type Builds struct {
	find *builduc.FindBuild
}

var _ port.Builds = (*Builds)(nil)

// New crea el adaptador.
func New(find *builduc.FindBuild) *Builds { return &Builds{find: find} }

// Find conserva el código de error de build.
func (b *Builds) Find(ctx context.Context, code string) (domain.Build, error) {
	ref, err := b.find.Execute(ctx, builduc.FindBuildInput{Code: code})
	if err != nil {
		return domain.Build{}, err
	}
	return domain.Build{
		Ref:       domain.BuildRef{Id: ref.Id, Code: ref.Code, Version: ref.Version, Checksum: ref.Checksum},
		SystemKey: ref.SystemKey,
	}, nil
}

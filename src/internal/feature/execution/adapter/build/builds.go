// Package build implementa el puerto Builds con los casos de uso de build.
package build

import (
	"context"

	builduc "kaoru/internal/feature/build/application/usecase"
	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/domain"
)

// Builds busca BUILD en el feature build.
type Builds struct {
	find *builduc.FindBuild
}

var _ port.Builds = (*Builds)(nil)

// New crea el adaptador.
func New(find *builduc.FindBuild) *Builds { return &Builds{find: find} }

// Find conserva el código de error de build.
func (b *Builds) Find(ctx context.Context, code string) (domain.BuildRef, error) {
	ref, err := b.find.Execute(ctx, builduc.FindBuildInput{Code: code})
	if err != nil {
		return domain.BuildRef{}, err
	}
	return domain.BuildRef{Id: ref.Id, Code: ref.Code, SystemKey: ref.SystemKey, Version: ref.Version, Checksum: ref.Checksum}, nil
}

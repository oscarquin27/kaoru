package usecase

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
)

// ListPackages devuelve una página de PKG, del más reciente al más antiguo.
type ListPackages struct {
	d Deps
}

// ListPackagesInput es la entrada de ListPackages. Un filtro vacío no filtra.
type ListPackagesInput struct {
	SystemKey string
	StateKey  string
	Query     string
	Page      int
	Size      int
}

// ListPackagesOutput es una página de PKG.
type ListPackagesOutput struct {
	Packages []PackageOutput
	Total    int
}

// Execute lista los PKG con su contenido; «qué falta» no se calcula aquí. Leer no exige permiso.
func (uc *ListPackages) Execute(ctx context.Context, in ListPackagesInput) (ListPackagesOutput, error) {
	packages, total, err := uc.d.Repository.List(ctx, port.Filter{
		SystemKey: in.SystemKey, StateKey: in.StateKey, Query: in.Query, Page: in.Page, Size: in.Size,
	})
	if err != nil {
		return ListPackagesOutput{}, err
	}
	out := ListPackagesOutput{Packages: make([]PackageOutput, len(packages)), Total: total}
	for i, pkg := range packages {
		out.Packages[i] = PackageOutput{Package: pkg.Snapshot()}
	}
	return out, nil
}

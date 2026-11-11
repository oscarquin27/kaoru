package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/domain"
)

// ErrLinkNotFound es el error de un enlace que no existe en ese REQ.
var ErrLinkNotFound = axis.New(axis.NotFoundResource, "El enlace no existe.")

// ListLinks devuelve los enlaces de un REQ. Leer no exige permiso.
type ListLinks struct {
	d Deps
}

// ListLinksInput es la entrada de ListLinks.
type ListLinksInput struct {
	Code string
}

// Execute lista los enlaces en el orden en que se agregaron.
func (uc *ListLinks) Execute(ctx context.Context, in ListLinksInput) ([]domain.LinkSnapshot, error) {
	requirement, err := findRequirement(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return nil, err
	}
	links, err := uc.d.Repository.ListLinks(ctx, requirement.Snapshot().Id)
	if err != nil {
		return nil, err
	}
	out := make([]domain.LinkSnapshot, len(links))
	for i, l := range links {
		out[i] = l.Snapshot()
	}
	return out, nil
}

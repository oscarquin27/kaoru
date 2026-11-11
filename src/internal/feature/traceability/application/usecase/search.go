package usecase

import (
	"context"
	"strings"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/traceability/domain"
)

const (
	minSearchLength = 2
	maxSearchHits   = 20
)

// Search es la búsqueda global: por ID legible o por texto del título, en todas las entidades.
type Search struct {
	d Deps
}

// SearchInput es la entrada de Search.
type SearchInput struct {
	Query string
}

// Execute devuelve hasta 20 resultados, con las coincidencias exactas de ID primero.
func (uc *Search) Execute(ctx context.Context, in SearchInput) ([]domain.SearchHit, error) {
	query := strings.TrimSpace(in.Query)
	if len([]rune(query)) < minSearchLength {
		return nil, axis.Invalid(axis.InvalidField("q", "Escribe al menos 2 caracteres."))
	}
	return uc.d.Repository.Search(ctx, query, maxSearchHits)
}

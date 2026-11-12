// Package usecase contiene los casos de uso de savedview.
package usecase

import (
	"errors"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/savedview/application/port"
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	SaveView   *SaveView
	ListViews  *ListViews
	DeleteView *DeleteView
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("savedview: faltan dependencias")
	}
	return &UseCases{SaveView: &SaveView{d: d}, ListViews: &ListViews{d: d}, DeleteView: &DeleteView{d: d}}, nil
}

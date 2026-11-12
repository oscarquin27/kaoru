// Package usecase contiene los casos de uso de traceability. Todos son lecturas.
package usecase

import (
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/feature/traceability/application/port"
)

// ErrRequirementNotFound es el error de un REQ que no existe.
var ErrRequirementNotFound = axis.New(axis.NotFoundResource, "El REQ no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository port.Repository
	Readiness  port.Readiness
	Coverage   port.Coverage
	Clock      clock.Clock
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	ListSystems            *ListSystems
	GetRequirementOverview *GetRequirementOverview
	GetMatrix              *GetMatrix
	ExportMatrix           *ExportMatrix
	Search                 *Search
	GetPackageTrace        *GetPackageTrace
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Readiness == nil || d.Coverage == nil || d.Clock == nil {
		return nil, errors.New("traceability: faltan dependencias")
	}
	return &UseCases{
		ListSystems:            &ListSystems{d: d},
		GetRequirementOverview: &GetRequirementOverview{d: d},
		GetMatrix:              &GetMatrix{d: d},
		ExportMatrix:           &ExportMatrix{d: d},
		Search:                 &Search{d: d},
		GetPackageTrace:        &GetPackageTrace{d: d},
	}, nil
}

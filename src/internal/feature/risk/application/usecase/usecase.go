// Package usecase contiene los casos de uso de risk.
package usecase

import (
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/risk/application/port"
)

// PermissionAssess es el permiso para evaluar el riesgo de un REQ.
const PermissionAssess = "risk.assess"

// ErrRiskNotFound es el error de un REQ cuyo riesgo no está evaluado.
var ErrRiskNotFound = axis.New(axis.NotFoundResource, "El riesgo de este REQ no está evaluado.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository   port.Repository
	Requirements port.Requirements
	Permissions  port.Permissions
	AuditTrail   port.AuditTrail
	UnitOfWork   uow.Factory
	Clock        clock.Clock
	IdGenerator  idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	AssessRisk *AssessRisk
	GetRisk    *GetRisk
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Requirements == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("risk: faltan dependencias")
	}
	return &UseCases{AssessRisk: &AssessRisk{d: d}, GetRisk: &GetRisk{d: d}}, nil
}

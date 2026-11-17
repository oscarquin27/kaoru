// Package usecase contiene los casos de uso de system.
package usecase

import (
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/system/application/port"
)

// PermissionConfigure es el permiso de configurar catálogos (qa-admin por omisión).
const PermissionConfigure = "admin.configure"

// ErrSystemNotFound es el error de una clave que no existe.
var ErrSystemNotFound = axis.New(axis.NotFoundResource, "El sistema no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateSystem *CreateSystem
	UpdateSystem *UpdateSystem
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("system: faltan dependencias")
	}
	return &UseCases{CreateSystem: &CreateSystem{d: d}, UpdateSystem: &UpdateSystem{d: d}}, nil
}

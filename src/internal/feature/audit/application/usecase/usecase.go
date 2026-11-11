// Package usecase contiene los casos de uso de audit.
package usecase

import (
	"errors"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/audit/application/port"
)

// PermissionRead es el permiso para consultar la auditoría.
const PermissionRead = "audit.read"

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Permissions port.Permissions
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	Record      *Record
	ListEntries *ListEntries
	VerifyChain *VerifyChain
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Permissions == nil || d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("audit: faltan dependencias")
	}
	return &UseCases{
		Record:      &Record{d: d},
		ListEntries: &ListEntries{d: d},
		VerifyChain: &VerifyChain{d: d},
	}, nil
}

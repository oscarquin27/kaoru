// Package usecase contiene los casos de uso de comment.
package usecase

import (
	"errors"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/comment/application/port"
)

// PermissionCreate es el permiso para comentar.
const PermissionCreate = "comment.create"

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Entities    port.Entities
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateComment *CreateComment
	ListComments  *ListComments
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Entities == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("comment: faltan dependencias")
	}
	return &UseCases{CreateComment: &CreateComment{d: d}, ListComments: &ListComments{d: d}}, nil
}

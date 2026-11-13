// Package usecase contiene los casos de uso de change.
package usecase

import (
	"errors"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/change/application/port"
	"kaoru/internal/feature/change/domain"
)

// Permisos del feature.
const (
	PermissionCreate = "chg.create"
	PermissionEdit   = "chg.edit"
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Workflow    port.Workflow
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateChange     *CreateChange
	GetChange        *GetChange
	ListChanges      *ListChanges
	ListWindows      *ListWindows
	UpdateChange     *UpdateChange
	TransitionChange *TransitionChange
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Workflow == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("change: faltan dependencias")
	}
	return &UseCases{
		CreateChange:     &CreateChange{d: d},
		GetChange:        &GetChange{d: d},
		ListChanges:      &ListChanges{d: d},
		ListWindows:      &ListWindows{d: d},
		UpdateChange:     &UpdateChange{d: d},
		TransitionChange: &TransitionChange{d: d},
	}, nil
}

// ChangeOutput es un CHG tal como lo devuelven los casos de uso.
type ChangeOutput struct {
	Change        domain.Snapshot
	CreatedByName string
}

func toOutput(v port.ChangeView) ChangeOutput {
	return ChangeOutput{Change: v.Change.Snapshot(), CreatedByName: v.CreatedByName}
}

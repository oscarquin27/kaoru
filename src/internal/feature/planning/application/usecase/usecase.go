// Package usecase contiene los casos de uso de planning.
package usecase

import (
	"context"
	"errors"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionManage es el permiso de mantener los hitos, que tiene change management.
const PermissionManage = "milestone.manage"

// ErrMilestoneNotFound es el error de un hito que no existe.
var ErrMilestoneNotFound = axis.New(axis.NotFoundResource, "El hito no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Changes     port.Changes
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateMilestone *CreateMilestone
	UpdateMilestone *UpdateMilestone
	ListMilestones  *ListMilestones
	GetCalendar     *GetCalendar
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Changes == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("planning: faltan dependencias")
	}
	return &UseCases{
		CreateMilestone: &CreateMilestone{d: d},
		UpdateMilestone: &UpdateMilestone{d: d},
		ListMilestones:  &ListMilestones{d: d},
		GetCalendar:     &GetCalendar{d: d},
	}, nil
}

// MilestoneOutput es un hito con sus días de atraso a la hora de leerlo.
type MilestoneOutput struct {
	Milestone domain.Snapshot
	DaysLate  int
}

func toOutput(m domain.Milestone, now time.Time) MilestoneOutput {
	return MilestoneOutput{Milestone: m.Snapshot(), DaysLate: m.DaysLate(now)}
}

// findOwner lee el responsable; vacío es sin responsable y uno que no existe es un error del campo owner.
func findOwner(ctx context.Context, repo port.Repository, id string) (*domain.UserRef, error) {
	if id == "" {
		return nil, nil
	}
	user, err := repo.FindUser(ctx, shared.UserId(id))
	if axis.IsCode(err, axis.NotFoundResource) {
		return nil, axis.Invalid(axis.InvalidField("owner", "El usuario no existe."))
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

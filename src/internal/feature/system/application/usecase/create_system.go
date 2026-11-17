package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/system/application/port"
	"kaoru/internal/feature/system/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateSystem agrega un sistema bajo prueba al catálogo.
type CreateSystem struct {
	d Deps
}

// CreateSystemInput es la entrada de CreateSystem.
type CreateSystemInput struct {
	Actor        shared.Actor
	Key          string
	Name         string
	DefaultRoute string
}

// Execute exige el permiso admin.configure.
func (uc *CreateSystem) Execute(ctx context.Context, in CreateSystemInput) (domain.Snapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionConfigure); err != nil {
		return domain.Snapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.Snapshot{}, err
	}
	system, err := domain.New(domain.NewParams{
		Id: id, Key: in.Key, Name: in.Name, DefaultRoute: in.DefaultRoute, Now: uc.d.Clock.Now(),
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	s := system.Snapshot()
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		if err := uc.d.Repository.Insert(ctx, system); err != nil {
			if axis.IsCode(err, axis.DuplicateResource) {
				return axis.Invalid(axis.InvalidField("key", "Ya hay un sistema con esa clave."))
			}
			return err
		}
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityId: s.Id, Key: s.Key, Action: port.AuditCreate, NewValue: s.Name,
		})
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return s, nil
}

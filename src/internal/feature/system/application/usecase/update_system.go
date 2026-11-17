package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/system/application/port"
	"kaoru/internal/feature/system/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateSystem cambia el nombre o la ruta por omisión de un sistema y audita cada campo que cambió.
type UpdateSystem struct {
	d Deps
}

// UpdateSystemInput es la entrada de UpdateSystem.
type UpdateSystemInput struct {
	Actor shared.Actor
	Key   string
	Edit  domain.Edit
}

// Execute exige el permiso admin.configure.
func (uc *UpdateSystem) Execute(ctx context.Context, in UpdateSystemInput) (domain.Snapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionConfigure); err != nil {
		return domain.Snapshot{}, err
	}
	var out domain.Snapshot
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		system, err := uc.d.Repository.Lock(ctx, domain.NormalizeKey(in.Key))
		if axis.IsCode(err, axis.NotFoundResource) {
			return ErrSystemNotFound
		}
		if err != nil {
			return err
		}
		changes, err := system.Apply(in.Edit, uc.d.Clock.Now())
		if err != nil {
			return err
		}
		out = system.Snapshot()
		if len(changes) == 0 {
			return nil
		}
		if err := uc.d.Repository.Update(ctx, system); err != nil {
			return err
		}
		for _, c := range changes {
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, EntityId: out.Id, Key: out.Key, Action: port.AuditUpdate,
				Field: c.Field, OldValue: c.Old, NewValue: c.New,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}

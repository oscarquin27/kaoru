package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/environment/application/port"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateEnvironment edita o retira un ambiente y audita cada campo que cambió.
type UpdateEnvironment struct {
	d Deps
}

// UpdateEnvironmentInput es la entrada de UpdateEnvironment.
type UpdateEnvironmentInput struct {
	Actor shared.Actor
	Code  string
	Edit  domain.Edit
}

// Execute exige el permiso env.manage.
func (uc *UpdateEnvironment) Execute(ctx context.Context, in UpdateEnvironmentInput) (EnvironmentOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return EnvironmentOutput{}, err
	}
	var view port.EnvironmentView
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		var err error
		if view, err = lockEnvironment(ctx, uc.d.Repository, in.Code); err != nil {
			return err
		}
		environment := view.Environment
		changes, err := environment.Apply(in.Edit, uc.d.Clock.Now())
		if err != nil || len(changes) == 0 {
			return err
		}
		if err := uc.d.Repository.UpdateEnvironment(ctx, environment); err != nil {
			if axis.IsCode(err, axis.DuplicateResource) {
				return errNameTaken()
			}
			return err
		}
		snapshot := environment.Snapshot()
		for _, c := range changes {
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, EntityType: port.EntityEnvironment, EntityId: snapshot.Id, Code: snapshot.Code,
				Action: port.AuditUpdate, Field: c.Field, OldValue: c.Old, NewValue: c.New,
			}); err != nil {
				return err
			}
		}
		view.Environment = environment
		return nil
	})
	if err != nil {
		return EnvironmentOutput{}, err
	}
	return toOutput(view), nil
}

package usecase

import (
	"context"

	"kaoru/internal/feature/component/application/port"
	"kaoru/internal/feature/component/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateComponent edita, retira o reactiva un COMP y audita cada campo que cambió.
type UpdateComponent struct {
	d Deps
}

// UpdateComponentInput es la entrada de UpdateComponent.
type UpdateComponentInput struct {
	Actor shared.Actor
	Code  string
	Edit  domain.Edit
}

// Execute exige el permiso comp.manage.
func (uc *UpdateComponent) Execute(ctx context.Context, in UpdateComponentInput) (ComponentOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return ComponentOutput{}, err
	}
	var view port.ComponentView
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		var err error
		if view, err = lockComponent(ctx, uc.d.Repository, in.Code); err != nil {
			return err
		}
		component := view.Component
		changes, err := component.Apply(in.Edit, uc.d.Clock.Now())
		if err != nil || len(changes) == 0 {
			return err
		}
		if err := uc.d.Repository.Update(ctx, component); err != nil {
			return err
		}
		snapshot := component.Snapshot()
		for _, c := range changes {
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, ComponentId: snapshot.Id, Code: snapshot.Code, Action: port.AuditUpdate,
				Field: c.Field, OldValue: c.Old, NewValue: c.New,
			}); err != nil {
				return err
			}
		}
		view.Component = component
		return nil
	})
	if err != nil {
		return ComponentOutput{}, err
	}
	return toOutput(view), nil
}

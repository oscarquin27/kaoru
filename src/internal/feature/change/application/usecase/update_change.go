package usecase

import (
	"context"

	"kaoru/internal/feature/change/application/port"
	"kaoru/internal/feature/change/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateChange edita los datos de un CHG y audita cada campo que cambió.
type UpdateChange struct {
	d Deps
}

// UpdateChangeInput es la entrada de UpdateChange.
type UpdateChangeInput struct {
	Actor shared.Actor
	Code  string
	Edit  domain.Edit
}

// Execute exige el permiso chg.edit.
func (uc *UpdateChange) Execute(ctx context.Context, in UpdateChangeInput) (ChangeOutput, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionEdit); err != nil {
		return ChangeOutput{}, err
	}
	var view port.ChangeView
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		var err error
		if view, err = lockChange(ctx, uc.d.Repository, in.Code); err != nil {
			return err
		}
		change := view.Change
		changes, err := change.Apply(in.Edit, uc.d.Clock.Now())
		if err != nil || len(changes) == 0 {
			return err
		}
		if err := uc.d.Repository.Update(ctx, change); err != nil {
			return err
		}
		snapshot := change.Snapshot()
		for _, c := range changes {
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, ChangeId: snapshot.Id, Code: snapshot.Code, Action: port.AuditUpdate,
				Field: c.Field, OldValue: c.Old, NewValue: c.New,
			}); err != nil {
				return err
			}
		}
		view.Change = change
		return nil
	})
	return toOutput(view), err
}

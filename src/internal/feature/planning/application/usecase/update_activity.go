package usecase

import (
	"context"

	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateActivity reajusta una actividad, la cancela o la reactiva, y audita cada campo que cambió.
type UpdateActivity struct {
	d Deps
}

// UpdateActivityInput es la entrada de UpdateActivity. Un puntero nil deja el campo como está;
// Change apuntando a "" deja la actividad sin CHG y Operators reemplaza la lista.
type UpdateActivityInput struct {
	Actor     shared.Actor
	Code      string
	Kind      *string
	Title     *string
	Change    *string
	Start     *string
	End       *string
	Operators *[]string
	Notes     *string
	Cancelled *bool
}

// Execute exige el permiso activity.manage.
func (uc *UpdateActivity) Execute(ctx context.Context, in UpdateActivityInput) (ActivityOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionPlan); err != nil {
		return ActivityOutput{}, err
	}
	now := uc.d.Clock.Now()

	var out ActivityOutput
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		activity, err := lockActivity(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		edit := domain.ActivityEdit{Kind: in.Kind, Title: in.Title, Start: in.Start, End: in.End, Notes: in.Notes, Cancelled: in.Cancelled}
		if in.Change != nil {
			change, err := findChange(ctx, uc.d.Repository, *in.Change)
			if err != nil {
				return err
			}
			edit.Change = domain.ChangeEdit{Set: true, Value: change}
		}
		if in.Operators != nil {
			operators, err := findOperators(ctx, uc.d.Repository, *in.Operators)
			if err != nil {
				return err
			}
			edit.Operators = &operators
		}
		changes, err := activity.Apply(edit, now)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			if err := uc.d.Repository.UpdateActivity(ctx, activity); err != nil {
				return err
			}
			s := activity.Snapshot()
			for _, c := range changes {
				if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
					Actor: in.Actor.Id, Entity: port.EntityActivity, EntityId: s.Id, Code: s.Code, Action: port.AuditUpdate,
					Field: c.Field, OldValue: c.Old, NewValue: c.New,
				}); err != nil {
					return err
				}
			}
		}
		saved, err := uc.d.Repository.FindActivity(ctx, in.Code)
		if err != nil {
			return err
		}
		out, err = withOverlaps(ctx, uc.d.Repository, saved, now)
		return err
	})
	if err != nil {
		return ActivityOutput{}, err
	}
	return out, nil
}

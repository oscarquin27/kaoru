package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateMilestone edita un hito, lo marca cumplido o lo vuelve a dejar pendiente, y audita cada campo que cambió.
type UpdateMilestone struct {
	d Deps
}

// UpdateMilestoneInput es la entrada de UpdateMilestone. Un puntero nil deja el campo como está;
// Owner apuntando a "" deja el hito sin responsable.
type UpdateMilestoneInput struct {
	Actor     shared.Actor
	Id        string
	Title     *string
	DueDate   *string
	Owner     *string
	Notes     *string
	Completed *bool
}

// Execute exige el permiso milestone.manage.
func (uc *UpdateMilestone) Execute(ctx context.Context, in UpdateMilestoneInput) (MilestoneOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return MilestoneOutput{}, err
	}
	now := uc.d.Clock.Now()

	var out MilestoneOutput
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		err := uc.d.Repository.Lock(ctx, in.Id)
		if axis.IsCode(err, axis.NotFoundResource) {
			return ErrMilestoneNotFound
		}
		if err != nil {
			return err
		}
		milestone, err := uc.d.Repository.Find(ctx, in.Id)
		if err != nil {
			return err
		}
		edit := domain.Edit{Title: in.Title, DueDate: in.DueDate, Notes: in.Notes, Completed: in.Completed}
		if in.Owner != nil {
			owner, err := findOwner(ctx, uc.d.Repository, *in.Owner)
			if err != nil {
				return err
			}
			edit.Owner = domain.OwnerEdit{Set: true, Value: owner}
		}
		changes, err := milestone.Apply(edit, now)
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			if err := uc.d.Repository.Update(ctx, milestone); err != nil {
				return err
			}
			for _, c := range changes {
				if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
					Actor: in.Actor.Id, EntityId: in.Id, Action: port.AuditUpdate, Field: c.Field, OldValue: c.Old, NewValue: c.New,
				}); err != nil {
					return err
				}
			}
		}
		saved, err := uc.d.Repository.Find(ctx, in.Id)
		out = toOutput(saved, now)
		return err
	})
	if err != nil {
		return MilestoneOutput{}, err
	}
	return out, nil
}

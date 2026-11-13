package usecase

import (
	"context"

	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateMilestone registra un hito del calendario.
type CreateMilestone struct {
	d Deps
}

// CreateMilestoneInput es la entrada de CreateMilestone. DueDate viene como 2026-11-30; Owner vacío es sin responsable.
type CreateMilestoneInput struct {
	Actor   shared.Actor
	Title   string
	DueDate string
	Owner   string
	Notes   string
}

// Execute exige el permiso milestone.manage.
func (uc *CreateMilestone) Execute(ctx context.Context, in CreateMilestoneInput) (MilestoneOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return MilestoneOutput{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return MilestoneOutput{}, err
	}
	now := uc.d.Clock.Now()

	var out MilestoneOutput
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		owner, err := findOwner(ctx, uc.d.Repository, in.Owner)
		if err != nil {
			return err
		}
		milestone, err := domain.New(domain.NewParams{
			Id: id, Title: in.Title, DueDate: in.DueDate, Notes: in.Notes, Owner: owner, CreatedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.Insert(ctx, milestone); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityId: id, Action: port.AuditCreate, Field: "dueDate",
			NewValue: milestone.Snapshot().DueDate.Format(domain.DateLayout),
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.Find(ctx, id)
		out = toOutput(saved, now)
		return err
	})
	if err != nil {
		return MilestoneOutput{}, err
	}
	return out, nil
}

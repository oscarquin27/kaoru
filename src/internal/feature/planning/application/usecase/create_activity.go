package usecase

import (
	"context"

	"kaoru/internal/feature/planning/application/port"
	"kaoru/internal/feature/planning/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateActivity planifica una actividad. Un solape no impide guardarla: la salida lo trae.
type CreateActivity struct {
	d Deps
}

// CreateActivityInput es la entrada de CreateActivity. Las fechas vienen como 2026-11-30; Change vacío es sin CHG.
type CreateActivityInput struct {
	Actor     shared.Actor
	Kind      string
	Title     string
	Change    string
	Start     string
	End       string
	Operators []string
	Notes     string
}

// Execute exige el permiso activity.manage.
func (uc *CreateActivity) Execute(ctx context.Context, in CreateActivityInput) (ActivityOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionPlan); err != nil {
		return ActivityOutput{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return ActivityOutput{}, err
	}
	now := uc.d.Clock.Now()

	var out ActivityOutput
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		change, err := findChange(ctx, uc.d.Repository, in.Change)
		if err != nil {
			return err
		}
		operators, err := findOperators(ctx, uc.d.Repository, in.Operators)
		if err != nil {
			return err
		}
		code, err := uc.d.Repository.NextActivityCode(ctx, now.Year())
		if err != nil {
			return err
		}
		activity, err := domain.NewActivity(domain.NewActivityParams{
			Id: id, Code: code.String(), Kind: in.Kind, Title: in.Title, Change: change, Start: in.Start, End: in.End,
			Notes: in.Notes, Operators: operators, CreatedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertActivity(ctx, activity); err != nil {
			return err
		}
		s := activity.Snapshot()
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, Entity: port.EntityActivity, EntityId: id, Code: s.Code, Action: port.AuditCreate,
			Field: "dates", NewValue: s.Start.Format(domain.DateLayout) + " a " + s.End.Format(domain.DateLayout),
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindActivity(ctx, s.Code)
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

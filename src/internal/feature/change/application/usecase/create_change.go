package usecase

import (
	"context"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/change/application/port"
	"kaoru/internal/feature/change/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateChange registra un CHG en el estado inicial de su flujo.
type CreateChange struct {
	d Deps
}

// CreateChangeInput es la entrada de CreateChange. Route vacío toma la ruta del sistema.
type CreateChangeInput struct {
	Actor       shared.Actor
	Title       string
	Description string
	SystemKey   string
	Route       string
	WindowStart *time.Time
	WindowEnd   *time.Time
}

// Execute exige el permiso chg.create.
func (uc *CreateChange) Execute(ctx context.Context, in CreateChangeInput) (ChangeOutput, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionCreate); err != nil {
		return ChangeOutput{}, err
	}
	system, err := uc.d.Repository.FindSystem(ctx, in.SystemKey)
	if axis.IsCode(err, axis.NotFoundResource) {
		field := axis.InvalidField("system", "El sistema no existe.")
		if in.SystemKey == "" {
			field = axis.Required("system", "Elige un sistema.")
		}
		return ChangeOutput{}, axis.Invalid(field)
	}
	if err != nil {
		return ChangeOutput{}, err
	}
	state, err := uc.d.Workflow.InitialState(ctx)
	if err != nil {
		return ChangeOutput{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return ChangeOutput{}, err
	}
	now := uc.d.Clock.Now()

	var view port.ChangeView
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, now.Year())
		if err != nil {
			return err
		}
		change, err := domain.New(domain.NewParams{
			Id: id, Code: code, Title: in.Title, Description: in.Description, System: system, Route: in.Route,
			WindowStart: in.WindowStart, WindowEnd: in.WindowEnd, State: state, CreatedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.Insert(ctx, change); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, ChangeId: id, Code: code, Action: port.AuditCreate, NewValue: change.Snapshot().Title,
		}); err != nil {
			return err
		}
		view, err = uc.d.Repository.FindByCode(ctx, code)
		return err
	})
	return toOutput(view), err
}

package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/component/application/port"
	"kaoru/internal/feature/component/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateComponent agrega un COMP al catálogo de un sistema.
type CreateComponent struct {
	d Deps
}

// CreateComponentInput es la entrada de CreateComponent.
type CreateComponentInput struct {
	Actor       shared.Actor
	SystemKey   string
	Kind        string
	Name        string
	Description string
}

// Execute exige el permiso comp.manage.
func (uc *CreateComponent) Execute(ctx context.Context, in CreateComponentInput) (ComponentOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return ComponentOutput{}, err
	}
	system, err := uc.d.Repository.FindSystem(ctx, in.SystemKey)
	if axis.IsCode(err, axis.NotFoundResource) {
		field := axis.InvalidField("system", "El sistema no existe.")
		if in.SystemKey == "" {
			field = axis.Required("system", "Elige un sistema.")
		}
		return ComponentOutput{}, axis.Invalid(field)
	}
	if err != nil {
		return ComponentOutput{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return ComponentOutput{}, err
	}
	now := uc.d.Clock.Now()

	var view port.ComponentView
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, now.Year())
		if err != nil {
			return err
		}
		component, err := domain.New(domain.NewParams{
			Id: id, Code: code, System: system, Kind: in.Kind, Name: in.Name, Description: in.Description,
			CreatedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.Insert(ctx, component); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, ComponentId: id, Code: code, Action: port.AuditCreate, NewValue: component.Snapshot().Name,
		}); err != nil {
			return err
		}
		view, err = uc.d.Repository.FindByCode(ctx, code)
		return err
	})
	if err != nil {
		return ComponentOutput{}, err
	}
	return toOutput(view), nil
}

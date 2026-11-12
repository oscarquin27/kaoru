package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/environment/application/port"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateEnvironment registra un ambiente de un sistema.
type CreateEnvironment struct {
	d Deps
}

// CreateEnvironmentInput es la entrada de CreateEnvironment.
type CreateEnvironmentInput struct {
	Actor     shared.Actor
	SystemKey string
	Name      string
	Kind      string
	OS        string
	Notes     string
}

// Execute exige el permiso env.manage.
func (uc *CreateEnvironment) Execute(ctx context.Context, in CreateEnvironmentInput) (EnvironmentOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionManage); err != nil {
		return EnvironmentOutput{}, err
	}
	system, err := uc.d.Repository.FindSystem(ctx, in.SystemKey)
	if axis.IsCode(err, axis.NotFoundResource) {
		field := axis.InvalidField("system", "El sistema no existe.")
		if in.SystemKey == "" {
			field = axis.Required("system", "Elige un sistema.")
		}
		return EnvironmentOutput{}, axis.Invalid(field)
	}
	if err != nil {
		return EnvironmentOutput{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return EnvironmentOutput{}, err
	}
	now := uc.d.Clock.Now()

	var view port.EnvironmentView
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixEnvironment, now.Year())
		if err != nil {
			return err
		}
		environment, err := domain.New(domain.NewParams{
			Id: id, Code: code, System: system, Name: in.Name, Kind: in.Kind, OS: in.OS, Notes: in.Notes,
			CreatedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertEnvironment(ctx, environment); err != nil {
			if axis.IsCode(err, axis.DuplicateResource) {
				return errNameTaken()
			}
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityEnvironment, EntityId: id, Code: code,
			Action: port.AuditCreate, NewValue: environment.Snapshot().Name,
		}); err != nil {
			return err
		}
		view, err = uc.d.Repository.FindEnvironment(ctx, code)
		return err
	})
	if err != nil {
		return EnvironmentOutput{}, err
	}
	return toOutput(view), nil
}

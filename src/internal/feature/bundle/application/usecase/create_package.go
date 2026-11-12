package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// CreatePackage crea un PKG vacío en el estado inicial de su flujo: «En armado».
type CreatePackage struct {
	d Deps
}

// CreatePackageInput es la entrada de CreatePackage.
type CreatePackageInput struct {
	Actor       shared.Actor
	Title       string
	Description string
	SystemKey   string
}

// Execute exige el permiso pkg.build.
func (uc *CreatePackage) Execute(ctx context.Context, in CreatePackageInput) (PackageOutput, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionBuild); err != nil {
		return PackageOutput{}, err
	}
	system, err := uc.d.Repository.FindSystem(ctx, in.SystemKey)
	if axis.IsCode(err, axis.NotFoundResource) {
		field := axis.InvalidField("system", "El sistema no existe.")
		if in.SystemKey == "" {
			field = axis.Required("system", "Elige un sistema.")
		}
		return PackageOutput{}, axis.Invalid(field)
	}
	if err != nil {
		return PackageOutput{}, err
	}
	state, err := uc.d.Workflow.InitialState(ctx)
	if err != nil {
		return PackageOutput{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return PackageOutput{}, err
	}
	now := uc.d.Clock.Now()

	var out PackageOutput
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, now.Year())
		if err != nil {
			return err
		}
		pkg, err := domain.New(domain.NewParams{
			Id: id, Code: code, Title: in.Title, Description: in.Description, System: system, State: state,
			CreatedBy: domain.UserRef{Id: in.Actor.Id}, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.Insert(ctx, pkg); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, PackageId: id, Code: code, Action: port.AuditCreate, NewValue: pkg.Snapshot().Title,
		}); err != nil {
			return err
		}
		out, err = readPackage(ctx, uc.d, code.String())
		return err
	})
	if err != nil {
		return PackageOutput{}, err
	}
	return out, nil
}

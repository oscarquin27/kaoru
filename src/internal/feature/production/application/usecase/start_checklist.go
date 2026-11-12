package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/production/application/port"
	"kaoru/internal/feature/production/domain"
	shared "kaoru/internal/shared/domain"
)

// StartChecklist abre el checklist de producción de un PKG con la última versión de la plantilla de su sistema.
type StartChecklist struct {
	d Deps
}

// StartChecklistInput es la entrada de StartChecklist.
type StartChecklistInput struct {
	Actor           shared.Actor
	PackageCode     string
	EnvironmentCode string
}

// Execute exige el permiso chk.start. Devuelve INVALID_ACTION si el sistema no tiene plantilla.
func (uc *StartChecklist) Execute(ctx context.Context, in StartChecklistInput) (ChecklistOutput, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionStart); err != nil {
		return ChecklistOutput{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return ChecklistOutput{}, err
	}
	now := uc.d.Clock.Now()

	var out ChecklistOutput
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		pkg, environment, err := uc.findTargets(ctx, in)
		if err != nil {
			return err
		}
		template, err := uc.d.Repository.LatestTemplate(ctx, pkg.SystemKey)
		if axis.IsCode(err, axis.NotFoundResource) {
			return ErrNoTemplate
		}
		if err != nil {
			return err
		}
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixChecklist, now.Year())
		if err != nil {
			return err
		}
		checklist, err := domain.NewChecklist(domain.NewChecklistParams{
			Id: id, Code: code, Package: pkg, Environment: environment, Template: template.Snapshot(),
			StartedBy: in.Actor.Id, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertChecklist(ctx, checklist); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityChecklist, EntityId: id, Code: code.String(),
			Action: port.AuditCreate, Field: "package", NewValue: pkg.Code,
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindChecklist(ctx, code)
		out = toChecklistOutput(saved)
		return err
	})
	if err != nil {
		return ChecklistOutput{}, err
	}
	return out, nil
}

// findTargets lee el PKG y el ambiente; los que faltan o no existen salen juntos, cada uno en su campo.
func (uc *StartChecklist) findTargets(ctx context.Context, in StartChecklistInput) (domain.PackageRef, domain.EnvironmentRef, error) {
	var errs axis.FieldErrors
	pkg, err := uc.d.Repository.FindPackage(ctx, in.PackageCode)
	switch {
	case axis.IsCode(err, axis.NotFoundResource) && in.PackageCode == "":
		errs.Add(axis.Required("package", "Indica el PKG."))
	case axis.IsCode(err, axis.NotFoundResource):
		errs.Add(axis.InvalidField("package", "El PKG no existe."))
	case err != nil:
		return domain.PackageRef{}, domain.EnvironmentRef{}, err
	}
	environment, err := uc.d.Repository.FindEnvironment(ctx, in.EnvironmentCode)
	switch {
	case axis.IsCode(err, axis.NotFoundResource) && in.EnvironmentCode == "":
		errs.Add(axis.Required("environment", "Indica el ambiente de producción."))
	case axis.IsCode(err, axis.NotFoundResource):
		errs.Add(axis.InvalidField("environment", "El ambiente no existe."))
	case err != nil:
		return domain.PackageRef{}, domain.EnvironmentRef{}, err
	}
	return pkg, environment, errs.Err()
}

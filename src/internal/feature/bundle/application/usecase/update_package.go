package usecase

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdatePackage edita el título o la descripción de un PKG y audita cada campo que cambió.
type UpdatePackage struct {
	d Deps
}

// UpdatePackageInput es la entrada de UpdatePackage.
type UpdatePackageInput struct {
	Actor shared.Actor
	Code  string
	Edit  domain.Edit
}

// Execute exige el permiso pkg.build.
func (uc *UpdatePackage) Execute(ctx context.Context, in UpdatePackageInput) (PackageOutput, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionBuild); err != nil {
		return PackageOutput{}, err
	}
	var out PackageOutput
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		pkg, err := lockPackage(ctx, uc.d.Repository, in.Code)
		if err != nil {
			return err
		}
		changes, err := pkg.Apply(in.Edit, uc.d.Clock.Now())
		if err != nil {
			return err
		}
		if len(changes) > 0 {
			if err := uc.d.Repository.Update(ctx, pkg); err != nil {
				return err
			}
			s := pkg.Snapshot()
			for _, c := range changes {
				if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
					Actor: in.Actor.Id, PackageId: s.Id, Code: s.Code, Action: port.AuditUpdate,
					Field: c.Field, OldValue: c.Old, NewValue: c.New,
				}); err != nil {
					return err
				}
			}
		}
		out, err = readPackage(ctx, uc.d, in.Code)
		return err
	})
	if err != nil {
		return PackageOutput{}, err
	}
	return out, nil
}

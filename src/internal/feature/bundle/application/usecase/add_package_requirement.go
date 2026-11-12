package usecase

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// AddPackageRequirement agrega un REQ a un PKG en armado. Si el REQ está «Aprobado QA»,
// la misma acción lo pasa a «En paquete».
type AddPackageRequirement struct {
	d Deps
}

// AddPackageRequirementInput es la entrada de AddPackageRequirement.
type AddPackageRequirementInput struct {
	Actor           shared.Actor
	Code            string
	RequirementCode string
}

// Execute exige el permiso pkg.build y, si el REQ está «Aprobado QA», el de pasarlo a «En paquete»:
// sin él no se agrega. Devuelve NOT_FOUND_RESOURCE si el PKG o el REQ no existen e INVALID_ACTION si
// el PKG no está en armado, el REQ es de otro sistema o no está «Aprobado QA» ni «En paquete».
// Agregar un REQ que ya está no cambia nada.
func (uc *AddPackageRequirement) Execute(ctx context.Context, in AddPackageRequirementInput) (PackageOutput, error) {
	return changeContent(ctx, uc.d, in.Actor, in.Code, func(ctx context.Context, pkg *domain.Package) (port.AuditRecord, bool, error) {
		requirement, err := findRequirement(ctx, uc.d, in.RequirementCode)
		if err != nil {
			return port.AuditRecord{}, false, err
		}
		now := uc.d.Clock.Now()
		added, err := pkg.AddRequirement(requirement, now)
		if err != nil || !added {
			return port.AuditRecord{}, false, err
		}
		if requirement.Ref.State.Key == domain.RequirementApprovedQA {
			if err := uc.d.Requirements.MoveToPackage(ctx, in.Actor, requirement.Ref.Code); err != nil {
				return port.AuditRecord{}, false, err
			}
		}
		if err := uc.d.Repository.AddRequirement(ctx, pkg.Snapshot().Id, requirement.Ref.Id, in.Actor.Id, now); err != nil {
			return port.AuditRecord{}, false, err
		}
		return port.AuditRecord{Field: fieldRequirements, NewValue: requirement.Ref.Code}, true, nil
	})
}

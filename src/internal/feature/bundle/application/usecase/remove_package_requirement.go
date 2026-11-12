package usecase

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// RemovePackageRequirement quita un REQ de un PKG en armado. El estado del REQ no cambia.
type RemovePackageRequirement struct {
	d Deps
}

// RemovePackageRequirementInput es la entrada de RemovePackageRequirement.
type RemovePackageRequirementInput struct {
	Actor           shared.Actor
	Code            string
	RequirementCode string
}

// Execute exige el permiso pkg.build. Devuelve NOT_FOUND_RESOURCE si el PKG o el REQ no existen e
// INVALID_ACTION si el PKG no está en armado. Quitar un REQ que no está no cambia nada.
func (uc *RemovePackageRequirement) Execute(ctx context.Context, in RemovePackageRequirementInput) (PackageOutput, error) {
	return changeContent(ctx, uc.d, in.Actor, in.Code, func(ctx context.Context, pkg *domain.Package) (port.AuditRecord, bool, error) {
		requirement, err := findRequirement(ctx, uc.d, in.RequirementCode)
		if err != nil {
			return port.AuditRecord{}, false, err
		}
		removed, err := pkg.RemoveRequirement(requirement.Ref.Code, uc.d.Clock.Now())
		if err != nil || !removed {
			return port.AuditRecord{}, false, err
		}
		if err := uc.d.Repository.RemoveRequirement(ctx, pkg.Snapshot().Id, requirement.Ref.Id); err != nil {
			return port.AuditRecord{}, false, err
		}
		return port.AuditRecord{Field: fieldRequirements, OldValue: requirement.Ref.Code}, true, nil
	})
}

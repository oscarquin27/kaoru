package usecase

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// RemovePackageBuild quita un BUILD de un PKG en armado.
type RemovePackageBuild struct {
	d Deps
}

// RemovePackageBuildInput es la entrada de RemovePackageBuild.
type RemovePackageBuildInput struct {
	Actor     shared.Actor
	Code      string
	BuildCode string
}

// Execute exige el permiso pkg.build. Devuelve NOT_FOUND_RESOURCE si el PKG o el BUILD no existen e
// INVALID_ACTION si el PKG no está en armado. Quitar un BUILD que no está no cambia nada.
func (uc *RemovePackageBuild) Execute(ctx context.Context, in RemovePackageBuildInput) (PackageOutput, error) {
	return changeContent(ctx, uc.d, in.Actor, in.Code, func(ctx context.Context, pkg *domain.Package) (port.AuditRecord, bool, error) {
		build, err := findBuild(ctx, uc.d, in.BuildCode)
		if err != nil {
			return port.AuditRecord{}, false, err
		}
		removed, err := pkg.RemoveBuild(build.Ref.Code, uc.d.Clock.Now())
		if err != nil || !removed {
			return port.AuditRecord{}, false, err
		}
		if err := uc.d.Repository.RemoveBuild(ctx, pkg.Snapshot().Id, build.Ref.Id); err != nil {
			return port.AuditRecord{}, false, err
		}
		return port.AuditRecord{Field: fieldBuilds, OldValue: build.Ref.Code}, true, nil
	})
}

package usecase

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	shared "kaoru/internal/shared/domain"
)

// AddPackageBuild agrega un BUILD del catálogo a un PKG en armado.
type AddPackageBuild struct {
	d Deps
}

// AddPackageBuildInput es la entrada de AddPackageBuild.
type AddPackageBuildInput struct {
	Actor     shared.Actor
	Code      string
	BuildCode string
}

// Execute exige el permiso pkg.build. Devuelve NOT_FOUND_RESOURCE si el PKG o el BUILD no existen e
// INVALID_ACTION si el PKG no está en armado o el BUILD es de otro sistema. Agregar un BUILD que ya
// está no cambia nada.
func (uc *AddPackageBuild) Execute(ctx context.Context, in AddPackageBuildInput) (PackageOutput, error) {
	return changeContent(ctx, uc.d, in.Actor, in.Code, func(ctx context.Context, pkg *domain.Package) (port.AuditRecord, bool, error) {
		build, err := findBuild(ctx, uc.d, in.BuildCode)
		if err != nil {
			return port.AuditRecord{}, false, err
		}
		now := uc.d.Clock.Now()
		added, err := pkg.AddBuild(build, now)
		if err != nil || !added {
			return port.AuditRecord{}, false, err
		}
		if err := uc.d.Repository.AddBuild(ctx, pkg.Snapshot().Id, build.Ref.Id, in.Actor.Id, now); err != nil {
			return port.AuditRecord{}, false, err
		}
		return port.AuditRecord{Field: fieldBuilds, NewValue: build.Ref.Code}, true, nil
	})
}

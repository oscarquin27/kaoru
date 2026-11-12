package usecase

import (
	"context"
	"time"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/environment/application/port"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// RegisterDeployment registra el despliegue de un BUILD o de un PKG en un ambiente.
// Quién lo registra y cuándo salen de la sesión y del servidor. Después no se edita ni se borra.
type RegisterDeployment struct {
	d Deps
}

// RegisterDeploymentInput es la entrada de RegisterDeployment. Lleva BuildCode o PackageCode, uno de los dos.
type RegisterDeploymentInput struct {
	Actor           shared.Actor
	EnvironmentCode string
	BuildCode       string
	PackageCode     string
	Result          string
	Notes           string
	ExecutedAt      time.Time
}

// Execute exige dep.create_production si el ambiente es de producción y dep.create en los demás.
func (uc *RegisterDeployment) Execute(ctx context.Context, in RegisterDeploymentInput) (domain.DeploymentSnapshot, error) {
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.DeploymentSnapshot{}, err
	}
	now := uc.d.Clock.Now()

	var out domain.DeploymentSnapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		// El ambiente se bloquea: no se retira ni cambia de tipo mientras se registra el despliegue.
		view, envErr := lockEnvironment(ctx, uc.d.Repository, in.EnvironmentCode)
		if envErr != nil && !axis.IsCode(envErr, axis.NotFoundResource) {
			return envErr
		}
		environment := view.Environment.Snapshot()
		permission := PermissionDeploy
		if envErr == nil && environment.Kind == domain.KindProduction {
			permission = PermissionDeployProduction
		}
		if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, permission); err != nil {
			return err
		}
		if envErr != nil {
			field := axis.InvalidField("environment", "El ambiente no existe.")
			if in.EnvironmentCode == "" {
				field = axis.Required("environment", "Indica el ambiente.")
			}
			return axis.Invalid(field)
		}
		build, err := uc.findBuild(ctx, in.BuildCode)
		if err != nil {
			return err
		}
		pkg, err := uc.findPackage(ctx, in.PackageCode)
		if err != nil {
			return err
		}
		code, err := uc.d.Repository.NextCode(ctx, shared.PrefixDeployment, now.Year())
		if err != nil {
			return err
		}
		deployment, err := domain.NewDeployment(domain.NewDeploymentParams{
			Id: id, Code: code, Environment: environment, Build: build, Package: pkg, Result: in.Result, Notes: in.Notes,
			ExecutedBy: in.Actor.Id, ExecutedAt: in.ExecutedAt, Now: now,
		})
		if err != nil {
			return err
		}
		if err := uc.d.Repository.InsertDeployment(ctx, deployment); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, EntityType: port.EntityDeployment, EntityId: id, Code: code,
			Action: port.AuditCreate, Field: "result", NewValue: string(deployment.Snapshot().Result),
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindDeployment(ctx, code)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.DeploymentSnapshot{}, err
	}
	return out, nil
}

// findBuild devuelve nil si no se indicó un BUILD; uno que no existe es un error del campo.
func (uc *RegisterDeployment) findBuild(ctx context.Context, code string) (*domain.BuildRef, error) {
	if code == "" {
		return nil, nil
	}
	build, err := uc.d.Builds.Find(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return nil, axis.Invalid(axis.InvalidField("build", "El BUILD no existe."))
	}
	if err != nil {
		return nil, err
	}
	return &build, nil
}

// findPackage devuelve nil si no se indicó un PKG; uno que no existe es un error del campo.
func (uc *RegisterDeployment) findPackage(ctx context.Context, code string) (*domain.PackageRef, error) {
	if code == "" {
		return nil, nil
	}
	pkg, err := uc.d.Repository.FindPackage(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return nil, axis.Invalid(axis.InvalidField("package", "El PKG no existe."))
	}
	if err != nil {
		return nil, err
	}
	return &pkg, nil
}

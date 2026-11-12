// Package usecase contiene los casos de uso de environment.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/environment/application/port"
	"kaoru/internal/feature/environment/domain"
	shared "kaoru/internal/shared/domain"
)

// Permisos del feature. Desplegar en producción es de change management (regla de negocio 6).
const (
	PermissionManage           = "env.manage"
	PermissionDeploy           = "dep.create"
	PermissionDeployProduction = "dep.create_production"
)

// Errores de los casos de uso.
var (
	ErrEnvironmentNotFound = axis.New(axis.NotFoundResource, "El ambiente no existe.")
	ErrDeploymentNotFound  = axis.New(axis.NotFoundResource, "El despliegue no existe.")
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository  port.Repository
	Builds      port.Builds
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateEnvironment       *CreateEnvironment
	GetEnvironment          *GetEnvironment
	ListEnvironments        *ListEnvironments
	UpdateEnvironment       *UpdateEnvironment
	FindEnvironment         *FindEnvironment
	RegisterDeployment      *RegisterDeployment
	GetDeployment           *GetDeployment
	ListDeployments         *ListDeployments
	HasProductionDeployment *HasProductionDeployment
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Builds == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("environment: faltan dependencias")
	}
	return &UseCases{
		CreateEnvironment:       &CreateEnvironment{d: d},
		GetEnvironment:          &GetEnvironment{d: d},
		ListEnvironments:        &ListEnvironments{d: d},
		UpdateEnvironment:       &UpdateEnvironment{d: d},
		FindEnvironment:         &FindEnvironment{d: d},
		RegisterDeployment:      &RegisterDeployment{d: d},
		GetDeployment:           &GetDeployment{d: d},
		ListDeployments:         &ListDeployments{d: d},
		HasProductionDeployment: &HasProductionDeployment{d: d},
	}, nil
}

// EnvironmentOutput es un ENV tal como lo devuelven los casos de uso.
type EnvironmentOutput struct {
	Environment domain.Snapshot
	// Installed es nil si el ambiente nunca tuvo un despliegue exitoso.
	Installed *domain.Installed
}

func toOutput(v port.EnvironmentView) EnvironmentOutput {
	return EnvironmentOutput{Environment: v.Environment.Snapshot(), Installed: v.Installed}
}

// errNameTaken es el error de un nombre que ya usa otro ambiente del mismo sistema.
func errNameTaken() error {
	return axis.Invalid(axis.InvalidField("name", "Ya hay un ambiente con ese nombre en el sistema."))
}

// findEnvironment valida el código y lee el ENV; un código mal formado tampoco existe.
func findEnvironment(ctx context.Context, repo port.Repository, raw string) (port.EnvironmentView, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return port.EnvironmentView{}, ErrEnvironmentNotFound
	}
	view, err := repo.FindEnvironment(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return port.EnvironmentView{}, ErrEnvironmentNotFound
	}
	return view, err
}

// lockEnvironment bloquea el ENV y lo lee: se usa dentro de la transacción de todo caso de uso que
// escribe sobre un ambiente, para que una edición y un despliegue simultáneos no se pisen.
func lockEnvironment(ctx context.Context, repo port.Repository, raw string) (port.EnvironmentView, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return port.EnvironmentView{}, ErrEnvironmentNotFound
	}
	if err := repo.LockEnvironment(ctx, code); err != nil {
		if axis.IsCode(err, axis.NotFoundResource) {
			return port.EnvironmentView{}, ErrEnvironmentNotFound
		}
		return port.EnvironmentView{}, err
	}
	return findEnvironment(ctx, repo, raw)
}

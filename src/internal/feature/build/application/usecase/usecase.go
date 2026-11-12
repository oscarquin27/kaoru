// Package usecase contiene los casos de uso de build.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/build/application/port"
	"kaoru/internal/feature/build/domain"
	shared "kaoru/internal/shared/domain"
)

// PermissionRegister es el permiso para registrar un BUILD.
const PermissionRegister = "build.register"

// ErrBuildNotFound es el error de un BUILD que no existe.
var ErrBuildNotFound = axis.New(axis.NotFoundResource, "El BUILD no existe.")

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository   port.Repository
	Requirements port.Requirements
	Permissions  port.Permissions
	AuditTrail   port.AuditTrail
	UnitOfWork   uow.Factory
	Clock        clock.Clock
	IdGenerator  idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	RegisterBuild *RegisterBuild
	GetBuild      *GetBuild
	ListBuilds    *ListBuilds
	FindBuild     *FindBuild
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.Requirements == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("build: faltan dependencias")
	}
	return &UseCases{
		RegisterBuild: &RegisterBuild{d: d},
		GetBuild:      &GetBuild{d: d},
		ListBuilds:    &ListBuilds{d: d},
		FindBuild:     &FindBuild{d: d},
	}, nil
}

// findBuild valida el código y lee el BUILD; un código mal formado tampoco existe.
func findBuild(ctx context.Context, repo port.Repository, raw string) (domain.Build, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Build{}, ErrBuildNotFound
	}
	build, err := repo.FindByCode(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Build{}, ErrBuildNotFound
	}
	return build, err
}

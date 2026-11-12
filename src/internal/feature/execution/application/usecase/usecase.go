// Package usecase contiene los casos de uso de execution.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/idgen"
	"kaoru/internal/core/pkg/uow"
	"kaoru/internal/feature/execution/application/port"
	"kaoru/internal/feature/execution/domain"
	shared "kaoru/internal/shared/domain"
)

// Permisos del feature.
const (
	PermissionCreateRun         = "run.create"
	PermissionCreateExploration = "exp.create"
)

// Errores de los casos de uso.
var (
	ErrRunNotFound         = axis.New(axis.NotFoundResource, "El RUN no existe.")
	ErrExplorationNotFound = axis.New(axis.NotFoundResource, "El EXP no existe.")
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository   port.Repository
	TestDesign   port.TestDesign
	Builds       port.Builds
	Environments port.Environments
	Permissions  port.Permissions
	AuditTrail   port.AuditTrail
	UnitOfWork   uow.Factory
	Clock        clock.Clock
	IdGenerator  idgen.IdGenerator
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CreateRun         *CreateRun
	GetRun            *GetRun
	ListRuns          *ListRuns
	GetCoverage       *GetCoverage
	CreateExploration *CreateExploration
	GetExploration    *GetExploration
	ListExplorations  *ListExplorations
	UpdateExploration *UpdateExploration
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.TestDesign == nil || d.Builds == nil || d.Environments == nil ||
		d.Permissions == nil || d.AuditTrail == nil || d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
		return nil, errors.New("execution: faltan dependencias")
	}
	return &UseCases{
		CreateRun:         &CreateRun{d: d},
		GetRun:            &GetRun{d: d},
		ListRuns:          &ListRuns{d: d},
		GetCoverage:       &GetCoverage{d: d},
		CreateExploration: &CreateExploration{d: d},
		GetExploration:    &GetExploration{d: d},
		ListExplorations:  &ListExplorations{d: d},
		UpdateExploration: &UpdateExploration{d: d},
	}, nil
}

// findCatalog busca el BUILD y el ENV sobre los que se registra un RUN o un EXP.
// Uno que falta o no existe es un error de su campo, que se suma a errs.
func findCatalog(ctx context.Context, d Deps, buildCode, environmentCode string, errs *axis.FieldErrors) (domain.BuildRef, domain.EnvironmentRef, error) {
	build, err := d.Builds.Find(ctx, buildCode)
	switch {
	case buildCode == "":
		errs.Add(axis.Required("build", "Elige el BUILD sobre el que se ejecutó."))
	case axis.IsCode(err, axis.NotFoundResource):
		errs.Add(axis.InvalidField("build", "El BUILD no existe."))
	case err != nil:
		return domain.BuildRef{}, domain.EnvironmentRef{}, err
	}
	environment, err := d.Environments.Find(ctx, environmentCode)
	switch {
	case environmentCode == "":
		errs.Add(axis.Required("environment", "Elige el ambiente en el que se ejecutó."))
	case axis.IsCode(err, axis.NotFoundResource):
		errs.Add(axis.InvalidField("environment", "El ambiente no existe."))
	case err != nil:
		return domain.BuildRef{}, domain.EnvironmentRef{}, err
	}
	return build, environment, nil
}

func findExploration(ctx context.Context, repo port.Repository, raw string) (domain.Exploration, error) {
	code, err := shared.ParseCode(raw)
	if err != nil {
		return domain.Exploration{}, ErrExplorationNotFound
	}
	exploration, err := repo.FindExploration(ctx, code)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Exploration{}, ErrExplorationNotFound
	}
	return exploration, err
}

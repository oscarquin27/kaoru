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
	Repository  port.Repository
	TestDesign  port.TestDesign
	Permissions port.Permissions
	AuditTrail  port.AuditTrail
	UnitOfWork  uow.Factory
	Clock       clock.Clock
	IdGenerator idgen.IdGenerator
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
	if d.Repository == nil || d.TestDesign == nil || d.Permissions == nil || d.AuditTrail == nil ||
		d.UnitOfWork == nil || d.Clock == nil || d.IdGenerator == nil {
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

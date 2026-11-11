// Package usecase contiene los casos de uso de workflow.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository port.Repository
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CheckPermission   *CheckPermission
	ListPermissions   *ListPermissions
	GetWorkflow       *GetWorkflow
	GetInitialState   *GetInitialState
	ResolveTransition *ResolveTransition
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil {
		return nil, errors.New("workflow: falta Repository")
	}
	return &UseCases{
		CheckPermission:   &CheckPermission{repo: d.Repository},
		ListPermissions:   &ListPermissions{repo: d.Repository},
		GetWorkflow:       &GetWorkflow{repo: d.Repository},
		GetInitialState:   &GetInitialState{repo: d.Repository},
		ResolveTransition: &ResolveTransition{repo: d.Repository},
	}, nil
}

// authorize comprueba que alguno de los roles tenga el permiso.
func authorize(ctx context.Context, repo port.Repository, roles []string, permission string) error {
	grants, err := repo.Grants(ctx, roles)
	if err != nil {
		return err
	}
	if !domain.Allows(grants, permission) {
		return domain.ErrPermissionDenied
	}
	return nil
}

// loadWorkflow valida el tipo de entidad y lee su flujo.
func loadWorkflow(ctx context.Context, repo port.Repository, entityType string) (domain.Workflow, error) {
	t, err := domain.ParseEntityType(entityType)
	if err != nil {
		return domain.Workflow{}, err
	}
	return repo.Workflow(ctx, t)
}

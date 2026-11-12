// Package usecase contiene los casos de uso de workflow.
package usecase

import (
	"context"
	"errors"

	"kaoru/internal/core/pkg/clock"
	"kaoru/internal/core/pkg/uow"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
)

// Deps son las dependencias de los casos de uso.
type Deps struct {
	Repository port.Repository
	// AuditTrail, UnitOfWork y Clock los usan los casos de uso que cambian la configuración.
	AuditTrail port.AuditTrail
	UnitOfWork uow.Factory
	Clock      clock.Clock
}

// UseCases reúne los casos de uso del feature.
type UseCases struct {
	CheckPermission   *CheckPermission
	ListPermissions   *ListPermissions
	GetWorkflow       *GetWorkflow
	GetInitialState   *GetInitialState
	ResolveTransition *ResolveTransition

	ListRolePermissions *ListRolePermissions
	SetRolePermissions  *SetRolePermissions
	UpdateBlockingRule  *UpdateBlockingRule

	AddState         *AddState
	UpdateState      *UpdateState
	RemoveState      *RemoveState
	SetTransition    *SetTransition
	RemoveTransition *RemoveTransition
}

// New arma los casos de uso; falla si falta una dependencia.
func New(d Deps) (*UseCases, error) {
	if d.Repository == nil || d.AuditTrail == nil || d.UnitOfWork == nil || d.Clock == nil {
		return nil, errors.New("workflow: faltan dependencias")
	}
	return &UseCases{
		CheckPermission:   &CheckPermission{repo: d.Repository},
		ListPermissions:   &ListPermissions{repo: d.Repository},
		GetWorkflow:       &GetWorkflow{repo: d.Repository},
		GetInitialState:   &GetInitialState{repo: d.Repository},
		ResolveTransition: &ResolveTransition{repo: d.Repository},

		ListRolePermissions: &ListRolePermissions{d: d},
		SetRolePermissions:  &SetRolePermissions{d: d},
		UpdateBlockingRule:  &UpdateBlockingRule{d: d},

		AddState:         &AddState{d: d},
		UpdateState:      &UpdateState{d: d},
		RemoveState:      &RemoveState{d: d},
		SetTransition:    &SetTransition{d: d},
		RemoveTransition: &RemoveTransition{d: d},
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

// RolePermissions es la asignación de permisos de un rol.
type RolePermissions struct {
	Role        string
	Permissions []string
}

// RolePermissionsOutput son los permisos que existen y cómo están asignados.
type RolePermissionsOutput struct {
	Permissions []domain.Permission
	Roles       []RolePermissions
}

// loadRolePermissions lee los permisos y su asignación, con todos los roles aunque no tengan ninguno.
func loadRolePermissions(ctx context.Context, repo port.Repository) (RolePermissionsOutput, error) {
	permissions, err := repo.Permissions(ctx)
	if err != nil {
		return RolePermissionsOutput{}, err
	}
	grants, err := repo.AllGrants(ctx)
	if err != nil {
		return RolePermissionsOutput{}, err
	}
	out := RolePermissionsOutput{Permissions: permissions, Roles: make([]RolePermissions, len(domain.Roles))}
	for i, role := range domain.Roles {
		out.Roles[i] = RolePermissions{Role: role, Permissions: []string{}}
		for _, g := range grants {
			if g.Role == role {
				out.Roles[i].Permissions = append(out.Roles[i].Permissions, g.Permission)
			}
		}
	}
	return out, nil
}

package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// ListRolePermissions devuelve los permisos que existen y su asignación por rol.
type ListRolePermissions struct {
	d Deps
}

// ListRolePermissionsInput es la entrada de ListRolePermissions.
type ListRolePermissionsInput struct {
	Actor shared.Actor
}

// Execute exige el permiso admin.configure.
func (uc *ListRolePermissions) Execute(ctx context.Context, in ListRolePermissionsInput) (RolePermissionsOutput, error) {
	if err := authorize(ctx, uc.d.Repository, in.Actor.Roles, domain.PermissionConfigure); err != nil {
		return RolePermissionsOutput{}, err
	}
	return loadRolePermissions(ctx, uc.d.Repository)
}

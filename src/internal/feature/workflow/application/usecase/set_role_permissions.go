package usecase

import (
	"context"
	"slices"
	"strings"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// SetRolePermissions reemplaza los permisos de un rol. Así se resuelve, sin tocar código,
// quién puede hacer cada acción (por ejemplo, quién construye el PKG).
type SetRolePermissions struct {
	d Deps
}

// SetRolePermissionsInput es la entrada de SetRolePermissions.
type SetRolePermissionsInput struct {
	Actor       shared.Actor
	Role        string
	Permissions []string
}

// Execute exige el permiso admin.configure.
func (uc *SetRolePermissions) Execute(ctx context.Context, in SetRolePermissionsInput) (RolePermissionsOutput, error) {
	repo := uc.d.Repository
	if err := authorize(ctx, repo, in.Actor.Roles, domain.PermissionConfigure); err != nil {
		return RolePermissionsOutput{}, err
	}
	known, err := repo.Permissions(ctx)
	if err != nil {
		return RolePermissionsOutput{}, err
	}
	permissions, err := domain.ValidateRolePermissions(in.Role, in.Permissions, known)
	if err != nil {
		return RolePermissionsOutput{}, err
	}
	var out RolePermissionsOutput
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		grants, err := repo.Grants(ctx, []string{in.Role})
		if err != nil {
			return err
		}
		current := make([]string, len(grants))
		for i, g := range grants {
			current[i] = g.Permission
		}
		slices.Sort(current)
		if !slices.Equal(current, permissions) {
			if err := repo.ReplaceRoleGrants(ctx, in.Role, permissions); err != nil {
				return err
			}
			if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
				Actor: in.Actor.Id, Entity: "role_permission", Key: in.Role, Field: "permissions",
				OldValue: strings.Join(current, ","), NewValue: strings.Join(permissions, ","),
			}); err != nil {
				return err
			}
		}
		out, err = loadRolePermissions(ctx, repo)
		return err
	})
	if err != nil {
		return RolePermissionsOutput{}, err
	}
	return out, nil
}

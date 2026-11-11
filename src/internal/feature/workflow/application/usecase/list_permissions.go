package usecase

import (
	"context"
	"slices"

	"kaoru/internal/feature/workflow/application/port"
)

// ListPermissions devuelve los permisos que suman los roles de un actor.
type ListPermissions struct {
	repo port.Repository
}

// ListPermissionsInput es la entrada de ListPermissions.
type ListPermissionsInput struct {
	Roles []string
}

// ListPermissionsOutput trae los roles que Kaoru reconoce y sus permisos, sin repetir.
type ListPermissionsOutput struct {
	Roles       []string
	Permissions []string
}

// Execute lee los permisos. Un rol del token sin permisos en Kaoru no aparece en Roles.
func (uc *ListPermissions) Execute(ctx context.Context, in ListPermissionsInput) (ListPermissionsOutput, error) {
	grants, err := uc.repo.Grants(ctx, in.Roles)
	if err != nil {
		return ListPermissionsOutput{}, err
	}
	out := ListPermissionsOutput{Roles: []string{}, Permissions: []string{}}
	for _, g := range grants {
		if !slices.Contains(out.Roles, g.Role) {
			out.Roles = append(out.Roles, g.Role)
		}
		if !slices.Contains(out.Permissions, g.Permission) {
			out.Permissions = append(out.Permissions, g.Permission)
		}
	}
	slices.Sort(out.Permissions)
	return out, nil
}

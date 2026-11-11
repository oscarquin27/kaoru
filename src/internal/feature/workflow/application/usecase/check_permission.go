package usecase

import (
	"context"

	"kaoru/internal/feature/workflow/application/port"
)

// CheckPermission responde si los roles de un actor tienen un permiso.
type CheckPermission struct {
	repo port.Repository
}

// CheckPermissionInput es la entrada de CheckPermission.
type CheckPermissionInput struct {
	Roles      []string
	Permission string
}

// Execute devuelve PERMISSION_DENIED si ningún rol tiene el permiso.
func (uc *CheckPermission) Execute(ctx context.Context, in CheckPermissionInput) error {
	return authorize(ctx, uc.repo, in.Roles, in.Permission)
}

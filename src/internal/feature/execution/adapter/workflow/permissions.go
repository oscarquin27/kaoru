// Package workflow implementa el puerto Permissions con los casos de uso de workflow.
package workflow

import (
	"context"

	"kaoru/internal/feature/execution/application/port"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Permissions comprueba permisos con el feature workflow.
type Permissions struct {
	check *workflowuc.CheckPermission
}

var _ port.Permissions = (*Permissions)(nil)

// NewPermissions crea el adaptador.
func NewPermissions(check *workflowuc.CheckPermission) *Permissions {
	return &Permissions{check: check}
}

// Require devuelve PERMISSION_DENIED si ningún rol tiene el permiso.
func (p *Permissions) Require(ctx context.Context, roles []string, permission string) error {
	return p.check.Execute(ctx, workflowuc.CheckPermissionInput{Roles: roles, Permission: permission})
}

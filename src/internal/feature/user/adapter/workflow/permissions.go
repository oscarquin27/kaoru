// Package workflow implementa el puerto Permissions con los casos de uso de workflow.
package workflow

import (
	"context"

	"kaoru/internal/feature/user/application/port"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
)

// Permissions lee permisos con el feature workflow.
type Permissions struct {
	list *workflowuc.ListPermissions
}

var _ port.Permissions = (*Permissions)(nil)

// NewPermissions crea el adaptador.
func NewPermissions(list *workflowuc.ListPermissions) *Permissions {
	return &Permissions{list: list}
}

// Granted devuelve los roles que Kaoru reconoce y los permisos que suman.
func (p *Permissions) Granted(ctx context.Context, roles []string) ([]string, []string, error) {
	out, err := p.list.Execute(ctx, workflowuc.ListPermissionsInput{Roles: roles})
	if err != nil {
		return nil, nil, err
	}
	return out.Roles, out.Permissions, nil
}

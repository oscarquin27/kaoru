// Package workflow implementa el puerto Workflow con los casos de uso de workflow.
package workflow

import (
	"context"

	"kaoru/internal/feature/change/application/port"
	"kaoru/internal/feature/change/domain"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
	workflowdomain "kaoru/internal/feature/workflow/domain"
)

const entityType = string(workflowdomain.EntityChange)

// Workflow consulta permisos y estados del flujo del CHG.
type Workflow struct {
	uc *workflowuc.UseCases
}

var _ port.Workflow = (*Workflow)(nil)

// New crea el adaptador.
func New(uc *workflowuc.UseCases) *Workflow { return &Workflow{uc: uc} }

// Require devuelve PERMISSION_DENIED si ningún rol tiene el permiso.
func (w *Workflow) Require(ctx context.Context, roles []string, permission string) error {
	return w.uc.CheckPermission.Execute(ctx, workflowuc.CheckPermissionInput{Roles: roles, Permission: permission})
}

// InitialState devuelve el estado en el que nace un CHG.
func (w *Workflow) InitialState(ctx context.Context) (domain.State, error) {
	s, err := w.uc.GetInitialState.Execute(ctx, workflowuc.GetInitialStateInput{EntityType: entityType})
	if err != nil {
		return domain.State{}, err
	}
	return domain.State{Id: s.Id, Key: s.Key, Name: s.Name}, nil
}

// ResolveTransition conserva el código de error de workflow.
func (w *Workflow) ResolveTransition(ctx context.Context, from, to string, roles []string) (port.Transition, error) {
	out, err := w.uc.ResolveTransition.Execute(ctx, workflowuc.ResolveTransitionInput{
		EntityType: entityType, From: from, To: to, Roles: roles,
	})
	if err != nil {
		return port.Transition{}, err
	}
	return port.Transition{
		To:             domain.State{Id: out.To.Id, Key: out.To.Key, Name: out.To.Name},
		RequiresReason: out.RequiresReason,
	}, nil
}

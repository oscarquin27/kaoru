// Package workflow implementa el puerto Workflow con los casos de uso de workflow.
package workflow

import (
	"context"

	"kaoru/internal/feature/defect/application/port"
	"kaoru/internal/feature/defect/domain"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
	workflowdomain "kaoru/internal/feature/workflow/domain"
)

const entityType = string(workflowdomain.EntityDefect)

// Workflow consulta permisos, estados y reglas del flujo del DEF.
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

// InitialState devuelve el estado en el que nace un DEF.
func (w *Workflow) InitialState(ctx context.Context) (domain.State, error) {
	s, err := w.uc.GetInitialState.Execute(ctx, workflowuc.GetInitialStateInput{EntityType: entityType})
	if err != nil {
		return domain.State{}, err
	}
	return toState(s), nil
}

// ResolveTransition conserva el código de error de workflow.
func (w *Workflow) ResolveTransition(ctx context.Context, from, to string, roles []string) (port.Transition, error) {
	out, err := w.uc.ResolveTransition.Execute(ctx, workflowuc.ResolveTransitionInput{
		EntityType: entityType, From: from, To: to, Roles: roles,
	})
	if err != nil {
		return port.Transition{}, err
	}
	rules := make([]domain.Rule, len(out.Rules))
	for i, r := range out.Rules {
		rules[i] = domain.Rule{Key: r.Key, Params: r.Params, Blocks: r.Blocks}
	}
	return port.Transition{To: toState(out.To), RequiresReason: out.RequiresReason, Rules: rules}, nil
}

func toState(s workflowdomain.State) domain.State {
	return domain.State{Id: s.Id, Key: s.Key, Name: s.Name, IsFinal: s.IsFinal}
}

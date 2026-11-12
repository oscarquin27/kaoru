// Package workflow implementa el puerto Workflow con los casos de uso de workflow.
package workflow

import (
	"context"

	"kaoru/internal/feature/bundle/application/port"
	"kaoru/internal/feature/bundle/domain"
	workflowuc "kaoru/internal/feature/workflow/application/usecase"
	workflowdomain "kaoru/internal/feature/workflow/domain"
)

const entityType = string(workflowdomain.EntityPackage)

// Workflow consulta permisos, estados y reglas del flujo del PKG.
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

// InitialState devuelve el estado en el que nace un PKG.
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
	return port.Transition{To: toState(out.To), RequiresReason: out.RequiresReason, Rules: toRules(out.Rules)}, nil
}

// Outgoing devuelve las transiciones que salen de un estado, con sus reglas activas.
func (w *Workflow) Outgoing(ctx context.Context, from string) ([]port.Transition, error) {
	wf, err := w.uc.GetWorkflow.Execute(ctx, workflowuc.GetWorkflowInput{EntityType: entityType})
	if err != nil {
		return nil, err
	}
	var out []port.Transition
	for _, t := range wf.Transitions {
		if t.From != from {
			continue
		}
		to, err := wf.State(t.To)
		if err != nil {
			return nil, err
		}
		var enabled []workflowdomain.Rule
		for _, r := range t.Rules {
			if r.Enabled {
				enabled = append(enabled, r)
			}
		}
		out = append(out, port.Transition{To: toState(to), RequiresReason: t.RequiresReason, Rules: toRules(enabled)})
	}
	return out, nil
}

func toState(s workflowdomain.State) domain.State {
	return domain.State{Id: s.Id, Key: s.Key, Name: s.Name, IsInitial: s.IsInitial}
}

func toRules(rules []workflowdomain.Rule) []domain.Rule {
	out := make([]domain.Rule, len(rules))
	for i, r := range rules {
		out[i] = domain.Rule{Key: r.Key, Params: r.Params, Blocks: r.Blocks}
	}
	return out
}

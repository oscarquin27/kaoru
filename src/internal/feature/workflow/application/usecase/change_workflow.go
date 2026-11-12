package usecase

import (
	"context"
	"encoding/json"

	"kaoru/internal/feature/workflow/application/port"
	"kaoru/internal/feature/workflow/domain"
	shared "kaoru/internal/shared/domain"
)

// Entidades de auditoría de la configuración de un flujo.
const (
	auditState      = "workflow_state"
	auditTransition = "workflow_transition"
)

// changeWorkflow es el esqueleto de todo cambio a un flujo: exige admin.configure, lee el flujo
// dentro de la transacción, aplica el cambio, lo audita y devuelve el flujo como quedó.
// change devuelve un registro sin Field cuando no cambió nada.
func changeWorkflow(
	ctx context.Context, d Deps, actor shared.Actor, entityType string,
	change func(ctx context.Context, wf domain.Workflow) (port.AuditRecord, error),
) (domain.Workflow, error) {
	if err := authorize(ctx, d.Repository, actor.Roles, domain.PermissionConfigure); err != nil {
		return domain.Workflow{}, err
	}
	t, err := domain.ParseEntityType(entityType)
	if err != nil {
		return domain.Workflow{}, err
	}
	var out domain.Workflow
	err = d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		wf, err := d.Repository.Workflow(ctx, t)
		if err != nil {
			return err
		}
		record, err := change(ctx, wf)
		if err != nil {
			return err
		}
		if record.Field != "" {
			record.Actor = actor.Id
			if err := d.AuditTrail.Record(ctx, record); err != nil {
				return err
			}
		}
		out, err = d.Repository.Workflow(ctx, t)
		return err
	})
	if err != nil {
		return domain.Workflow{}, err
	}
	return out, nil
}

// describeState resume un estado en una línea, para comparar y para la auditoría.
func describeState(s domain.State) string {
	raw, _ := json.Marshal(struct {
		Name     string `json:"name"`
		Position *int   `json:"position"`
		IsFinal  bool   `json:"isFinal"`
	}{s.Name, s.Position, s.IsFinal})
	return string(raw)
}

// describeTransition resume una transición en una línea, para comparar y para la auditoría.
func describeTransition(t domain.Transition) string {
	raw, _ := json.Marshal(struct {
		Permission     string `json:"permission"`
		RequiresReason bool   `json:"requiresReason"`
	}{t.Permission, t.RequiresReason})
	return string(raw)
}

func stateAuditKey(wf domain.Workflow, key string) string { return string(wf.EntityType) + ":" + key }

func transitionAuditKey(wf domain.Workflow, from, to string) string {
	return string(wf.EntityType) + ":" + from + ">" + to
}

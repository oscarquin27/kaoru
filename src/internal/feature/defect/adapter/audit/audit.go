// Package audit implementa los puertos AuditTrail y Timeline con los casos de uso de audit.
package audit

import (
	"context"
	"time"

	audituc "kaoru/internal/feature/audit/application/usecase"
	auditdomain "kaoru/internal/feature/audit/domain"
	"kaoru/internal/feature/defect/application/port"
)

const entityType = "defect"

// Trail registra los hechos de un DEF en la auditoría y lee de ella cuándo cambió de estado.
type Trail struct {
	record      *audituc.Record
	transitions *audituc.GetLastTransitions
}

var (
	_ port.AuditTrail = (*Trail)(nil)
	_ port.Timeline   = (*Trail)(nil)
)

// New crea el adaptador.
func New(uc *audituc.UseCases) *Trail {
	return &Trail{record: uc.Record, transitions: uc.GetLastTransitions}
}

// Record escribe en la transacción del context; conserva el código de error de audit.
func (t *Trail) Record(ctx context.Context, r port.AuditRecord) error {
	return t.record.Execute(ctx, audituc.RecordInput{
		ActorId: r.Actor, EntityType: entityType, EntityId: r.DefectId, EntityCode: r.Code.String(),
		Action: auditdomain.Action(r.Action), Field: r.Field, OldValue: r.OldValue, NewValue: r.NewValue, Reason: r.Reason,
	})
}

// LastTransitions devuelve cuándo entró cada DEF a su estado actual.
func (t *Trail) LastTransitions(ctx context.Context, defectIds []string) (map[string]time.Time, error) {
	return t.transitions.Execute(ctx, audituc.GetLastTransitionsInput{EntityIds: defectIds})
}

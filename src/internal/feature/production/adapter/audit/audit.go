// Package audit implementa el puerto AuditTrail con el caso de uso Record de audit.
package audit

import (
	"context"

	audituc "kaoru/internal/feature/audit/application/usecase"
	auditdomain "kaoru/internal/feature/audit/domain"
	"kaoru/internal/feature/production/application/port"
)

// Trail registra los hechos de plantillas, CHK, DIAG y SMOKE en la auditoría.
type Trail struct {
	record *audituc.Record
}

var _ port.AuditTrail = (*Trail)(nil)

// New crea el adaptador.
func New(record *audituc.Record) *Trail { return &Trail{record: record} }

// Record escribe en la transacción del context; conserva el código de error de audit.
func (t *Trail) Record(ctx context.Context, r port.AuditRecord) error {
	return t.record.Execute(ctx, audituc.RecordInput{
		ActorId: r.Actor, EntityType: r.EntityType, EntityId: r.EntityId, EntityCode: r.Code,
		Action: auditdomain.Action(r.Action), Field: r.Field, NewValue: r.NewValue, Reason: r.Reason,
	})
}

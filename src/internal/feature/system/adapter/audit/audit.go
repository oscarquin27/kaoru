// Package audit implementa el puerto AuditTrail con el caso de uso Record de audit.
package audit

import (
	"context"

	audituc "kaoru/internal/feature/audit/application/usecase"
	auditdomain "kaoru/internal/feature/audit/domain"
	"kaoru/internal/feature/system/application/port"
)

// Trail registra los cambios del catálogo de sistemas en la auditoría.
type Trail struct {
	record *audituc.Record
}

var _ port.AuditTrail = (*Trail)(nil)

// New crea el adaptador.
func New(record *audituc.Record) *Trail { return &Trail{record: record} }

// Record escribe en la transacción del context; conserva el código de error de audit.
func (t *Trail) Record(ctx context.Context, r port.AuditRecord) error {
	return t.record.Execute(ctx, audituc.RecordInput{
		ActorId: r.Actor, EntityType: port.EntitySystem, EntityId: r.EntityId, EntityCode: r.Key,
		Action: auditdomain.Action(r.Action), Field: r.Field, OldValue: r.OldValue, NewValue: r.NewValue,
	})
}

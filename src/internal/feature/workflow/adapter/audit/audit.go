// Package audit implementa el puerto AuditTrail de workflow con el caso de uso Record de audit.
// audit depende de workflow para los permisos, así que este adaptador nace sin conectar y
// bootstrap lo conecta cuando los dos módulos existen (reglas §A.7, decisión D30).
package audit

import (
	"context"

	"github.com/google/uuid"

	"kaoru/internal/core/pkg/axis"
	audituc "kaoru/internal/feature/audit/application/usecase"
	auditdomain "kaoru/internal/feature/audit/domain"
	"kaoru/internal/feature/workflow/application/port"
)

// Trail registra los cambios de configuración en la auditoría.
type Trail struct {
	record *audituc.Record
}

var _ port.AuditTrail = (*Trail)(nil)

// New crea el adaptador sin conectar.
func New() *Trail { return &Trail{} }

// Connect conecta el feature audit. Se llama una sola vez, al arrancar.
func (t *Trail) Connect(record *audituc.Record) { t.record = record }

// Record escribe en la transacción del context. Sin auditoría conectada, falla:
// un cambio de configuración nunca se guarda sin dejar rastro.
func (t *Trail) Record(ctx context.Context, r port.AuditRecord) error {
	if t.record == nil {
		return axis.New(axis.InternalServerError, "workflow: auditoría sin conectar")
	}
	entityID := r.Key
	if _, err := uuid.Parse(r.Key); err != nil {
		// Un rol no tiene llave propia: se le deriva una estable de su nombre.
		entityID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(r.Entity+":"+r.Key)).String()
	}
	return t.record.Execute(ctx, audituc.RecordInput{
		ActorId: r.Actor, EntityType: r.Entity, EntityId: entityID, EntityCode: r.Key,
		Action: auditdomain.ActionUpdate, Field: r.Field, OldValue: r.OldValue, NewValue: r.NewValue,
	})
}

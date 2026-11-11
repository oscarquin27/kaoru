// Package audit implementa el puerto AuditTrail con el caso de uso Record de audit.
package audit

import (
	"context"

	audituc "kaoru/internal/feature/audit/application/usecase"
	auditdomain "kaoru/internal/feature/audit/domain"
	"kaoru/internal/feature/comment/application/port"
	"kaoru/internal/feature/comment/domain"
)

// Trail registra los comentarios en la auditoría de la entidad comentada.
type Trail struct {
	record *audituc.Record
}

var _ port.AuditTrail = (*Trail)(nil)

// New crea el adaptador.
func New(record *audituc.Record) *Trail { return &Trail{record: record} }

// RecordComment deja el comentario en la historia de la entidad, con su texto.
func (t *Trail) RecordComment(ctx context.Context, c domain.Comment) error {
	return t.record.Execute(ctx, audituc.RecordInput{
		ActorId: c.AuthorId, EntityType: c.Entity.Type, EntityId: c.Entity.Id, EntityCode: c.Entity.Code,
		Action: auditdomain.ActionCreate, Field: "comment", NewValue: c.Body,
	})
}

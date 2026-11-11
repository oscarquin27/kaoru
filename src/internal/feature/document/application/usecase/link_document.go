package usecase

import (
	"context"

	"kaoru/internal/feature/document/application/port"
	"kaoru/internal/feature/document/domain"
	shared "kaoru/internal/shared/domain"
)

// LinkDocument enlaza un documento existente a otra entidad.
type LinkDocument struct {
	d Deps
}

// LinkDocumentInput es la entrada de LinkDocument.
type LinkDocumentInput struct {
	Actor      shared.Actor
	Code       string
	EntityType string
	EntityCode string
}

// Execute exige el permiso doc.upload. Enlazar dos veces a lo mismo no cambia nada.
func (uc *LinkDocument) Execute(ctx context.Context, in LinkDocumentInput) (domain.Snapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionUpload); err != nil {
		return domain.Snapshot{}, err
	}
	document, err := findDocument(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return domain.Snapshot{}, err
	}
	entity, err := resolveEntity(ctx, uc.d, in.EntityType, in.EntityCode)
	if err != nil {
		return domain.Snapshot{}, err
	}
	s := document.Snapshot()
	if document.IsLinkedTo(entity) {
		return s, nil
	}
	var out domain.Snapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		if err := uc.d.Repository.AddLink(ctx, s.Id, entity, in.Actor.Id, uc.d.Clock.Now()); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, DocumentId: s.Id, Code: s.Code, Action: port.AuditUpdate,
			Field: "link", NewValue: entity.Code,
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindByCode(ctx, s.Code)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}

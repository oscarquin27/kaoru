package usecase

import (
	"context"

	"kaoru/internal/feature/document/application/port"
	"kaoru/internal/feature/document/domain"
	shared "kaoru/internal/shared/domain"
)

// CreateDocument adjunta un documento a una entidad. El servidor valida el tipo real del
// contenido y calcula su SHA-256.
type CreateDocument struct {
	d Deps
}

// CreateDocumentInput es la entrada de CreateDocument.
type CreateDocumentInput struct {
	Actor      shared.Actor
	Title      string
	Kind       string
	EntityType string
	EntityCode string
	File       FileInput
}

// Execute exige el permiso doc.upload.
func (uc *CreateDocument) Execute(ctx context.Context, in CreateDocumentInput) (domain.Snapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionUpload); err != nil {
		return domain.Snapshot{}, err
	}
	entity, err := resolveEntity(ctx, uc.d, in.EntityType, in.EntityCode)
	if err != nil {
		return domain.Snapshot{}, err
	}
	version, err := storeFile(ctx, uc.d, in.File, in.Actor.Id)
	if err != nil {
		return domain.Snapshot{}, err
	}
	id, err := uc.d.IdGenerator.NewId()
	if err != nil {
		return domain.Snapshot{}, err
	}
	now := uc.d.Clock.Now()
	var out domain.Snapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		code, err := uc.d.Repository.NextCode(ctx, now.Year())
		if err != nil {
			return err
		}
		document, err := domain.New(id, code, in.Title, in.Kind, version, entity, now)
		if err != nil {
			return err
		}
		if err := uc.d.Repository.Insert(ctx, document, in.Actor.Id); err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, DocumentId: id, Code: code, Action: port.AuditCreate,
			Field: "checksum", NewValue: version.Checksum,
		}); err != nil {
			return err
		}
		saved, err := uc.d.Repository.FindByCode(ctx, code)
		out = saved.Snapshot()
		return err
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	return out, nil
}

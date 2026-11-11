package usecase

import (
	"context"
	"strconv"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/document/application/port"
	"kaoru/internal/feature/document/domain"
	shared "kaoru/internal/shared/domain"
)

// AddVersion sube una versión nueva de un documento. Las anteriores se conservan.
type AddVersion struct {
	d Deps
}

// AddVersionInput es la entrada de AddVersion.
type AddVersionInput struct {
	Actor shared.Actor
	Code  string
	File  FileInput
}

// Execute exige el permiso doc.upload.
func (uc *AddVersion) Execute(ctx context.Context, in AddVersionInput) (domain.Snapshot, error) {
	if err := uc.d.Permissions.Require(ctx, in.Actor.Roles, PermissionUpload); err != nil {
		return domain.Snapshot{}, err
	}
	document, err := findDocument(ctx, uc.d.Repository, in.Code)
	if err != nil {
		return domain.Snapshot{}, err
	}
	version, err := storeFile(ctx, uc.d, in.File, in.Actor.Id)
	if err != nil {
		return domain.Snapshot{}, err
	}
	s := document.Snapshot()
	var out domain.Snapshot
	err = uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		number, err := uc.d.Repository.AddVersion(ctx, s.Code, version)
		if axis.IsCode(err, axis.NotFoundResource) {
			return ErrDocumentNotFound
		}
		if err != nil {
			return err
		}
		if err := uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, DocumentId: s.Id, Code: s.Code, Action: port.AuditUpdate,
			Field: "version." + strconv.Itoa(number), NewValue: version.Checksum,
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

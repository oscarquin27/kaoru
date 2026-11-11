package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// UpdateLink cambia la etiqueta, la marca de «documento oficial» o la instantánea de un enlace.
type UpdateLink struct {
	d Deps
}

// UpdateLinkInput es la entrada de UpdateLink. SnapshotSet dice si se toca la instantánea;
// SnapshotCode vacío la quita.
type UpdateLinkInput struct {
	Actor        shared.Actor
	Code         string
	LinkId       string
	Label        *string
	IsOfficial   *bool
	SnapshotSet  bool
	SnapshotCode string
}

// Execute exige el permiso link.manage.
func (uc *UpdateLink) Execute(ctx context.Context, in UpdateLinkInput) (domain.LinkSnapshot, error) {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionManageLinks); err != nil {
		return domain.LinkSnapshot{}, err
	}
	edit := domain.LinkEdit{Label: in.Label, IsOfficial: in.IsOfficial}
	if in.SnapshotSet {
		edit.Snapshot = domain.SnapshotEdit{Set: true, Code: in.SnapshotCode}
		if in.SnapshotCode != "" {
			documentId, err := uc.d.Repository.FindDocumentId(ctx, in.SnapshotCode)
			if axis.IsCode(err, axis.NotFoundResource) {
				return domain.LinkSnapshot{}, axis.Invalid(axis.InvalidField("snapshotDocument", "El documento no existe."))
			}
			if err != nil {
				return domain.LinkSnapshot{}, err
			}
			edit.Snapshot.DocumentId = documentId
		}
	}
	var out domain.LinkSnapshot
	err := uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		requirement, link, err := findLink(ctx, uc.d.Repository, in.Code, in.LinkId)
		if err != nil {
			return err
		}
		changes, err := link.Apply(edit, uc.d.Clock.Now())
		out = link.Snapshot()
		if err != nil || len(changes) == 0 {
			return err
		}
		if err := uc.d.Repository.UpdateLink(ctx, link); err != nil {
			return err
		}
		return recordChanges(ctx, uc.d, in.Actor.Id, requirement.Snapshot(), changes)
	})
	if err != nil {
		return domain.LinkSnapshot{}, err
	}
	return out, nil
}

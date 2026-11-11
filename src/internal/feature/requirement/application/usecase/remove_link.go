package usecase

import (
	"context"

	"kaoru/internal/core/pkg/axis"
	"kaoru/internal/feature/requirement/application/port"
	"kaoru/internal/feature/requirement/domain"
	shared "kaoru/internal/shared/domain"
)

// RemoveLink quita un enlace. El hecho queda en la auditoría con su URL.
type RemoveLink struct {
	d Deps
}

// RemoveLinkInput es la entrada de RemoveLink.
type RemoveLinkInput struct {
	Actor  shared.Actor
	Code   string
	LinkId string
}

// Execute exige el permiso link.manage.
func (uc *RemoveLink) Execute(ctx context.Context, in RemoveLinkInput) error {
	if err := uc.d.Workflow.Require(ctx, in.Actor.Roles, PermissionManageLinks); err != nil {
		return err
	}
	return uc.d.UnitOfWork.Within(ctx, func(ctx context.Context) error {
		requirement, link, err := findLink(ctx, uc.d.Repository, in.Code, in.LinkId)
		if err != nil {
			return err
		}
		if err := uc.d.Repository.DeleteLink(ctx, link.Snapshot().Id); err != nil {
			return err
		}
		s := requirement.Snapshot()
		return uc.d.AuditTrail.Record(ctx, port.AuditRecord{
			Actor: in.Actor.Id, RequirementId: s.Id, Code: s.Code, Action: port.AuditDelete,
			Field: "link", OldValue: link.Snapshot().URL,
		})
	})
}

// findLink lee el REQ y uno de sus enlaces.
func findLink(ctx context.Context, repo port.Repository, code, linkId string) (domain.Requirement, domain.Link, error) {
	requirement, err := lockRequirement(ctx, repo, code)
	if err != nil {
		return domain.Requirement{}, domain.Link{}, err
	}
	link, err := repo.FindLink(ctx, requirement.Snapshot().Id, linkId)
	if axis.IsCode(err, axis.NotFoundResource) {
		return domain.Requirement{}, domain.Link{}, ErrLinkNotFound
	}
	return requirement, link, err
}
